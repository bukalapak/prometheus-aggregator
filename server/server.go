package server

import (
	"net"
	"syscall"
	"time"

	pr "github.com/bukalapak/prometheus-aggregator"
	"github.com/bukalapak/prometheus-aggregator/protomodel"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

type Server struct {
	pProtor                       *pr.Protor
	buf                           []byte
	defRegistry                   *prometheus.Registry
	metricRequestsTotal           prometheus.Counter
	metricSamplesTotal            prometheus.Counter
	metricRequestHandlingDuration prometheus.Summary
}

func NewServer(pProtor *pr.Protor) *Server {
	s := Server{
		pProtor: pProtor,
		buf:     make([]byte, 1024),
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
		defRegistry: prometheus.NewRegistry(),
	}
	s.defRegistry.MustRegister(s.metricRequestsTotal)
	s.defRegistry.MustRegister(s.metricSamplesTotal)
	s.defRegistry.MustRegister(s.metricRequestHandlingDuration)
	return &s
}

func (s *Server) Run(ip string, port string) {
	laddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	tcp, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	for {
		conn, err := tcp.Accept()
		if err != nil {
			exitOnFatal(err, "TCPServer")
		}
		go s.handleRequest(conn)
	}
}

func (s *Server) handleRequest(conn net.Conn) {
	var tS time.Time
	reqLen, err := conn.Read(s.buf)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	tS = time.Now()
	protodata := &protomodel.Sample{}
	s.metricRequestsTotal.Inc()
	err = proto.Unmarshal(s.buf[0:reqLen], protodata)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	s.metricSamplesTotal.Add(float64(1))
	pSample := protoToSample(protodata)
	err = s.pProtor.WriteToCollector(pSample)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	s.metricRequestHandlingDuration.Observe(float64(time.Since(tS).Nanoseconds()))
}

func protoToSample(pd *protomodel.Sample) *pr.Sample {
	return &pr.Sample{
		Service:      pd.Service,
		Name:         pd.Name,
		Kind:         pd.Kind,
		Label:        pd.Label,
		Value:        pd.Value,
		HistogramDef: pd.HistogramDef,
	}
}

func exitOnFatal(err error, loc string) {
	log.Fatalf("EXIT on %s: err=%s\n", loc, err)
	syscall.Exit(1)
}
