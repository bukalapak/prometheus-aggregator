package main

import (
	"errors"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/bukalapak/prometheus-aggregator/protomodel"
	"github.com/rolandhawk/dynamicvector"
)

const (
	// TODO(szpakas): move to config
	ingressQueueSize = 1024 * 100
	expirationTime = 100 * time.Second
)

type collector struct {
	startTime time.Time

	// ingress holds incoming samples for processing
	ingressCh chan *protomodel.Sample

	counters map[string]*dynamicvector.Counter
	// countersMu protects scraping functions from interfering with processing
	countersMu sync.RWMutex

	gauges   map[string]*dynamicvector.Gauge
	gaugesMu sync.RWMutex

	histograms   map[string]*dynamicvector.Histogram
	histogramsMu sync.RWMutex

	//to check if the service with names and kind exist
	mapflag map[string] bool
	mapflagMu sync.RWMutex

	pregistryMU sync.RWMutex
 
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
}

func newCollector() *collector {
	return &collector{
		ingressCh:                 make(chan *protomodel.Sample, ingressQueueSize),
		counters:                  make(map[string]*dynamicvector.Counter),
		gauges:                    make(map[string]*dynamicvector.Gauge),
		histograms:                make(map[string]*dynamicvector.Histogram),
		testHookProcessSampleDone: func() {},
		quitCh:          make(chan struct{}),
		shutdownDownCh:  make(chan struct{}),
		shutdownTimeout: time.Second,

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
}


// Collect implements prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
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
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.metricAppStart.Describe(ch)
	c.metricAppDuration.Describe(ch)
	c.metricQueueLength.Describe(ch)
	c.metricProcessingDuration.Describe(ch)
}

func (c *collector) start() {
	c.startTime = time.Now()

	c.metricAppStart.Set(float64(c.startTime.UnixNano()) / 1e9)

	go c.process()
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
func (c *collector) Write(s *protomodel.Sample) error {
	select {
	case c.ingressCh <- s:
	default:
		return errors.New("collector: ingress queue is full")
	}
	return nil
}

func (c *collector) process() {
	var (
		s  *protomodel.Sample
		tS time.Time
	)
	for {
		select {
		case s = <-c.ingressCh:
			tS = time.Now()
			c.metricQueueLength.Set(float64(len(c.ingressCh)))
			sampleservice := s.Service
			samplename := s.Name
			samplekind := s.Kind
			samplevalue := s.Value
			sampleHDef := s.HistogramDef

			samplelabel := s.GetLabel()
			plabel:=prometheus.Labels{}
			plabel=samplelabel

			//checking if service registry exist
			c.pregistryMU.RLock()
			_, registryexist := PRegistry[sampleservice]
			c.pregistryMU.RUnlock()

			if !registryexist{
				c.pregistryMU.Lock()
				PRegistry[sampleservice] = prometheus.NewRegistry()
				c.pregistryMU.Unlock()
			}

			//checking if metric vector exist
			flagname := sampleservice+":"+samplekind+":"+samplename
			vectorname := sampleservice+":"+samplename
			c.mapflagMu.RLock()
			_, vectorexist := c.mapflag[flagname]
			c.mapflagMu.RUnlock()
			if !vectorexist{
				c.mapflagMu.Lock()
				c.mapflag[flagname]=true
				c.mapflagMu.Unlock()
				switch samplekind {
				case "c":
					c.countersMu.Lock()
					c.counters[vectorname] = dynamicvector.NewCounter(dynamicvector.CounterOpts{
						Name: samplename,
						Help: "auto",
						Expire: expirationTime,
					})
					c.countersMu.Unlock()
					PRegistry[sampleservice].MustRegister(c.counters[vectorname])
					
				case "g":
					c.gaugesMu.Lock()
					c.gauges[vectorname] = dynamicvector.NewGauge(dynamicvector.GaugeOpts{
						Name: samplename,
						Help: "auto",
						Expire: expirationTime,
					})
					c.gaugesMu.Unlock()
					PRegistry[sampleservice].MustRegister(c.gauges[vectorname])

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
						Name: samplename,
						Help: "auto",
						Buckets:     buckets,
						Expire: expirationTime,
					})
					c.histogramsMu.Unlock()
					PRegistry[sampleservice].MustRegister(c.histograms[vectorname])

				case "hl":
					start, _ := strconv.ParseFloat(sampleHDef[0], 10)
					width, _ := strconv.ParseFloat(sampleHDef[1], 10)
					count, _ := strconv.Atoi(sampleHDef[2])
					c.histogramsMu.Lock()
					c.histograms[vectorname] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
						Name: samplename,
						Help: "auto",
						Buckets:     prometheus.LinearBuckets(start, width, count),
						Expire: expirationTime,
					})
					c.histogramsMu.Unlock()
					PRegistry[sampleservice].MustRegister(c.histograms[vectorname])
				}		
			}

			switch samplekind {
			case "c":
				c.counters[vectorname].With(plabel).Add(samplevalue)
			case "g":
				c.gauges[vectorname].With(plabel).Set(samplevalue)
			case "h":
				c.histograms[vectorname].With(plabel).Observe(samplevalue)
			}


			c.testHookProcessSampleDone()

			c.metricProcessingDuration.WithLabelValues(string(samplekind)).
				Observe(float64(time.Since(tS).Nanoseconds()))


		case <-c.quitCh:
			close(c.shutdownDownCh)
			return
		}
	}
}
