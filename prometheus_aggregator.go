package protor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Protor struct {
	collector Collector
}

type Sample struct {
	Service      string            `protobuf:"bytes,1,opt,name=service" json:"service,omitempty"`
	Name         string            `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Kind         string            `protobuf:"bytes,3,opt,name=kind" json:"kind,omitempty"`
	Label        map[string]string `protobuf:"bytes,4,rep,name=label" json:"label,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Value        float64           `protobuf:"fixed64,5,opt,name=value" json:"value,omitempty"`
	HistogramDef []string          `protobuf:"bytes,6,rep,name=histogramDef" json:"histogramDef,omitempty"`
}

type Collector interface {
	Write(*Sample) error
	IsRegistryExist(string) (*prometheus.Registry, error)
}

func NewProtor(c Collector) *Protor {
	return &Protor{
		collector: c,
	}
}

func (p *Protor) WriteToCollector(s *Sample) error {
	err := p.collector.Write(s)
	if err != nil {
		return err
	}
	return nil
}

func (p *Protor) AskForRegistry(s string) (*prometheus.Registry, error) {
	return p.collector.IsRegistryExist(s)
}
