package main

import (
	"errors"
	"io"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// TODO(szpakas): move to config
	ingressQueueSize = 1024 * 100
)

var (
	// ErrIngressQueueFull is returned when ingress queue for samples is full.
	// Sample is not queued in such case.
	// Optional retries should be handled on caller side.
	ErrIngressQueueFull = errors.New("collector: ingress queue is full")
)

type collector struct {
	startTime time.Time

	// ingress holds incoming samples for processing
	ingressCh chan *sample

	// sampleParser parses samples represented in transport (text) format and converts it to samples
	sampleParser func(r io.Reader) ([]sample, error)

	counters map[string]*UpdatingCounter
	// countersMu protects scraping functions from interfering with processing
	countersMu sync.RWMutex

	gauges   map[string]*UpdatingGauge
	gaugesMu sync.RWMutex

	histograms   map[string]*UpdatingHistogram
	histogramsMu sync.RWMutex

	testHookProcessSampleDone func()

	// quitCh is used to signal shutdown request
	quitCh chan struct{}

	// shutdownDownCh is used to signal when shutdown is done
	shutdownDownCh  chan struct{}
	shutdownTimeout time.Duration

	metricAppStart           prometheus.Gauge
	metricAppDuration        prometheus.Gauge
	metricQueueLength        prometheus.Gauge
	metricProcessingDuration *prometheus.SummaryVec
	metricExpiringDuration   *prometheus.SummaryVec

	// expiryTime defines the duration for expiring metrics.
	expiryTime time.Duration
}

func newCollector(et time.Duration) *collector {
	return &collector{
		ingressCh:                 make(chan *sample, ingressQueueSize),
		counters:                  make(map[string]*UpdatingCounter),
		gauges:                    make(map[string]*UpdatingGauge),
		histograms:                make(map[string]*UpdatingHistogram),
		testHookProcessSampleDone: func() {},
		quitCh:          make(chan struct{}),
		shutdownDownCh:  make(chan struct{}),
		shutdownTimeout: time.Second,
		expiryTime:      et,

		metricAppStart: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "app_start_timestamp_seconds",
				Help: "Unix timestamp of the app collector start.",
			},
		),
		metricAppDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "app_duration_seconds",
				Help: "Time in seconds since start of the app.",
			},
		),

		metricQueueLength: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "app_collector_queue_length",
				Help: "Number of elements waiting in collector queue for processing.",
			},
		),

		metricProcessingDuration: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "app_collector_processing_duration_ns",
				Help: "Duration of the processing in the collector in ns.",
			},
			[]string{"sampleKind"},
		),

		metricExpiringDuration: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: "app_collector_expiring_duration_ns",
				Help: "Duration of the expiring in the collector in ns.",
			},
			[]string{"sampleKind"},
		),
	}
}

// Collect implements prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.metricAppStart.Collect(ch)

	c.metricAppDuration.Set(time.Now().Sub(c.startTime).Seconds())
	c.metricAppDuration.Collect(ch)

	c.metricQueueLength.Collect(ch)
	c.metricProcessingDuration.Collect(ch)
	c.metricExpiringDuration.Collect(ch)

	c.countersMu.RLock()
	for _, m := range c.counters {
		m.Counter.Collect(ch)
	}
	c.countersMu.RUnlock()

	c.gaugesMu.RLock()
	for _, m := range c.gauges {
		m.Gauge.Collect(ch)
	}
	c.gaugesMu.RUnlock()

	c.histogramsMu.RLock()
	for _, m := range c.histograms {
		m.Histogram.Collect(ch)
	}
	c.histogramsMu.RUnlock()
}

// Describe implements prometheus.Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.metricAppStart.Describe(ch)
	c.metricAppDuration.Describe(ch)
	c.metricQueueLength.Describe(ch)
	c.metricProcessingDuration.Describe(ch)
	c.metricExpiringDuration.Describe(ch)
}

func (c *collector) start() {
	c.startTime = time.Now()

	c.metricAppStart.Set(float64(c.startTime.UnixNano()) / 1e9)

	go c.process()
	go c.processExpiring()
}

func (c *collector) stop() error {
	close(c.quitCh)
	runtime.Gosched()

	select {
	case <-c.shutdownDownCh:
	case <-time.After(c.shutdownTimeout):
		return errors.New("collector: shutdown timed out")
	}

	return nil
}

// Write adds samples to internal queue for processing.
// Will result in ErrIngressQueueFull error if queue is full. The sample is not added to queue in such case.
func (c *collector) Write(s *sample) error {
	select {
	case c.ingressCh <- s:
	default:
		return ErrIngressQueueFull
	}
	return nil
}

// process is responsible from converting samples to metrics and persisting in storage (in-memory)
// Function is run in a separate goroutine. There is always single instance of this function running.
func (c *collector) process() {
	var (
		s  *sample
		h  []byte
		tS time.Time
	)
	for {
		select {
		case s = <-c.ingressCh:
			tS = time.Now()
			c.metricQueueLength.Set(float64(len(c.ingressCh)))

			h = s.hash()

			switch s.kind {
			case sampleCounter:
				// race avoidance is not needed on existence check as "process" is the only one modifying storage
				m, found := c.counters[string(h)]
				if !found {
					m = NewUpdatingCounter(
						prometheus.NewCounter(
							prometheus.CounterOpts{
								Name:        s.name,
								Help:        "auto",
								ConstLabels: s.labels,
							},
						),
					)
					c.countersMu.Lock()
					c.counters[string(h)] = m
					c.countersMu.Unlock()
				}

				m.Counter.Add(s.value)
				m.Touch()

			case sampleGauge:
				m, found := c.gauges[string(h)]
				if !found {
					m = NewUpdatingGauge(
						prometheus.NewGauge(
							prometheus.GaugeOpts{
								Name:        s.name,
								Help:        "auto",
								ConstLabels: s.labels,
							},
						),
					)
					c.gaugesMu.Lock()
					c.gauges[string(h)] = m
					c.gaugesMu.Unlock()
				}

				m.Gauge.Set(s.value)
				m.Touch()

			case sampleHistogramLinear:
				m, found := c.histograms[string(h)]
				if !found {
					start, _ := strconv.ParseFloat(s.histogramDef[0], 10)
					width, _ := strconv.ParseFloat(s.histogramDef[1], 10)
					count, _ := strconv.Atoi(s.histogramDef[2])
					m = NewUpdatingHistogram(
						prometheus.NewHistogram(
							prometheus.HistogramOpts{
								Name:        s.name,
								Help:        "auto",
								ConstLabels: s.labels,
								Buckets:     prometheus.LinearBuckets(start, width, count),
							},
						),
					)
					c.histogramsMu.Lock()
					c.histograms[string(h)] = m
					c.histogramsMu.Unlock()
				}

				m.Histogram.Observe(s.value)
				m.Touch()

			case sampleHistogram:
				m, found := c.histograms[string(h)]
				if !found {
					var buckets = []float64{}
					for _, i := range s.histogramDef {
						j, err := strconv.ParseFloat(i, 64)
						if err != nil {
							panic(err)
						}
						buckets = append(buckets, j)
					}

					m = NewUpdatingHistogram(
						prometheus.NewHistogram(
							prometheus.HistogramOpts{
								Name:        s.name,
								Help:        "auto",
								ConstLabels: s.labels,
								Buckets:     buckets,
							},
						),
					)
					c.histogramsMu.Lock()
					c.histograms[string(h)] = m
					c.histogramsMu.Unlock()
				}

				m.Histogram.Observe(s.value)
				m.Touch()
			}

			c.testHookProcessSampleDone()

			c.metricProcessingDuration.WithLabelValues(string(s.kind)).
				Observe(float64(time.Since(tS).Nanoseconds()))

		case <-c.quitCh:
			close(c.shutdownDownCh)
			return
		}
	}
}

func (c *collector) processExpiring() {
	ticker := time.NewTicker(c.expiryTime)
	for {
		select {
		case <-ticker.C:
			c.expire()
		case <-c.quitCh:
			return
		}
	}
}

func (c *collector) expire() {
	now := time.Now()
	var ts time.Time

	c.countersMu.Lock()
	ts = time.Now()
	for k, m := range c.counters {
		if now.Sub(m.UpdatedAt) > c.expiryTime {
			delete(c.counters, k)
		}
	}
	c.countersMu.Unlock()
	c.metricExpiringDuration.WithLabelValues("counter").
		Observe(float64(time.Since(ts).Nanoseconds()))

	c.gaugesMu.Lock()
	ts = time.Now()
	for k, m := range c.gauges {
		if now.Sub(m.UpdatedAt) > c.expiryTime {
			delete(c.gauges, k)
		}
	}
	c.gaugesMu.Unlock()
	c.metricExpiringDuration.WithLabelValues("gauge").
		Observe(float64(time.Since(ts).Nanoseconds()))

	c.histogramsMu.Lock()
	ts = time.Now()
	for k, m := range c.histograms {
		if now.Sub(m.UpdatedAt) > c.expiryTime {
			delete(c.histograms, k)
		}
	}
	c.histogramsMu.Unlock()
	c.metricExpiringDuration.WithLabelValues("histogram").
		Observe(float64(time.Since(ts).Nanoseconds()))
}
