package registry_manager

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
)

type RegistryManager struct {
	RegistryList map[string]*prometheus.Registry
}

func New() *RegistryManager {
	return &RegistryManager{
		RegistryList: make(map[string]*prometheus.Registry),
	}
}

func (rm *RegistryManager) FindRegistry(s string) (*prometheus.Registry, error) {
	_, ok := rm.RegistryList[s]
	if !ok {
		return nil, errors.New("No Registry")
	}
	return rm.RegistryList[s], nil
}

func (rm *RegistryManager) MakeRegistry(s string) *prometheus.Registry {
	rm.RegistryList[s] = prometheus.NewRegistry()
	return rm.RegistryList[s]
}
