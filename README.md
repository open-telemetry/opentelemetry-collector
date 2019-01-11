# OpenCensus Service

[![Build Status][travis-image]][travis-url]
[![GoDoc][godoc-image]][godoc-url]
[![Gitter chat][gitter-image]][gitter-url]

# Table of contents
- [Introduction](#introduction)
- [Goals](#goals)
- [Deployment](#deploy-example)
    - [Kubernetes](#deploy-k8s)
    - [Standalone](#deploy-standalone)
- [Configuration](#config-file)
    - [Receivers](#config-receivers)
    - [Exporters](#config-exporters)
    - [Diagnostics](#config-diagnostics)
- [OpenCensus Agent](#opencensus-agent)
    - [Building binaries](#agent-building-binaries)
    - [Usage](#agent-usage)
- [OpenCensus Collector](#opencensus-collector)
    - [Usage](#collector-usage)

## Introduction

The OpenCensus Service is an component that can collect traces
and metrics from processes instrumented by OpenCensus or other
monitoring/tracing libraries (Jaeger, Prometheus, etc.), do
aggregation and smart sampling, and export traces and metrics
to one or more monitoring/tracing backends.

Some frameworks and ecosystems are now providing out-of-the-box
instrumentation by using OpenCensus, but the user is still expected
to register an exporter in order to export data. This is a problem
during an incident. Even though our users can benefit from having
more diagnostics data coming out of services already instrumented
with OpenCensus, they have to modify their code to register an
exporter and redeploy. Asking our users recompile and redeploy is
not an ideal at an incident time. In addition, currently users need
to decide which service backend they want to export to, before they
distribute their binary instrumented by OpenCensus.

The OpenCensus Service is trying to eliminate these requirements. With the
OpenCensus Service, users do not need to redeploy or restart their applications
as long as it has the OpenCensus exporter. All they need to do is
just configure and deploy the OpenCensus Service separately. The OpenCensus
Service will then automatically collect traces and metrics and export to any
backend of users' choice.

Currently the OpenCensus Service consists of two components,
[OpenCensus Agent](#opencensus-agent) and [OpenCensus Collector](#opencensus-collector).
High-level workflow:

![service-architecture](images/opencensus-service.png)

For the detailed design specs, please see [DESIGN.md](DESIGN.md).

## Goals

* Allow enabling/configuring of exporters lazily. After deploying code,
optionally run a daemon on the host and it will read the
collected data and upload to the configured backend.
* Binaries can be instrumented without thinking about the exporting story.
Allows open source binary projects (e.g. web servers like Caddy or Istio Mixer)
to adopt OpenCensus without having to link any exporters into their binary.
* Easier to scale the exporter development. Not every language has to
implement support for each backend.
* Custom daemons containing only the required exporters compiled in can be created.

## <a name="deploy-example"></a>Deployment

The OpenCensus Service can be deployed in a variety of different ways. The OpenCensus
Agent can be deployed with the application either as a separate process, as a sidecar,
or via a Kubernetes daemonset. Typically, the OpenCensus Collector is deployed
separately as either a Docker container, VM, or Kubernetes pod.

![deployment-models](images/opencensus-service-deployment-models.png)

### <a name="deploy-k8s"></a>Kubernetes

Coming soon!

### <a name="deploy-standalone"></a>Standalone

Create an Agent [configuration file](#configuration-file) as described above.

Build the Agent, see [Building binaries](#agent-building-binaries),
and start it:

```shell
$ ./bin/ocagent_$(go env GOOS)
$ 2018/10/08 21:38:00 Running OpenCensus receiver as a gRPC service at "127.0.0.1:55678"
```

Create a Collector [configuration file](#configuration-file) as described above.

Build the Collector and start it:

```shell
$ make collector
$ ./bin/occollector_$($GOOS)
```

Run the demo application:

```shell
$ go run "$(go env GOPATH)/src/github.com/census-instrumentation/opencensus-service/example/main.go"
```

You should be able to see the traces in your exporter(s) of choice.
If you stop the ocagent, the example application will stop exporting.
If you run it again, exporting will resume.

## <a name="config-file"></a>Configuration

The OpenCensus Service (both the Agent and Collector) is configured via a config.yaml file. In general, you need to
configure one or more receivers as well as one or more exporters. In addition, diagnostics can also be configured.

### <a name="config-receivers"></a>Receivers

A receiver is how you get data into the OpenCensus Service. One or more receivers can be configured. By default,
the ``opencensus`` receiver is enabled on the OpenCensus Agent while no receivers are enabled on the OpenCensus Collector.

A basic example of all available receivers is provided below. For detailed receiver configuration,
please see the [receiver README.md](receiver/README.md).
```yaml
receivers:
  opencensus:
    address: "localhost:55678"

  zipkin:
    address: "localhost:9411"

  jaeger:
    collector_thrift_port: 14267
    collector_http_port: 14268
```

### <a name="config-exporters"></a>Exporters

An exporter is how you send data to one or more backends/destinations. One or more exporters can be configured.
By default, no exporters are configured on the OpenCensus Agent or Collector.

A basic example of all available exporters is provided below. For detailed exporter configuration,
please see the [exporter README.md](exporter/exporterparser/README.md).
```yaml
exporters:
  opencensus:
    endpoint: "localhost:10001"

  jaeger:
    collector_endpoint: "http://localhost:14268/api/traces"

  kafka:
    brokers: ["localhost:9092"]
    topic: "opencensus-spans"

  stackdriver:
    project: "your-project-id"
    enable_tracing: true

  zipkin:
    endpoint: "http://localhost:9411/api/v2/spans"
```

### <a name="config-diagnostics"></a>Diagnostics

zPages is provided for monitoring. Today, the OpenCensus Agent is configured with zPages running by default on port ``55679``.
These routes below contain the various diagnostic resources:

Resource|Route
---|---
RPC stats|/debug/rpcz
Trace information|/debug/tracez

The zPages configuration can be updated in the config.yaml file with fields:
* `disabled`: if set to true, won't run zPages
* `port`: by default is 55679, otherwise should be set to a value between 0 an 65535

For example:
```yaml
zpages:
    port: 8888 # To override the port from 55679 to 8888
```

To disable zPages, you can use `disabled` like this:
```yaml
zpages:
    disabled: true
```

and for example navigating to http://localhost:55679/debug/tracez to debug the
OpenCensus receiver's traces in your browser should produce something like this

![zPages](https://user-images.githubusercontent.com/4898263/47132981-892bb500-d25b-11e8-980c-08f0115ba72e.png)

## OpenCensus Agent

### <a name="agent-building-binaries"></a>Building binaries

> It is recommended that you use the latest [release](https://github.com/census-instrumentation/opencensus-service/releases).

Please run file `build_binaries.sh` in the root of this repository, with argument `binaries` or any of:
* linux
* darwin
* windows

which will then place the binaries in the directory `bin` which is in your current working directory
```shell
$ ./build_binaries.sh binaries

GOOS=darwin go build -ldflags "-X github.com/census-instrumentation/opencensus-service/internal/version.GitHash=8e102b4" -o bin/ocagent_darwin ./cmd/ocagent
GOOS=linux go build -ldflags "-X github.com/census-instrumentation/opencensus-service/internal/version.GitHash=8e102b4" -o bin/ocagent_linux ./cmd/ocagent
GOOS=windows go build -ldflags "-X github.com/census-instrumentation/opencensus-service/internal/version.GitHash=8e102b4" -o bin/ocagent_windows ./cmd/ocagent
```
which should then create binaries inside `bin/` that have a version command attached to them such as
```shell
$ ./bin/ocagent_darwin version

Version      0.0.1
GitHash      8e102b4
Goversion    devel +7f3313133e Mon Oct 15 22:11:26 2018 +0000
OS           darwin
Architecture amd64
```

### <a name="agent-usage"></a>Usage

The ocagent can be run directly from sources, binary, or a Docker image.

The minimum Go version required for this project is Go 1.11.4.

1. Run from sources:

```shell
$ GO111MODULE=on go get github.com/census-instrumentation/opencensus-service/cmd/ocagent
```

2. Run from binary (from the root of your repo):

```shell
$ make agent
```

3. Build a Docker scratch image and use the appropriate Docker command for your scenario
(note: additional ports may be required depending on your receiver configuration):

```shell
./build_binaries.sh docker <image_version>
```

For example, to create a Docker image of the agent, tagged `v1.0.0`:
```shell
./build_binaries.sh docker v1.0.0
```

and then the Docker image `v1.0.0` of the agent can be started  by
```shell
docker run -v $(pwd)/config.yaml:/config.yaml  -p 55678:55678  ocagent:v1.0.0
```

A Docker scratch image can be built with make by targeting `docker-agent`.

## OpenCensus Collector

The OpenCensus Collector is a component that runs “nearby” (e.g. in the same
VPC, AZ, etc.) a user’s application components and receives trace spans and
metrics emitted by the OpenCensus Agent or tasks instrumented with OpenCensus
instrumentation (or other supported protocols/libraries). The received spans
and metrics could be emitted directly by clients in instrumented tasks, or
potentially routed via intermediate proxy sidecar/daemon agents (such as the
OpenCensus Agent). The collector provides a central egress point for exporting
traces and metrics to one or more tracing and metrics backends, with buffering
and retries as well as advanced aggregation, filtering and annotation
capabilities.

The collector is extensible enabling it to support a range of out-of-the-box
(and custom) capabilities such as:

* Retroactive (tail-based) sampling of traces
* Cluster-wide z-pages
* Filtering of traces and metrics
* Aggregation of traces and metrics
* Decoration with meta-data from infrastructure provider (e.g. k8s master)
* much more ...

The collector also serves as a control plane for agents/clients by supplying
them updated configuration (e.g. trace sampling policies), and reporting
agent/client health information/inventory metadata to downstream exporters.

### <a name="collector-usage"></a>Usage

> It is recommended that you use the latest [release](https://github.com/census-instrumentation/opencensus-service/releases).

The collector can be run directly from sources, binary, or a Docker image.

The minimum Go version required for this project is Go 1.11.4.

1. Run from sources:
```shell
$ GO111MODULE=on go run github.com/census-instrumentation/opencensus-service/cmd/occollector
```
2. Run from binary (from the root of your repo):
```shell
$ make collector
$ ./bin/occollector_$($GOOS)
```
3. Build a Docker scratch image and use the appropriate Docker command for your scenario
(note: additional ports may be required depending on your receiver configuration):
```shell
$ make docker-collector
$ docker run --rm -it -p 55678:55678 occollector
```

It can be configured via command-line or config file:
```
OpenCensus Collector

Usage:
  occollector [flags]

Flags:
      --config string                Path to the config file
      --debug-processor              Flag to add a debug processor (combine with log level DEBUG to log incoming spans)
      --health-check-http-port int   Port on which to run the healthcheck http server. (default 13133)
  -h, --help                         help for occollector
      --log-level string             Output level of logs (TRACE, DEBUG, INFO, WARN, ERROR, FATAL) (default "INFO")
      --metrics-level string         Output level of telemetry metrics (NONE, BASIC, NORMAL, DETAILED) (default "BASIC")
      --metrics-port uint16          Port exposing collector telemetry. (default 8888)
      --receive-jaeger               Flag to run the Jaeger receiver (i.e.: Jaeger Collector), default settings: {ThriftTChannelPort:14267 ThriftHTTPPort:14268}
      --receive-oc-trace             Flag to run the OpenCensus trace receiver, default settings: {Port:55678} (default true)
      --receive-zipkin               Flag to run the Zipkin receiver, default settings: {Port:9411}
```

Sample configuration file:
```yaml
log-level: DEBUG

receivers:
  opencensus: {} # Runs OpenCensus receiver with default configuration

queued-exporters:
  jaeger-sender-test: # A friendly name for the exporter
    # num-workers is the number of queue workers that will be dequeuing batches and sending them out (default is 10)
    num-workers: 2 

    # queue-size is the maximum number of batches allowed in the queue at a given time (default is 5000)
    queue-size: 100

    # retry-on-failure indicates whether queue processor should retry span batches in case of processing failure (default is true)
    retry-on-failure: true

    # backoff-delay is the amount of time a worker waits after a failed send before retrying (default is 5 seconds)
    backoff-delay: 3s

    # sender-type is the type of sender used by this processor, the default is an invalid sender so it forces one to be specified
    sender-type: jaeger-thrift-http

    # configuration of the selected sender-type, in this example Jaeger jaeger-thrift-http. Which supports 3 settings:
    # collector-endpoint: address of Jaeger collector jaeger-thrift-http endpoint
    # headers: a map of any additional headers to be sent with each batch (e.g.: api keys, etc)
    # timeout: the timeout for the sender to consider the operation as failed
    jaeger-thrift-http:
      collector-endpoint: "http://svc-jaeger-collector:14268/api/traces"
      headers: { "x-header-key":"00000000-0000-0000-0000-000000000001" }
      timeout: 5s
```

[travis-image]: https://travis-ci.org/census-instrumentation/opencensus-service.svg?branch=master
[travis-url]: https://travis-ci.org/census-instrumentation/opencensus-service
[godoc-image]: https://godoc.org/github.com/census-instrumentation/opencensus-service?status.svg
[godoc-url]: https://godoc.org/github.com/census-instrumentation/opencensus-service
[gitter-image]: https://badges.gitter.im/census-instrumentation/lobby.svg
[gitter-url]: https://gitter.im/census-instrumentation/lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge
