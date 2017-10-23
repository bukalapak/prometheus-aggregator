package main

import (
	"flag"
	"fmt"
	"net/http"
	"runtime"
	"syscall"

	//"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/log"
	"github.com/vrischmann/envconfig"
	
)

var (
	PRegistry = make(map[string]*prometheus.Registry)
)

type config struct {
	// TCPHost is address on which TCP server is listening
	TCPHost string `envconfig:"default=0.0.0.0"`

	// TCPPort is port number on which TCP server is listening
	TCPPort string `envconfig:"default=8080"`

	// TCPBufferSize is a size of a buffer in bytes used for incoming samples.
	// Sample not fitting in buffer will be partially discarded.
	// Sync buffer size with client.
	TCPBufferSize int `envconfig:"default=65536"`

	// MetricsHost is address on which metric server for prometheus is listening
	MetricsHost string `envconfig:"default=0.0.0.0"`

	// MetricsHost is port number on which metric server for prometheus is listening
	MetricsPort int `envconfig:"default=9090"`

	// LogLevel is a minimal log severity required for the message to be logged.
	// Valid levels: [debug, info, warn, error, fatal, panic].
	LogLevel string `envconfig:"default=info"`

	// MaxProcs limits number of processors used by the app.
	MaxProcs int `envconfig:"default=0"`

	// Metrics path for prometheus scrape
	MetricsPath string `envconfig:"default=/metrics"`
}

func main() {
	// -> config from env
	cfg := &config{}
	if err := envconfig.InitWithPrefix(&cfg, "APP"); err != nil {
		exitOnFatal(err, "init config")
	}

	// convert env config to flag one for prometheus log package
	flag.Set("log.level", cfg.LogLevel)
	flag.Parse()

	log.Debugf("Parsed config from env => %+v", *cfg)

	if cfg.MaxProcs != 0 {
		nGot := runtime.GOMAXPROCS(cfg.MaxProcs)
		log.Debugf("Processor limiting, Req: %d, MaxAvailable: %d, NumCPU: %d", cfg.MaxProcs, nGot, runtime.NumCPU())
	}

	c := newCollector()
	prometheus.MustRegister(c)
	c.start()

	s := newServer(c.Write, cfg.TCPBufferSize)
	log.Infof("Starting ingress samples server => %s:%s with buffersize %d", cfg.TCPHost, cfg.TCPPort, cfg.TCPBufferSize)
	if err := s.Listen(cfg.TCPHost, cfg.TCPPort); err != nil {
		exitOnFatal(err, "TCP server init")
	}


	log.Infof("Handle metrics endpoint in %s", cfg.MetricsPath)

	http.Handle(cfg.MetricsPath, prometheus.Handler())
	http.Handle("/", EndPoint())
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	
	metricsListenOn := fmt.Sprintf("%s:%d", cfg.MetricsHost, cfg.MetricsPort)
	log.Infof("Starting metrics server => %s", metricsListenOn)
	if err := http.ListenAndServe(metricsListenOn, nil); err != nil {
		exitOnFatal(err, "metric server")
	}
}

func exitOnFatal(err error, loc string) {
	log.Fatalf("EXIT on %s: err=%s\n", loc, err)
	syscall.Exit(1)
}

func EndPoint() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := PRegistry[r.URL.Path]

		if ok {
			h := promhttp.HandlerFor(PRegistry[r.URL.Path], promhttp.HandlerOpts{})
			h.ServeHTTP(w, r)
		} else {
			fmt.Fprint(w, "End Point not exist")
			return
		}
	})
}


