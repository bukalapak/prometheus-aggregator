package collector

import (
	"errors"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/bukalapak/prometheus-aggregator"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rolandhawk/dynamicvector"
)

const (
	// TODO(szpakas): move to config
	ingressQueueSize = 1024 * 100
	expirationTime   = 100 * time.Second
)

type Collector struct {
	startTime      time.Time
	queueCh        chan *protor.Sample
	expirationTime time.Duration

	PRegistry   map[string]*prometheus.Registry
	PRegistryMu sync.RWMutex

	mapFlag   map[string]bool
	mapFlagMu sync.RWMutex

	counters   map[string]*dynamicvector.Counter
	countersMu sync.RWMutex

	gauges   map[string]*dynamicvector.Gauge
	gaugesMu sync.RWMutex

	histograms   map[string]*dynamicvector.Histogram
	histogramsMu sync.RWMutex

	quitCh chan struct{}

	testHookProcessSampleDone func()

	shutdownDownCh  chan struct{}
	shutdownTimeout time.Duration

	metricAppStart           prometheus.Gauge
	metricAppDuration        prometheus.Gauge
	metricQueueLength        prometheus.Gauge
	metricProcessingDuration *prometheus.SummaryVec

	Metricz *prometheus.Registry
}

func NewCollector() *Collector {
	c := Collector{
		queueCh:                   make(chan *protor.Sample, ingressQueueSize),
		PRegistry:                 make(map[string]*prometheus.Registry),
		mapFlag:                   make(map[string]bool),
		counters:                  make(map[string]*dynamicvector.Counter),
		gauges:                    make(map[string]*dynamicvector.Gauge),
		histograms:                make(map[string]*dynamicvector.Histogram),
		quitCh:                    make(chan struct{}),
		shutdownDownCh:            make(chan struct{}),
		shutdownTimeout:           time.Second,
		testHookProcessSampleDone: func() {},
		expirationTime:            expirationTime,
		Metricz:                   prometheus.NewRegistry(),
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
	}
	c.Metricz.MustRegister(c.metricAppDuration)
	c.Metricz.MustRegister(c.metricAppStart)
	c.Metricz.MustRegister(c.metricQueueLength)
	c.Metricz.MustRegister(c.metricProcessingDuration)
	return &c
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.metricAppStart.Collect(ch)

	c.metricAppDuration.Set(time.Now().Sub(c.startTime).Seconds())
	c.metricAppDuration.Collect(ch)

	c.metricQueueLength.Collect(ch)
	c.metricProcessingDuration.Collect(ch)

	c.countersMu.RLock()
	for _, m := range c.counters {
		m.Collect(ch)
	}
	c.countersMu.RUnlock()

	c.gaugesMu.RLock()
	for _, m := range c.gauges {
		m.Collect(ch)
	}
	c.gaugesMu.RUnlock()

	c.histogramsMu.RLock()
	for _, m := range c.histograms {
		m.Collect(ch)
	}
	c.histogramsMu.RUnlock()
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	c.metricAppStart.Describe(ch)
	c.metricAppDuration.Describe(ch)
	c.metricQueueLength.Describe(ch)
	c.metricProcessingDuration.Describe(ch)
}

func (c *Collector) Start() error {
	c.startTime = time.Now()

	c.metricAppStart.Set(float64(c.startTime.UnixNano()) / 1e9)

	go c.process()
	return nil

}

func (c *Collector) Stop() error {
	close(c.quitCh)
	runtime.Gosched()

	select {
	case <-c.shutdownDownCh:
	case <-time.After(c.shutdownTimeout):
		return errors.New("collector: shutdown timed out")
	}

	return nil
}

func (c *Collector) IsRegistryExist(s string) (*prometheus.Registry, error) {
	c.PRegistryMu.RLock()
	_, ok := c.PRegistry[s]
	c.PRegistryMu.RUnlock()
	if ok {
		return c.PRegistry[s], nil
	}
	if s == "metricz" {
		return c.Metricz, nil
	}

	return nil, errors.New("registry not exist")
}

// Write adds samples to internal queue for processing.
// Will result in ErrIngressQueueFull error if queue is full. The sample is not added to queue in such case.
func (c *Collector) Write(s *protor.Sample) error {
	select {
	case c.queueCh <- s:
	default:
		return errors.New("collector: ingress queue is full")
	}
	return nil
}

func (c *Collector) process() {
	var (
		s  *protor.Sample
		tS time.Time
	)
	for {
		select {
		case s = <-c.queueCh:
			tS = time.Now()
			c.metricQueueLength.Set(float64(len(c.queueCh)))
			sampleservice := s.Service
			samplename := s.Name
			samplekind := s.Kind
			samplevalue := s.Value
			sampleHDef := s.HistogramDef
			samplelabel := s.Label

			plabel := prometheus.Labels{}
			plabel = samplelabel

			//checking if service registry exist
			_, registryexist := c.IsRegistryExist(sampleservice)

			if registryexist != nil {
				c.PRegistryMu.Lock()
				c.PRegistry[sampleservice] = prometheus.NewRegistry()
				c.PRegistryMu.Unlock()
			}

			flagname := sampleservice + ":" + samplekind + ":" + samplename
			vectorname := sampleservice + ":" + samplename

			//checking if metric vector exist
			c.mapFlagMu.RLock()
			_, vectorexist := c.mapFlag[flagname]
			c.mapFlagMu.RUnlock()

			if !vectorexist {

				c.mapFlagMu.Lock()
				c.mapFlag[flagname] = true
				c.mapFlagMu.Unlock()

				switch samplekind {

				case "c":
					c.countersMu.Lock()
					c.counters[vectorname] = dynamicvector.NewCounter(dynamicvector.CounterOpts{
						Name:   samplename,
						Help:   "auto",
						Expire: expirationTime,
					})
					c.countersMu.Unlock()
					c.PRegistry[sampleservice].MustRegister(c.counters[vectorname])

				case "g":
					c.gaugesMu.Lock()
					c.gauges[vectorname] = dynamicvector.NewGauge(dynamicvector.GaugeOpts{
						Name:   samplename,
						Help:   "auto",
						Expire: expirationTime,
					})
					c.gaugesMu.Unlock()
					c.PRegistry[sampleservice].MustRegister(c.gauges[vectorname])

				case "h":
					var buckets = []float64{}
					for _, i := range sampleHDef {
						j, err := strconv.ParseFloat(i, 64)
						if err != nil {
							panic(err)
						}
						buckets = append(buckets, j)
					}
					c.histogramsMu.Lock()
					c.histograms[vectorname] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
						Name:    samplename,
						Help:    "auto",
						Buckets: buckets,
						Expire:  expirationTime,
					})
					c.histogramsMu.Unlock()
					c.PRegistry[sampleservice].MustRegister(c.histograms[vectorname])

				case "hl":
					start, _ := strconv.ParseFloat(sampleHDef[0], 10)
					width, _ := strconv.ParseFloat(sampleHDef[1], 10)
					count, _ := strconv.Atoi(sampleHDef[2])
					c.histogramsMu.Lock()
					c.histograms[vectorname] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
						Name:    samplename,
						Help:    "auto",
						Buckets: prometheus.LinearBuckets(start, width, count),
						Expire:  expirationTime,
					})
					c.histogramsMu.Unlock()
					c.PRegistry[sampleservice].MustRegister(c.histograms[vectorname])
				}
			}

			switch samplekind {

			case "c":
				c.counters[vectorname].With(plabel).Add(samplevalue)

			case "g":
				c.gauges[vectorname].With(plabel).Set(samplevalue)

			case "h":
				c.histograms[vectorname].With(plabel).Observe(samplevalue)

			case "hl":
				c.histograms[vectorname].With(plabel).Observe(samplevalue)

			}

			c.testHookProcessSampleDone()

			c.metricProcessingDuration.WithLabelValues(string(samplekind)).Observe(float64(time.Since(tS).Nanoseconds()))

		case <-c.quitCh:
			close(c.shutdownDownCh)
			return
		}
	}
}
