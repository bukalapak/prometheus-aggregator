package registrymanager_test

import (
	"testing"

	rm "github.com/bukalapak/prometheus-aggregator/registrymanager"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RegistryManagerSuite struct {
	suite.Suite
	registryManager *rm.RegistryManager
}

func TestRegistryManagerSuite(t *testing.T) {
	suite.Run(t, &RegistryManagerSuite{})
}

func (rms *RegistryManagerSuite) SetupSuite() {
	rms.registryManager = rm.New()
}

func (rms *RegistryManagerSuite) TestRegistryManager() {
	//no registry
	registry, err := rms.registryManager.FindRegistry("test")
	assert.NotNil(rms.T(), err, "error at findregistry")
	assert.Nil(rms.T(), registry, "error at findregistry")

	//make registry
	registry = rms.registryManager.MakeRegistry("test")
	assert.NotNil(rms.T(), registry, "error at makeregistry")

	registry2, err := rms.registryManager.FindRegistry("test")
	assert.Nil(rms.T(), err, "error at findregistry")
	assert.NotNil(rms.T(), registry2, "error at findregistry")

}
