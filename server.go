package main

import (
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/golang/protobuf/proto"
	"github.com/bukalapak/prometheus-aggregator/protomodel"
)

type sampleHandler func(samples *protomodel.Sample) error
//type sampleHandler func(samples int) error

type server struct {
	sampleHandler sampleHandler
	buf           []byte

	metricRequestsTotal           prometheus.Counter
	metricSamplesTotal            prometheus.Counter
	metricRequestHandlingDuration prometheus.Summary
}

// newServer is factory for TCP server for incoming metrics data
//
// handler is a function of sampleHandler type responsible for dealing with incoming samples
// bs is a TCP buffer size in bytes
func newServer(handler sampleHandler, bs int) *server {
	s := server{
		sampleHandler: handler,
		buf:           make([]byte, bs),
		metricRequestsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "app_ingress_requests_total",
				Help: "Number of request entering server.",
			},
		),
		metricSamplesTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "app_ingress_samples_total",
				Help: "Number of samples entering server.",
			},
		),
		metricRequestHandlingDuration: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "app_ingress_request_handling_duration_ns",
				Help: "Time in ns spent on handling single request.",
			},
		),
	}
	prometheus.MustRegister(s.metricRequestsTotal)
	prometheus.MustRegister(s.metricSamplesTotal)
	prometheus.MustRegister(s.metricRequestHandlingDuration)
	return &s
}

func (s *server) Listen(ip string, port string) error {
	laddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		return errors.Wrap(err, "opening server socket failed")
	}
	conn, err := net.ListenTCP("tcp",laddr)
	if err != nil {
		return errors.Wrap(err, "opening server socket failed")
	}
	go s.handleRequest(conn)
	
	return nil
}

func (s *server) handleRequest(tcp *net.TCPListener) {		
	var (	
		tS     time.Time
	)
	conn, err := tcp.Accept()
	if err != nil {

	}
	for{
		reqLen, err := conn.Read(s.buf)
	 	if err != nil {
			return
		}
		tS = time.Now()
		s.metricRequestsTotal.Inc()
		protodata := &protomodel.Sample{}
		err = proto.Unmarshal(s.buf[0:reqLen], protodata)
		s.metricSamplesTotal.Add(float64(1))
		err = s.sampleHandler(protodata)
		if err != nil{

		}
		s.metricRequestHandlingDuration.Observe(float64(time.Since(tS).Nanoseconds()))
	}
}

