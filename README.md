# OpenCensus Service

[![Build Status][travis-image]][travis-url]
[![GoDoc][godoc-image]][godoc-url]
[![Gitter chat][gitter-image]][gitter-url]

# Table of contents
- [Introduction](#introduction)
- [Deployment](#deploy)
- [Getting Started](#getting-started)
    - [Demo](#getting-started-demo)
    - [Kubernetes](#getting-started-k8s)
    - [Standalone](#getting-started-standalone)
- [Configuration](#config)
    - [Receivers](#config-receivers)
    - [Exporters](#config-exporters)
    - [Diagnostics](#config-diagnostics)
- [OpenCensus Agent](#opencensus-agent)
    - [Building binaries](#agent-building-binaries)
    - [Usage](#agent-usage)
- [OpenCensus Collector](#opencensus-collector)
    - [Global Tags](#global-tags)
    - [Intelligent Sampling](#tail-sampling)
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
[OpenCensus Agent](#opencensus-agent) and [OpenCensus Collector](#opencensus-collector). For the detailed design specs,
please see [DESIGN.md](DESIGN.md).

## <a name="deploy"></a>Deployment

The OpenCensus Service can be deployed in a variety of different ways. The OpenCensus
Agent can be deployed with the application either as a separate process, as a sidecar,
or via a Kubernetes daemonset. Typically, the OpenCensus Collector is deployed
separately as either a Docker container, VM, or Kubernetes pod.

![deployment-models](images/opencensus-service-deployment-models.png)

## <a name="getting-started"></a>Getting Started

### <a name="getting-started-demo"></a>Demo

Instructions for setting up an end-to-end demo environment can be found [here](https://github.com/census-instrumentation/opencensus-service/tree/master/demos/trace)

### <a name="getting-started-k8s"></a>Kubernetes

Apply the [sample YAML](example/k8s.yaml) file:

```shell
$ kubectl apply -f example/k8s.yaml
```

### <a name="getting-started-standalone"></a>Standalone

Create an Agent [configuration](#config) file based on the options described below. Please note the Agent
requires the `opencensus` receiver be enabled. By default, the Agent has no exporters configured.

Build the Agent, see [Building binaries](#agent-building-binaries),
and start it:

```shell
$ ./bin/ocagent_$(go env GOOS)
$ 2018/10/08 21:38:00 Running OpenCensus receiver as a gRPC service at "127.0.0.1:55678"
```

Create an Collector [configuration](#config) file based on the options described below. By default, the
Collector has the `opencensus` receiver enabled, but no exporters.

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

## <a name="config"></a>Configuration

The OpenCensus Service (both the Agent and Collector) is configured via a YAML file. In general, you need to
configure one or more receivers as well as one or more exporters. In addition, diagnostics can also be configured.

### <a name="config-receivers"></a>Receivers

A receiver is how you get data into the OpenCensus Service. One or more receivers can be configured. By default,
the `opencensus` receiver is enabled on the Collector and required as a defined receiver for the Agent.

A basic example of all available receivers is provided below. For detailed receiver configuration,
please see the [receiver README.md](receiver/README.md).
```yaml
receivers:
  opencensus:
    address: "127.0.0.1:55678"

  zipkin:
    address: "127.0.0.1:9411"

  jaeger:
    jaeger-thrift-tchannel-port: 14267
    jaeger-thrift-http-port: 14268

  prometheus:
    config:
      scrape_configs:
        - job_name: 'caching_cluster'
          scrape_interval: 5s
          static_configs:
            - targets: ['localhost:8889']
```

### <a name="config-exporters"></a>Exporters

An exporter is how you send data to one or more backends/destinations. One or more exporters can be configured.
By default, no exporters are configured on the OpenCensus Service (either the Agent or Collector).

A basic example of all available exporters is provided below. For detailed exporter configuration,
please see the [exporter README.md](exporter/exporterparser/README.md).
```yaml
exporters:
  opencensus:
    headers: {"X-test-header": "test-header"}
    compression: "gzip"
    endpoint: "127.0.0.1:55678"

  jaeger:
    collector_endpoint: "http://127.0.0.1:14268/api/traces"

  kafka:
    brokers: ["127.0.0.1:9092"]
    topic: "opencensus-spans"

  stackdriver:
    project: "your-project-id"
    enable_tracing: true

  zipkin:
    endpoint: "http://127.0.0.1:9411/api/v2/spans"

  aws-xray:
    region: "us-west-2"
    default_service_name: "verifiability_agent"
    version: "latest"
    buffer_size: 200
  honeycomb:
    write_key: "739769d7-e61c-42ec-82b9-3ee88dfeff43"
    dataset_name: "dc8_9"
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
docker run --rm -it -p 55678:55678 -p 55679:55679 \
    -v $(pwd)/ocagent-config.yaml:/conf/ocagent-config.yaml \
    --config=/conf/ocagent-config.yaml \
    ocagent:v1.0.0
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

### <a name="global-tags"></a> Global Tags

The collector also takes some global configurations that modify its behavior for all receivers / exporters. One of the configurations
available is to add Attributes or Tags to all spans passing through this collector. These additional tags can be configured to either overwrite
attributes if they already exists on the span, or respect the original values. An example of this is provided below.
```yaml
global:
  attributes:
    overwrite: true
    values:
      # values are key value pairs where the value can be an int, float, bool, or string
      some_string: "hello world"
      some_int: 1234
      some_float: 3.14159
      some_bool: false
```

### <a name="tail-sampling"></a>Intelligent Sampling

```yaml
sampling:
  mode: tail
  # amount of time from seeing the first span in a trace until making the sampling decision
  decision-wait: 10s
  # maximum number of traces kept in the memory
  num-traces: 10000
  policies:
    # user-defined policy name
    my-string-tag-filter:
      # exporters the policy applies to
      exporters:
        - jaeger
        - omnition
      policy: string-tag-filter
      configuration:
        tag: tag1
        values:
          - value1
          - value2
    my-numeric-tag-filter:
      exporters:
        - jaeger
        - omnition
      policy: numeric-tag-filter
      configuration:
        tag: tag1
        min-value: 0
        max-value: 100
```

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
$ docker run --rm -it -p 55678:55678 -p 8888:8888 \
    -v $(pwd)/occollector-config.yaml:/conf/occollector-config.yaml \
    --config=/conf/occollector-config.yaml \
    occollector
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
  opencensus: {} # Runs OpenCensus receiver with default configuration (default behavior)

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
