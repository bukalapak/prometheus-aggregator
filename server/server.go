package server

import (
	"net"
	"syscall"

	pr "github.com/bukalapak/prometheus-aggregator"
	"github.com/bukalapak/prometheus-aggregator/protomodel"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/log"
)

type Server struct {
	Protor *pr.Protor
	buf    []byte
}

func New(pProtor *pr.Protor) *Server {
	s := Server{
		Protor: pProtor,
		buf:    make([]byte, 1024),
	}
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
	reqLen, err := conn.Read(s.buf)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	protodata := &protomodel.Sample{}
	err = proto.Unmarshal(s.buf[0:reqLen], protodata)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
	pSample := protoToSample(protodata)
	err = s.Protor.WriteToRegistry(pSample)
	if err != nil {
		exitOnFatal(err, "TCPServer")
	}
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
