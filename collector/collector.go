package collector

import (
	"strconv"
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
}

func New() *Collector {
	return &Collector{
		mapFlag:    make(map[string]bool),
		counters:   make(map[string]*dynamicvector.Counter),
		gauges:     make(map[string]*dynamicvector.Gauge),
		histograms: make(map[string]*dynamicvector.Histogram),
	}
}

func (c *Collector) Write(s *protor.Sample, Registry *prometheus.Registry) error {
	ExpirationTime := 24 * time.Hour
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
			c.countersMu.Unlock()

		case "g":

			c.gaugesMu.Lock()
			c.gauges[s.Name] = dynamicvector.NewGauge(dynamicvector.GaugeOpts{
				Name:   s.Name,
				Help:   "auto",
				Expire: ExpirationTime,
			})
			c.gaugesMu.Unlock()

		case "h":

			var buckets = []float64{}
			for _, i := range s.HistogramDef {
				j, err := strconv.ParseFloat(i, 64)
				if err != nil {
					return err
				}
				buckets = append(buckets, j)
			}

			c.histogramsMu.Lock()
			c.histograms[s.Name] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
				Name:    s.Name,
				Help:    "auto",
				Buckets: buckets,
				Expire:  ExpirationTime,
			})
			c.histogramsMu.Unlock()

		case "hl":

			start, err := strconv.ParseFloat(s.HistogramDef[0], 10)
			if err != nil {
				return err
			}
			width, err := strconv.ParseFloat(s.HistogramDef[1], 10)
			if err != nil {
				return err
			}
			count, err := strconv.Atoi(s.HistogramDef[2])
			if err != nil {
				return err
			}
			c.histogramsMu.Lock()
			c.histograms[s.Name] = dynamicvector.NewHistogram(dynamicvector.HistogramOpts{
				Name:    s.Name,
				Help:    "auto",
				Buckets: prometheus.LinearBuckets(start, width, count),
				Expire:  ExpirationTime,
			})

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
