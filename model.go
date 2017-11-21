package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type sampleHasherFunc func(*sample) []byte

// sampleHasher is a hashing function used on samples.
var sampleHasher sampleHasherFunc

type sampleKind string

const (
	sampleUnknown sampleKind = ""

	// sampleCounter represents a counter
	sampleCounter sampleKind = "c"

	// sampleGauge represents a gauge
	sampleGauge sampleKind = "g"

	// sampleHistogram represents histogram
	sampleHistogram sampleKind = "h"

	// sampleHistogramLinear represents histogram with linearly spaced buckets.
	// See Prometheus Go client LinearBuckets for details.
	sampleHistogramLinear sampleKind = "hl"
)

// sample represents single measurement submitted to the system.
// Samples are converted to metrics by collector.
type sample struct {
	// name is used to represent sample. It's used as metric name in export to prometheus.
	name string

	// kind of the sample wen mapped to prometheus metric type
	kind sampleKind

	// labels is a set of string pairs mapped to prometheus LabelPairs type
	labels map[string]string

	// value of the sample
	value float64

	// histogramDef is a set of values used in mapping for the histogram types
	histogramDef []string
}

// hash calculates a hash of the sample so it can be recognized.
// Should take all elements other than value under consideration.
func (s *sample) hash() []byte {
	return sampleHasher(s)
}

// UpdatingCounter wraps prometheus.Counter, adding last update time.
type UpdatingCounter struct {
	Counter   prometheus.Counter
	UpdatedAt time.Time
}

// NewUpdatingCounter creates new instance of UpdatingCounter, with UpdatedAt
// set to creation time.
func NewUpdatingCounter(c prometheus.Counter) *UpdatingCounter {
	return &UpdatingCounter{c, time.Now()}
}

// Touch updates UpdatedAt field to current time.
func (u *UpdatingCounter) Touch() {
	u.UpdatedAt = time.Now()
}

// UpdatingGauge wraps prometheus.Gauge, adding last update time.
type UpdatingGauge struct {
	Gauge     prometheus.Gauge
	UpdatedAt time.Time
}

// NewUpdatingGauge creates new instance of UpdatingGauge, with UpdatedAt
// set to creation time.
func NewUpdatingGauge(c prometheus.Gauge) *UpdatingGauge {
	return &UpdatingGauge{c, time.Now()}
}

// Touch updates UpdatedAt field to current time.
func (u *UpdatingGauge) Touch() {
	u.UpdatedAt = time.Now()
}

// UpdatingHistogram wraps prometheus.Histogram, adding last update time.
type UpdatingHistogram struct {
	Histogram prometheus.Histogram
	UpdatedAt time.Time
}

// NewUpdatingHistogram creates new instance of UpdatingHistogram, with UpdatedAt
// set to creation time.
func NewUpdatingHistogram(c prometheus.Histogram) *UpdatingHistogram {
	return &UpdatingHistogram{c, time.Now()}
}

// Touch updates UpdatedAt field to current time.
func (u *UpdatingHistogram) Touch() {
	u.UpdatedAt = time.Now()
}
