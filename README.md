# OpenCensus Service

[![Build Status][travis-image]][travis-url]
[![GoDoc][godoc-image]][godoc-url]
[![Gitter chat][gitter-image]][gitter-url]

# Table of contents
- [Introduction](#introduction)
- [Goals](#goals)
- [OpenCensus Agent](#opencensus-agent)
    - [Building binaries](#agent-building-binaries)
    - [Usage](#agent-usage)
    - [Configuration file](#agent-configuration-file)
        - [Exporters](#agent-config-exporters)
        - [Receivers](#agent-config-receivers)
            - [OpenCensus](#details-receivers-opencensus)
            - [Zipkin](#details-receivers-zipkin)
            - [Jaeger](#details-receivers-jaeger)
        - [End-to-end example](#agent-config-end-to-end-example)
    - [Diagnostics](#agent-diagnostics)
        - [zPages](#agent-zpages)
    - [Docker image](#agent-docker-image)
- [OpenCensus Collector](#opencensus-collector)
    - [Usage](#collector-usage)

## Introduction
OpenCensus Service is an experimental component that can collect traces
and metrics from processes instrumented by OpenCensus or other
monitoring/tracing libraries (Jaeger, Prometheus, etc.), do the
aggregation and smart sampling, and export traces and metrics
to monitoring/tracing backends.

Some frameworks and ecosystems are now providing out-of-the-box
instrumentation by using OpenCensus but the user is still expected
to register an exporter in order to export data. This is a problem
during an incident. Even though our users can benefit from having
more diagnostics data coming out of services already instrumented
with OpenCensus, they have to modify their code to register an
exporter and redeploy. Asking our users recompile and redeploy is
not an ideal at an incident time. In addition, currently users need
to decide which service backend they want to export to, before they
distribute their binary instrumented by OpenCensus.

OpenCensus Service is trying to eliminate these requirements. With
OpenCensus Service, users do not need to redeploy or restart their applications
as long as it has the OpenCensus Agent exporter. All they need to do is
just configure and deploy OpenCensus Service separately. OpenCensus Service
will then automatically collect traces and metrics and export to any
backend of users' choice.

Currently OpenCensus Service consists of two components,
[OpenCensus Agent](#opencensus-agent) and [OpenCensus Collector](#opencensus-collector).
High-level workflow:

![service-architecture](https://user-images.githubusercontent.com/10536136/46637070-65f05f80-cb0f-11e8-96e6-bc56468486b3.png)

For the detailed design specs, please see [DESIGN.md](DESIGN.md).

## Goals

* Allow enabling of configuring the exporters lazily. After deploying code,
optionally run a daemon on the host and it will read the
collected data and upload to the configured backend.
* Binaries can be instrumented without thinking about the exporting story.
Allows open source binary projects (e.g. web servers like Caddy or Istio Mixer)
to adopt OpenCensus without having to link any exporters into their binary.
* Easier to scale the exporter development. Not every language has to
implement support for each backend.
* Custom daemons containing only the required exporters compiled in can be created.

## OpenCensus Agent

### <a name="agent-building-binaries"></a>Building binaries

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

First, install ocagent if you haven't.

```shell
$ go get github.com/census-instrumentation/opencensus-service/cmd/ocagent
```

### <a name="agent-configuration-file"></a>Configuration file

Create a config.yaml file in the current directory and modify
it with the exporter and receiver configurations.


#### <a name="agent-config-exporters"></a>Exporters

For example, to allow trace exporting to Stackdriver and Zipkin:

```yaml
exporters:
    stackdriver:
        project: "your-project-id"
        enable_traces: true

    zipkin:
        endpoint: "http://localhost:9411/api/v2/spans"

    kafka:
        brokers:
            - "127.0.0.1:9092"
        topic: "opencensus-spans"
```

#### <a name="agent-config-receivers"></a>Receivers
Agent provides a couple of receivers that receive spans from instrumentation libraries.

#### <a name="details-receivers-opencensus"></a>OpenCensus

This receiver receives spans from OpenCensus instrumented applications and translates them into the internal span types that
are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "opencensus" and field "address".

For example:
```yaml
receivers:
    opencensus:
        address: "127.0.0.1:55678"
```

By default this receiver is ALWAYS started since it is the point of the "OpenCensus agent"

#### <a name="details-receivers-zipkin"></a>Zipkin

This receiver receives spans from Zipkin "/v2" API HTTP uploads and translates them into the internal span types that are then
sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "zipkin" and field "address".

For example:
```yaml
receivers:
    zipkin:
        address: "localhost:9411"
```

#### <a name="details-receivers-jaeger"></a>Jaeger

This receiver receives spans from Jaeger collector HTTP and Thrift uploads and translates them into the internal span types that are then
sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "receivers", subsection "jaeger" and fields "collector_http_port", "collector_thrift_port".

For example:
```yaml
receivers:
    jaeger
        collector_http_port: 14268
        collector_thrift_port: 14267
```

### <a name="agent-config-end-to-end-example"></a>Running an end-to-end example/demo

Run the example application that collects traces and exports them
to the daemon.

Create an [Agent configuration file](#agent-configuration-file) as described above.

Build the agent, see [Building binaries](#agent-building-binaries),
and start it:

```shell
$ ./bin/ocagent_$(go env GOOS)
$ 2018/10/08 21:38:00 Running OpenCensus receiver as a gRPC service at "127.0.0.1:55678"
```

Next run the demo application:

```shell
$ go run "$(go env GOPATH)/src/github.com/census-instrumentation/opencensus-service/example/main.go"
```

You should be able to see the traces in Stackdriver and Zipkin.
If you stop the ocagent, the example application will stop exporting.
If you run it again, exporting will resume.

### <a name="agent-diagnostics"></a>Diagnostics

To monitor the agent itself, we provide some diagnostic tools like:

#### <a name="agent-zpages"></a>zPages

We provide zPages for information on ocagent's internals, running by default on port `55679`.
These routes below contain the various diagnostic resources:

Resource|Route
---|---
RPC stats|/debug/rpcz
Trace information|/debug/tracez

The zPages configuration can be updated in the config.yaml file with fields:
* `disabled`: if set to true, won't run zPages
* `port`: by default is 55679, otherwise should be set to a value between 0 an 65535

For example
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

### <a name="agent-docker-image"></a>Docker image

For a Docker image for experimentation (based on Alpine) get your 
[Agent configuration file](#agent-configuration-file) as described above,
and run:

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

The collector is in its initial development stages. It can be run directly
from sources, binary, or a Docker image.

1. Run from sources:
```shell
$ go run github.com/census-instrumentation/opencensus-service/cmd/occollector
```
2. Run from binary (from the root of your repo):
```shell
$ make collector
$ ./bin/occollector_$($GOOS)
```
3. Build a Docker scratch image and use the appropria Docker command for your scenario:
```shell
$ make docker-collector
$ docker run --rm -it -p 55678:55678 occollector
```

4. It can be configured via command-line or config file:
```shell
OpenCensus Collector

Usage:
  occollector [flags]

Flags:
      --add-queued-processor   Flag to wrap one processor with the queued processor (flag will be remove soon, dev helper)
      --config string          Path to the config file
  -h, --help                   help for occollector
      --log-level string       Output level of logs (TRACE, DEBUG, INFO, WARN, ERROR, FATAL) (default "INFO")
      --noop-processor         Flag to add the no-op processor (combine with log level DEBUG to log incoming spans)
      --receive-jaeger         Flag to run the Jaeger receiver (i.e.: Jaeger Collector), default settings: {ThriftTChannelPort:14267 ThriftHTTPPort:14268}
      --receive-oc-trace       Flag to run the OpenCensus trace receiver, default settings: {Port:55678}
      --receive-zipkin         Flag to run the Zipkin receiver, default settings: {Port:9411}
```

[travis-image]: https://travis-ci.org/census-instrumentation/opencensus-service.svg?branch=master
[travis-url]: https://travis-ci.org/census-instrumentation/opencensus-service
[godoc-image]: https://godoc.org/github.com/census-instrumentation/opencensus-service?status.svg
[godoc-url]: https://godoc.org/github.com/census-instrumentation/opencensus-service
[gitter-image]: https://badges.gitter.im/census-instrumentation/lobby.svg
[gitter-url]: https://gitter.im/census-instrumentation/lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge
