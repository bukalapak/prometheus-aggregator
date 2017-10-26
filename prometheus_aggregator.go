package protor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Protor struct {
	Collector       CollectorInterface
	RegistryManager RegistryManagerInterface
}

type Sample struct {
	Service      string            `protobuf:"bytes,1,opt,name=service" json:"service,omitempty"`
	Name         string            `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Kind         string            `protobuf:"bytes,3,opt,name=kind" json:"kind,omitempty"`
	Label        map[string]string `protobuf:"bytes,4,rep,name=label" json:"label,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Value        float64           `protobuf:"fixed64,5,opt,name=value" json:"value,omitempty"`
	HistogramDef []string          `protobuf:"bytes,6,rep,name=histogramDef" json:"histogramDef,omitempty"`
}

type CollectorInterface interface {
	Write(*Sample, *prometheus.Registry) error
}

type RegistryManagerInterface interface {
	FindRegistry(string) (*prometheus.Registry, error)
	MakeRegistry(string) *prometheus.Registry
}

func New(c CollectorInterface, rm RegistryManagerInterface) *Protor {
	return &Protor{
		Collector:       c,
		RegistryManager: rm,
	}
}

func (p *Protor) WriteToRegistry(s *Sample) error {
	registry, err := p.RegistryManager.FindRegistry(s.Name)
	if err != nil {
		registry = p.RegistryManager.MakeRegistry(s.Name)
	}
	return p.Collector.Write(s, registry)
}
