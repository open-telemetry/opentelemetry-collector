# OpenTelemetry Collector

*IMPORTANT:* This is a pre-released version of the OpenTelemetry Collector.
For now, please use the [OpenCensus Service](https://github.com/census-instrumentation/opencensus-service).

[![Build Status][travis-image]][travis-url]
[![GoDoc][godoc-image]][godoc-url]
[![Gitter chat][gitter-image]][gitter-url]
[![Coverage Status][codecov-image]][codecov-url]

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
    - [Global Attributes](#global-attributes)
    - [Sampling](#sampling)
- [Usage](#usage)

## Introduction

The OpenTelemetry Collector can collect traces and metrics from processes
instrumented by OpenTelemetry or other monitoring/tracing libraries (Jaeger,
Prometheus, etc.), handles aggregation and smart sampling, and export traces
and metrics to one or more monitoring/tracing backends.

Some frameworks and ecosystems are now providing out-of-the-box instrumentation
by using OpenTelemetry, but the user is still expected to register an exporter
in order to export data. This is a problem during an incident. Even though our
users can benefit from having more diagnostics data coming out of services
already instrumented with OpenTelemetry, they have to modify their code to
register an exporter and redeploy. Asking our users recompile and redeploy is
not an ideal at an incident time. In addition, currently users need to decide
which service backend they want to export to, before they distribute their
binary instrumented by OpenTelemetry.

The OpenTelemetry Collector is trying to eliminate these requirements. With the
OpenTelemetry Collector, users do not need to redeploy or restart their
applications as long as it has the OpenTelemetry exporter. All they need to do
is just configure and deploy the OpenTelemetry Collector separately. The
OpenTelemetry Collector will then automatically collect traces and metrics and
export to any backend of users' choice.

Currently the OpenTelemetry Collector consists of a single binary and two
deployment methods:

1. Agent running with the application or on the same host as the application
2. Collector running as a standalone application

For the detailed design specs, please see [design.md](docs/design.md).

For OpenTelemetry Collector performance specs, please see [performance.md](docs/performance.md).

For the future vision of OpenTelemetry Collector please see [vision.md](docs/vision.md).

## <a name="deploy"></a>Deployment

The OpenTelemetry Collector can be deployed in a variety of different ways
depending on requirements. The Agent can be deployed with the application
either as a separate process, as a sidecar, or via a Kubernetes daemonset. The
Collector is deployed as a separate application as either a Docker container,
VM, or Kubernetes pod.

While the Agent and Collector share the same binary, the configuration between
the two may differ depending on requirements (e.g. queue size and feature-set
enabled).

![deployment-models](https://i.imgur.com/Tj384ap.png)

## <a name="getting-started"></a>Getting Started

### <a name="getting-started-demo"></a>Demo

Instructions for setting up an end-to-end demo environment can be found [here](https://github.com/open-telemetry/opentelemetry-service/tree/master/examples/demo)

### <a name="getting-started-k8s"></a>Kubernetes

Apply the [sample YAML](examples/k8s.yaml) file:

```shell
$ kubectl apply -f example/k8s.yaml
```

### <a name="getting-started-standalone"></a>Standalone

Create an Agent [configuration](#config) file based on the options described
below. By default, the Agent has the `opencensus` receiver enabled, but no
exporters configured.

Build the Agent and start it with the example configuration:

```shell
$ ./bin/$(go env GOOS)/otelcol  --config ./examples/demo/otel-agent-config.yaml
$ 2018/10/08 21:38:00 Running OpenTelemetry receiver as a gRPC service at "localhost:55678"
```

Create an Collector [configuration](#config) file based on the options
described below. By default, the Collector has the `opencensus` receiver
enabled, but no exporters configured.

Build the Collector and start it with the example configuration:

```shell
$ make otelcol
$ ./bin/$($GOOS)/otelcol --config ./examples/demo/otel-collector-config.yaml
```

Run the demo application:

```shell
$ go run "$(go env GOPATH)/src/github.com/open-telemetry/opentelemetry-service/examples/main.go"
```

You should be able to see the traces in your exporter(s) of choice. If you stop
the `otelcol`, the example application will stop exporting. If you run it again,
exporting will resume.

## <a name="config"></a>Configuration

The OpenTelemetry Collector is configured via a YAML file. 
In general, at least one enabled receiver and one enabled exporter
needs to be configured.

*Note* This documentation is still in progress. For any questions, please reach out in the
[OpenTelemetry Gitter](https://gitter.im/open-telemetry/opentelemetry-service) or
refer to the [issues page](https://github.com/open-telemetry/opentelemetry-service/issues).

There are four main parts to a config:
```yaml
receivers:
  ...
exporters:
  ...
processors:
  ...
pipelines:
  ...
```

### <a name="config-receivers"></a>Receivers

A receiver is how data gets into OpenTelemetry Collector. One or more receivers
must be configured.

A basic example of all available receivers is provided below. For detailed
receiver configuration, please see the [receiver
README.md](receiver/README.md).

```yaml
receivers:
  opencensus:
    address: "localhost:55678"

  zipkin:
    address: "localhost:9411"

  jaeger:
    jaeger-thrift-tchannel-port: 14267
    jaeger_thrift_http-port: 14268

  prometheus:
    config:
      scrape_configs:
        - job_name: 'caching_cluster'
          scrape_interval: 5s
          static_configs:
            - targets: ['localhost:8889']
```

### <a name="config-exporters"></a>Exporters

An exporter is how you send data to one or more backends/destinations. One or
more exporters can be configured. By default, no exporters are configured on
the OpenTelemetry Collector.

A basic example of all available exporters is provided below. For detailed
exporter configuration, please see the [exporter
README.md](exporter/README.md).

```yaml
exporters:
  opencensus:
    headers: {"X-test-header": "test-header"}
    compression: "gzip"
    cert_pem_file: "server-ca-public.pem" # optional to enable TLS
    endpoint: "localhost:55678"
    reconnection_delay: 2s

  jaeger:
    collector_endpoint: "http://localhost:14268/api/traces"

  zipkin:
    endpoint: "http://localhost:9411/api/v2/spans"
```

### <a name="config-pipelines"></a>Pipelines
Pipelines can be of two types:

- metrics: collects and processes metrics data.
- traces: collects and processes trace data.

A pipeline consists of a set of receivers, processors, and exporters. Each
receiver/processor/exporter must be specified in the configuration to be
included in a pipeline and each receiver/processor/exporter can be used in more
than one pipeline.

*Note:* For processor(s) referenced in multiple pipelines, each pipeline will
get a separate instance of that processor(s). This is in contrast to
receiver(s)/exporter(s) referenced in multiple pipelines, one instance of
a receiver/exporter is reference by all the pipelines.

The following is an example pipeline configuration. For more information, refer
to [pipeline documentation](docs/pipelines.md)
```yaml
pipelines:
  traces:
    receivers: [examplereceiver]
    processors: [exampleprocessor]
    exporters: [exampleexporter]

```

### <a name="config-diagnostics"></a>Diagnostics

zPages is provided for monitoring running by default on port ``55679``.
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


### <a name="global-attributes"></a> Global Attributes
**TODO** Remove this once processors have been documented since that handles
these features now.

The OpenTelemetry Collector also takes some global configurations that modify its
behavior for all receivers / exporters. This configuration is typically applied
on the Collector, but could also be added to the Agent.

1. Add Attributes to all spans passing through this collector. These additional
   attributes can be configured to either overwrite existing keys if they
   already exist on the span, or respect the original values.
2. The key of each attribute can also be mapped to different strings using the
   `key-mapping` configuration. The key matching is case sensitive.

An example using these configurations of this is provided below.

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
    key-mapping:
      # key-mapping is used to replace the attribute key with different keys
      - key: servertracer.http.responsecode
        replacement: http.status_code
      - key:  servertracer.http.responsephrase
        replacement: http.message
        overwrite: true # replace attribute key even if the replacement string is already a key on the span attributes
        keep: true # keep the attribute with the original key
```

### <a name="sampling"></a>Sampling

Sampling can also be configured on the OpenTelemetry Collector. Both head-based and
tail-based sampling are supported. Either the Agent or the Collector may enable
head-based sampling. Tail sampling must be configured on the Collector as it
requires all spans for a given trace to make a sampling decision.

#### Head-based Example

```yaml
sampling:
  # mode indicates if the sampling is head or tail based. For probabilistic the mode is head-based.
  mode: head
  policies:
    # section below defines a probabilistic trace sampler based on hashing the trace ID associated to
    # each span and sampling the span according to the given spans.
    probabilistic:
      configuration:
        # sampling_percentage is the percentage of sampling to be applied to all spans, unless their service is specified
        # on sampling_percentage.
        sampling_percentage: 5
        # hash_seed allows choosing the seed for the hash function used in the trace sampling. This is important when
        # multiple layers of collectors are being used with head sampling, in such scenarios make sure to
        # choose different seeds for each layer.
        hash_seed: 1
```

#### Tail-based Example

```yaml
sampling:
  mode: tail
  # amount of time from seeing the first span in a trace until making the sampling decision
  decision_wait: 10s
  # maximum number of traces kept in the memory
  num_traces: 10000
  policies:
    # user-defined policy name
    my_string_attribute_filter:
      # exporters the policy applies to
      exporters:
        - jaeger
      policy: string_attribute_filter
      configuration:
        key: key1
        values:
          - value1
          - value2
    my-numeric_attribute-filter:
      exporters:
        - zipkin
      policy: numeric_attribute-filter
      configuration:
        key: key1
        min_value: 0
        max_value: 100
```

> Note that an exporter can only have a single sampling policy today.
## <a name="collector-usage"></a>Usage

> It is recommended that you use the latest [release](https://github.com/open-telemetry/opentelemetry-service/releases).

The OpenTelemetry Collector can be run directly from sources, binary, or a Docker
image. If you are planning to run from sources or build on your machine start
by cloning the repo using `go get -d
github.com/open-telemetry/opentelemetry-service`.

The minimum Go version required for this project is Go 1.12.5.

1. Run from sources:
```shell
$ GO111MODULE=on go run github.com/open-telemetry/opentelemetry-service/cmd/otelcol --help
```
2. Run from binary (from the root of your repo):
```shell
$ make otelcol
$ ./bin/$($GOOS)/otelcol
```
3. Build a Docker scratch image and use the appropriate Docker command for your
   scenario (note: additional ports may be required depending on your receiver
   configuration):
```shell
$ make docker-otelcol
$ docker run \
    --rm \
    --interactive \
    --tty \
    --publish 55678:55678 --publish 55679:55679 --publish 8888:8888 \
    --volume $(pwd)/otel-collector-config.yaml:/conf/otelcol-config.yaml \
    otelcol \
    --config=/conf/otelcol-config.yaml
```

It can be configured via command-line or config file:
```
OpenTelemetry Collector

Usage:
  otelcol [flags]

Flags:
      --config string               Path to the config file
  -h, --help                        help for otelcol
      --log-level string            Output level of logs (TRACE, DEBUG, INFO, WARN, ERROR, FATAL) (default "INFO")
      --mem-ballast-size-mib uint   Flag to specify size of memory (MiB) ballast to set. Ballast is not used when this is not specified. default settings: 0
      --metrics-level string        Output level of telemetry metrics (NONE, BASIC, NORMAL, DETAILED) (default "BASIC")
      --metrics-port uint           Port exposing collector telemetry. (default 8888)
```

Sample configuration file:
```yaml
log-level: DEBUG

receivers:
  opencensus: {} # Runs OpenCensus receiver with default configuration (default behavior).

queued-exporters:
  jaeger-sender-test: # A friendly name for the exporter
    # num_workers is the number of queue workers that will be dequeuing batches and sending them out (default is 10)
    num_workers: 2

    # queue_size is the maximum number of batches allowed in the queue at a given time (default is 5000)
    queue_size: 100

    # retry_on_failure indicates whether queue processor should retry span batches in case of processing failure (default is true)
    retry_on_failure: true

    # backoff_delay is the amount of time a worker waits after a failed send before retrying (default is 5 seconds)
    backoff_delay: 3s

    # sender-type is the type of sender used by this processor, the default is an invalid sender so it forces one to be specified
    sender-type: jaeger_thrift_http

    # configuration of the selected sender-type, in this example Jaeger jaeger_thrift_http. Which supports 3 settings:
    # collector-endpoint: address of Jaeger collector jaeger_thrift_http endpoint
    # headers: a map of any additional headers to be sent with each batch (e.g.: api keys, etc)
    # timeout: the timeout for the sender to consider the operation as failed
    jaeger_thrift_http:
      collector-endpoint: "http://svc-jaeger-collector:14268/api/traces"
      headers: { "x-header-key":"00000000-0000-0000-0000-000000000001" }
      timeout: 5s
```

[travis-image]: https://travis-ci.org/open-telemetry/opentelemetry-service.svg?branch=master
[travis-url]: https://travis-ci.org/open-telemetry/opentelemetry-service
[godoc-image]: https://godoc.org/github.com/open-telemetry/opentelemetry-service?status.svg
[godoc-url]: https://godoc.org/github.com/open-telemetry/opentelemetry-service
[gitter-image]: https://badges.gitter.im/open-telemetry/opentelemetry-service.svg
[gitter-url]: https://gitter.im/open-telemetry/opentelemetry-service?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge
[codecov-image]: https://codecov.io/gh/open-telemetry/opentelemetry-service/branch/master/graph/badge.svg
[codecov-url]: https://codecov.io/gh/open-telemetry/opentelemetry-service/branch/master/
