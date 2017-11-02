package collector

import (
	"errors"
	"sync"
	"time"

	"github.com/bukalapak/prometheus-aggregator"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rolandhawk/dynamicvector"
)

type Collector struct {
	mapFlag   map[string]bool
	mapFlagMu sync.Mutex

	counters   map[string]*dynamicvector.Counter
	countersMu sync.Mutex

	gauges   map[string]*dynamicvector.Gauge
	gaugesMu sync.Mutex

	histograms   map[string]*dynamicvector.Histogram
	histogramsMu sync.Mutex

	ExpirationTime time.Duration
}

func New(days int) *Collector {
	return &Collector{
		mapFlag:        make(map[string]bool),
		counters:       make(map[string]*dynamicvector.Counter),
		gauges:         make(map[string]*dynamicvector.Gauge),
		histograms:     make(map[string]*dynamicvector.Histogram),
		ExpirationTime: time.Duration(days*24) * time.Hour,
	}
}

func (c *Collector) Write(s *protor.Sample, Registry *prometheus.Registry) error {
	ExpirationTime := c.ExpirationTime
	plabel := prometheus.Labels{}
	plabel = s.Label
	//check if vector exist, make one if not.
	c.mapFlagMu.Lock()
	_, vectorExist := c.mapFlag[s.Name]
	if !vectorExist {

		c.mapFlag[s.Name] = true
		c.mapFlagMu.Unlock()

		switch s.Kind {
		case "c":

			c.countersMu.Lock()
			c.counters[s.Name] = dynamicvector.NewCounter(dynamicvector.CounterOpts{
				Name:   s.Name,
				Help:   "auto",
				Expire: ExpirationTime,
			})
			Registry.MustRegister(c.counters[s.Name])
			c.countersMu.Unlock()

		case "g":

			c.gaugesMu.Lock()
			c.gauges[s.Name] = dynamicvector.NewGauge(dynamicvector.GaugeOpts{
				Name:   s.Name,
				Help:   "auto",
				Expire: ExpirationTime,
			})
			Registry.MustRegister(c.gauges[s.Name])
			c.gaugesMu.Unlock()

		case "h":

			var buckets = []float64{}
			for _, i := range s.HistogramDef {
				buckets = append(buckets, i)
			}

			c.histogramsMu.Lock()
			c.histograms[s.Name] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
				Name:    s.Name,
				Help:    "auto",
				Buckets: buckets,
				Expire:  ExpirationTime,
			})
			Registry.MustRegister(c.histograms[s.Name])
			c.histogramsMu.Unlock()

		case "hl":
			if len(s.HistogramDef) < 3 {
				return errors.New("not enough parameter")
			}
			start := s.HistogramDef[0]
			width := s.HistogramDef[1]
			count := int(s.HistogramDef[2])
			c.histogramsMu.Lock()
			c.histograms[s.Name] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
				Name:    s.Name,
				Help:    "auto",
				Buckets: prometheus.LinearBuckets(start, width, count),
				Expire:  ExpirationTime,
			})
			Registry.MustRegister(c.histograms[s.Name])
			c.histogramsMu.Unlock()

		}
	} else {
		c.mapFlagMu.Unlock()
	}

	//process metrics
	switch s.Kind {
	case "c":
		c.counters[s.Name].With(plabel).Add(s.Value)
	case "g":
		c.gauges[s.Name].With(plabel).Set(s.Value)
	case "h":
		c.histograms[s.Name].With(plabel).Observe(s.Value)
	case "hl":
		c.histograms[s.Name].With(plabel).Observe(s.Value)
	}
	return nil
}
