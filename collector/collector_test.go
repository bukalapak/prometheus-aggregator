package collector_test

import (
	"testing"

	pr "github.com/bukalapak/prometheus-aggregator"
	co "github.com/bukalapak/prometheus-aggregator/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CollectorSuite struct {
	suite.Suite
	collector *co.Collector
}

func TestCollectorSuite(t *testing.T) {
	suite.Run(t, &CollectorSuite{})
}

func (cs *CollectorSuite) SetupSuite() {
	cs.collector = co.New()
}

func (cs *CollectorSuite) TestWrite() {

	TestRegistry := prometheus.NewRegistry()

	TestCounter := &pr.Sample{
		Service:      "Test",
		Name:         "test_counter",
		Kind:         "c",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []float64{},
	}
	TestGauge := &pr.Sample{
		Service:      "Test",
		Name:         "test_gauge",
		Kind:         "g",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []float64{},
	}
	TestHistogram := &pr.Sample{
		Service:      "Test",
		Name:         "test_histogram",
		Kind:         "h",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []float64{},
	}
	TestHistogramLinearFail := &pr.Sample{
		Service:      "Test",
		Name:         "test_histogram_linear_wrong",
		Kind:         "hl",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []float64{},
	}
	TestHistogramLinear := &pr.Sample{
		Service:      "Test",
		Name:         "test_histogram_linear",
		Kind:         "hl",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []float64{3.1415, 3.1415, 22},
	}

	err := cs.collector.Write(TestCounter, TestRegistry)
	assert.Nil(cs.T(), err, "Failed to write counter")

	err = cs.collector.Write(TestGauge, TestRegistry)
	assert.Nil(cs.T(), err, "Failed to write gauge")

	err = cs.collector.Write(TestHistogram, TestRegistry)
	assert.Nil(cs.T(), err, "Failed to write histogram")

	err = cs.collector.Write(TestHistogramLinear, TestRegistry)
	assert.Nil(cs.T(), err, "Failed to write linear")

	err = cs.collector.Write(TestHistogramLinearFail, TestRegistry)
	assert.NotNil(cs.T(), err, "linear suppose to be failed")

}
