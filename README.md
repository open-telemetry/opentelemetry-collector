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
```

#### <a name="agent-config-receivers"></a>Receivers
Agent provides a couple of receivers that receive spans from instrumentation libraries.

#### <a name="details-receivers-opencensus"></a>OpenCensus

This receiver receives spans from OpenCensus instrumented applications and translates them into the internal span types that
are then sent to the collector/exporters.

Its address can be configured in the YAML configuration file under section "intercpetors", subsection "opencensus" and field "address".

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

Its address can be configured in the YAML configuration file under section "intercpetors", subsection "zipkin" and field "address".

For example:
```yaml
receivers:
    zipkin:
        address: "localhost:9411"
```

### <a name="agent-config-end-to-end-example"></a>Running an end-to-end example/demo

Run the example application that collects traces and exports them
to the daemon.

Firstly run ocagent:

```shell
$ ocagent
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

With your configuration file from above in [Agent configuration file](#agent-configuration-file),
the Docker image can be created by running:

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

First, install the collector if you haven't.

```
$ go get github.com/census-instrumentation/opencensus-service/cmd/occollector
```

Create a config.yaml file in the current directory and modify
it with the collector exporter configuration.

config.yaml:

```
omnition:
  tenant: "your-api-key"
```

Next, install ocagent if you haven't.

```
$ go get github.com/census-instrumentation/opencensus-service/cmd/ocagent
```

Create a config.yaml file in the current directory and modify
it with the collector exporter configuration.

config.yaml:

```
collector:
  endpoint: "https://collector.local"
```

Run the example application that collects traces and exports
to the daemon if it is running.

```
$ go run "$(go env GOPATH)/src/github.com/census-instrumentation/opencensus-service/example/main.go"
```

Run ocagent:

```shell
$ ocagent
2018/10/08 21:38:00 Running OpenCensus receiver as a gRPC service at "127.0.0.1:55678"
```

You should be able to see the traces in the configured tracing backend.
If you stop the ocagent, example application will stop exporting.
If you run it again, it will start exporting again.

[travis-image]: https://travis-ci.org/census-instrumentation/opencensus-service.svg?branch=master
[travis-url]: https://travis-ci.org/census-instrumentation/opencensus-service
[godoc-image]: https://godoc.org/github.com/census-instrumentation/opencensus-service?status.svg
[godoc-url]: https://godoc.org/github.com/census-instrumentation/opencensus-service
[gitter-image]: https://badges.gitter.im/census-instrumentation/lobby.svg
[gitter-url]: https://gitter.im/census-instrumentation/lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge
