## Overview

Aggregator was designed as a way to bridge short-lived PHP scripts with [prometheus](https://github.com/prometheus/prometheus).
It extends ideas brought by [statsd_exporter](https://github.com/prometheus/statsd_exporter) by supporting native labeling and histograms.

Short-lived client is shooting samples via TCP toward aggregator server which parses, aggregates and stores them in memory.
The storage is then scraped using standard Prometheus HTTP endpoint (both text and binary exposition formats are supported).

    +----------+              +-------------------------+                                              +--------------+
    |  client  |---( TCP )--->|  prometheus_aggregator  |<---( scrape /metrics && /$servicename$ )---  |  Prometheus  |
    +----------+              +-------------------------+                                              +--------------+

## Protobuf format

Protobuf format for samples metrics is a serialized protocol buffer data. see Attache for serializing protobuf example.

| field | desc               | allowed values |
|-------|--------------------|----------------|
| service | name of the service | a-zA-Z0-9_ |
| name  | name of the metric | a-zA-Z0-9_ |
| type  | type of the metric | counter: c<br>gauge: g<br>histogram: h<br>histogram with linear buckets: hl |
| histogramdef | additional configuration for the type<br>currently used only for histograms | |
| labels | pairs of name and value separated by semicolon (;)<br>field is optional | name: a-zA-Z0-9<br>value: a-zA-Z0-9. |
| value | sample value<br>negative values are not yet supported | 0-9. |

## Metrics

As of now following metrics are supported:
- counter
- gauge
- histogram
- histogram with linear buckets

### Counters
### Gauges
### Histograms

If no bucket specified, use Prometheus default buckets
    
### Histograms with linear buckets

Type config values are passed to LinearBuckets(start, width float64, count int)


## Internals

### Architecture

There are four major components: sample server, registry manager, collector, and handler.

#### Sample server

Sample server is responsible for listening for the incoming samples via TCP, parsing each packet to samples and handing over to collector for processing.
As of now there is single goroutine responsible for reading and parsing.

#### Registry manager

Registry manager is responsible for :
- create a separate registry for each service,
- route each service metrics to their own service registry.

#### Collector

Collector is responsible for:
- create needed metrics vector, 
- processing of the samples to metrics vector,
- storing metrics in registry.

#### Handler

Handler is responsible for exposing each registry metrics to their respective endpoint. 
E.g :
````
    exposing attache registry metrics on "/attache".
````
### Metrics

| name | module | type | unit | desc |
|------|--------|------|------|------|
| app_handle_requests_total | server | counter | - | Number of request entering server. |
| app_samples_total | server | counter | - | Number of samples entering server. |
| app_handle_requests_duration_ns | server | summary | nanosecond | Time in ns spent on handling single request. |

## Usage

### Building
[govend](https://github.com/govend/govend) is used for vendoring.

```bash
govend -v 

go build
```

### Configuration

Configuration options
```go
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

    // ExpirationDate limits the time of vector life when its not used
    // numbers of day
    ExpirationDate int `envconfig:"default=1"`
```

### Running
```bash
# !/usr/bin/env bash

export APP_TCP_HOST="0.0.0.0"
export APP_TCP_PORT="9090"
export APP_TCP_BUFFER_SIZE="2048"
export APP_METRICS_HOST="0.0.0.0"
export APP_METRICS_PORT="8080"
export APP_LOG_LEVEL="DEBUG"
export APP_MAX_PROCS="0"

./prometheus-aggregator
```

## Running tests

    $ make test

## Using docker

```bash
docker pull rolandhawk/prometheus-aggregator

docker run --rm -p 10901:8080/udp -p 10902:9090 --name prometheus_aggregator rolandhawk/prometheus-aggregator
```
