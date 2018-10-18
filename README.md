# OpenCensus Service

[![Build Status][travis-image]][travis-url]
[![GoDoc][godoc-image]][godoc-url]
[![Gitter chat][gitter-image]][gitter-url]

# Table of contents
- [Introduction](#introduction)
- [Goals](#goals)
- [OpenCensus Agent](#opencensus-agent)
    - [Architecture overview](#agent-architecture-overview)
    - [Communication](#agent-communication)
    - [Protocol Workflow](#agent-protocol-workflow)
    - [Implementation details of Agent server](#agent-implementation-details-of-agent-server)
        - [Interceptors](#agent-impl-interceptors)
        - [Agent Core](#agent-impl-agent-core)
        - [Exporters](#agent-impl-exporters)
    - [Building binaries](#agent-building-binaries)
    - [Usage](#agent-usage)
    - [Configuration file](#agent-configuration-file)
        - [Exporters](#agent-config-exporters)
        - [Interceptors](#agent-config-interceptors)
        - [End-to-end example](#agent-config-end-to-end-example)
    - [Docker image](#agent-docker-image)
- [OpenCensus Collector](#opencensus-collector)
    - [Architecture overview](#collector-architecture-overview)
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

### <a name="agent-architecture-overview"></a>Architecture Overview

On a typical VM/container, there are user applications running in some processes/pods with
OpenCensus Library (Library). Previously, Library did all the recording, collecting, sampling and
aggregation on spans/stats/metrics, and exported them to other persistent storage backends via the
Library exporters, or displayed them on local zpages. This pattern has several drawbacks, for
example:

1. For each OpenCensus Library, exporters/zpages need to be re-implemented in native languages.
2. In some programming languages (e.g Ruby, PHP), it is difficult to do the stats aggregation in
process.
3. To enable exporting OpenCensus spans/stats/metrics, application users need to manually add
library exporters and redeploy their binaries. This is especially difficult when there’s already
an incident and users want to use OpenCensus to investigate what’s going on right away.
4. Application users need to take the responsibility in configuring and initializing exporters.
This is error-prone (e.g they may not set up the correct credentials\monitored resources), and
users may be reluctant to “pollute” their code with OpenCensus.

To resolve the issues above, we are introducing OpenCensus Agent (Agent). Agent runs as a daemon
in the VM/container and can be deployed independent of Library. Once Agent is deployed and
running, it should be able to retrieve spans/stats/metrics from Library, export them to other
backends. We MAY also give Agent the ability to push configurations (e.g sampling probability) to
Library. For those languages that cannot do stats aggregation in process, they should also be
able to send raw measurements and have Agent do the aggregation.

For developers/maintainers of other libraries: Agent can also be extended to accept spans/stats/metrics from
other tracing/monitoring libraries, such as Zipkin, Prometheus, etc. This is done by adding specific
interceptors. See [Interceptors](#interceptors) for details.

![agent-architecture](https://user-images.githubusercontent.com/10536136/46707027-dae59700-cbec-11e8-80ae-f16a8bdbacf4.png)

To support Agent, Library should have “agent exporters”, similar to the existing exporters to
other backends. There should be 3 separate agent exporters for tracing/stats/metrics
respectively. Agent exporters will be responsible for sending spans/stats/metrics and (possibly)
receiving configuration updates from Agent.

### <a name="agent-communication"></a>Communication

Communication between Library and Agent should use a bi-directional gRPC stream. Library should
initiate the connection, since there’s only one dedicated port for Agent, while there could be
multiple processes with Library running.
By default, Agent is available on port 55678.

### <a name="agent-protocol-workflow"></a>Protocol Workflow

1. Library will try to directly establish connections for Config and Export streams.
2. As the first message in each stream, Library must send its identifier. Each identifier should
uniquely identify Library within the VM/container. If there is no identifier in the first message,
Agent should drop the whole message and return an error to the client. In addition, the first
message MAY contain additional data (such as `Span`s). As long as it has a valid identifier
assoicated, Agent should handle the data properly, as if they were sent in a subsequent message.
Identifier is no longer needed once the streams are established.
3. If streams were disconnected and retries failed, the Library identifier would be considered
expired on Agent side. Library needs to start a new connection with a unique identifier
(MAY be different than the previous one).

### <a name="agent-protocol-implementation-details-of-agent-server"></a>Implementation details of Agent Server

This section describes the in-process implementation details of OC-Agent.

![agent-implementation](https://user-images.githubusercontent.com/10536136/46637069-65f05f80-cb0f-11e8-99e1-3cbb368e9dbc.png)

Note: Red arrows represent RPCs or HTTP requests. Black arrows represent local method
invocations.

The Agent consists of three main parts:

1. The interceptors of different instrumentation libraries, such as OpenCensus, Zipkin,
Istio Mixer, Prometheus client, etc. Interceptors act as the “frontend” or “gateway” of
Agent. In addition, there MAY be one special receiver for receiving configuration updates
from outside.
2. The core Agent module. It acts as the “brain” or “dispatcher” of Agent.
3. The exporters to different monitoring backends or collector services, such as
Omnition Collector, Stackdriver Trace, Jaeger, Zipkin, etc.

#### <a name="agent-impl-interceptors"></a>Interceptors

Each interceptor can be connected with multiple instrumentation libraries. The
communication protocol between interceptors and libraries is the one we described in the
proto files (for example trace_service.proto). When a library opens the connection with the
corresponding interceptor, the first message it sends must have the `Node` identifier. The
interceptor will then cache the `Node` for each library, and `Node` is not required for
the subsequent messages from libraries.

#### <a name="agent-impl-core"></a>Agent Core

Most functionalities of Agent are in Agent Core. Agent Core's responsibilies include:

1. Accept `SpanProto` from each interceptor. Note that the `SpanProto`s that are sent to
Agent Core must have `Node` associated, so that Agent Core can differentiate and group
`SpanProto`s by each `Node`.
2. Store and batch `SpanProto`s.
3. Augment the `SpanProto` or `Node` sent from the interceptor.
For example, in a Kubernetes container, Agent Core can detect the namespace, pod id
and container name and then add them to its record of Node from interceptor
4. For some configured period of time, Agent Core will push `SpanProto`s (grouped by
`Node`s) to Exporters.
5. Display the currently stored `SpanProto`s on local zPages.
6. MAY accept the updated configuration from Config Receiver, and apply it to all the
config service clients.
7. MAY track the status of all the connections of Config streams. Depending on the
language and implementation of the Config service protocol, Agent Core MAY either
store a list of active Config streams (e.g gRPC-Java), or a list of last active time for
streams that cannot be kept alive all the time (e.g gRPC-Python).

#### <a name="agent-impl-exporters"></a>Exporters

Once in a while, Agent Core will push `SpanProto` with `Node` to each exporter. After
receiving them, each exporter will translate `SpanProto` to the format supported by the
backend (e.g Jaeger Thrift Span), and then push them to corresponding backend or service.

### <a name="building-binaries"></a>Building binaries

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
it with the exporter and interceptor configurations.


#### <a name="agent-config-exporters"></a>Exporters

For example, to allow trace exporting to Stackdriver and Zipkin:

```yaml
stackdriver:
  project: "your-project-id"
  enable_traces: true

zipkin:
  endpoint: "http://localhost:9411/api/v2/spans"
```

#### <a name="agent-config-interceptors"></a>Interceptors

To modify the address that the OpenCensus interceptor runs on, please use the
YAML field name `opencensus_interceptor` and it takes fields like `address`.
For example:

```yaml
opencensus_interceptor:
    address: "localhost:55678"
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

### <a name="collector-architecture-overview"></a>Architecture Overview

The OpenCensus Collector runs as a standalone instance and receives spans and
metrics exporterd by one or more OpenCensus Agents or Libraries, or by
tasks/agents that emit in one of the supported protocols. The Collector is
configured to send data to the configured exporter(s). The following figure
summarizes the deployment architecture:

![OpenCensus Collector Architecture](https://user-images.githubusercontent.com/10536136/46637070-65f05f80-cb0f-11e8-96e6-bc56468486b3.png "OpenCensus Collector Architecture")

The OpenCensus Collector can also be deployed in other configurations, such as
receiving data from other agents or clients in one of the formats supported by
its interceptors.

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
2018/10/08 21:38:00 Running OpenCensus interceptor as a gRPC service at "127.0.0.1:55678"
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
