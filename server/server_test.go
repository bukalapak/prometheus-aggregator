package server_test

import (
	"net"
	"testing"

	pr "github.com/bukalapak/prometheus-aggregator"
	co "github.com/bukalapak/prometheus-aggregator/collector"
	pm "github.com/bukalapak/prometheus-aggregator/protomodel"
	rm "github.com/bukalapak/prometheus-aggregator/registrymanager"
	"github.com/bukalapak/prometheus-aggregator/server"
	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ServerSuite struct {
	suite.Suite
	server *server.Server
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, &ServerSuite{})
}

func (s *ServerSuite) SetupSuite() {
	collector := co.New()
	registryManager := rm.New()
	pProtor := pr.New(collector, registryManager)
	s.server = server.New(pProtor)
	go s.server.Run("0.0.0.0", "8080")
}

func (ss *ServerSuite) TestProtoToSample() {

	protoSample := &pm.Sample{
		Service:      "Test",
		Name:         "test_counter",
		Kind:         "c",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []string{},
	}
	sampleTemp := &pr.Sample{
		Service:      "Test",
		Name:         "test_counter",
		Kind:         "c",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []string{},
	}
	sampleProtor := server.ProtoToSample(protoSample)
	assert.Equal(ss.T(), sampleProtor, sampleTemp, "")
}

func (ss *ServerSuite) TestHandleRequest() {
	protoSample := &pm.Sample{
		Service:      "Test",
		Name:         "test_counter",
		Kind:         "c",
		Label:        make(map[string]string),
		Value:        1,
		HistogramDef: []string{},
	}

	out, err := proto.Marshal(protoSample)
	assert.Nil(ss.T(), err, "fail at marshal protobuf")

	conn, err := net.Dial("tcp", "0.0.0.0:8080")
	assert.Nil(ss.T(), err, "fail at connecting")
	defer conn.Close()
	conn.Write(out)
}