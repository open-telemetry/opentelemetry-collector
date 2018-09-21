# OpenCensus service

OpenCensus service is an experimental component that collects and
exports from the OpenCensus instrumented processes available from the
same host machine.

Some frameworks and ecosystems are now providing out-of-the-box
instrumentation by using OpenCensus but the user is still expected
to register an exporter in order to export data. This is a problem
during an incident. Even though our users can benefit from having
more diagnostics data coming out of services already instrumented
with OpenCensus, they have to modify their code to register an
exporter and redeploy. Asking our users recompile and redeploy is
not an ideal at an incident time.

OpenCensus service is trying to eliminate this requirement.

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

## Future goals
* Providing cluster-wide collections and cluster-wide z-pages.
* Currently we provide no ways to push configuration changes such as the
default sampling rate. It might be a problem at troubleshooting time. 
We are planning to address this problem in the future.

## Usage

First, install ocagent if you haven't.

```
$ go get github.com/census-instrumentation/opencensus-service/cmd/ocagent
```

Create a config.yaml file in the current directory and modify
it with the exporter configuration. For example, following
configuration exports both to Stackdriver and Zipkin.


config.yaml:

```
stackdriver:
  project: "your-project-id"
  enableTraces: true

zipkin:
  endpoint: "http://localhost:9411/api/v2/spans"
```

Run the example application that collects traces and exports
to the daemon if it is running.

```
$ go run "$(go env GOPATH)/src/github.com/census-instrumentation/opencensus-service/example/main.go"
```

Run ocagent:

```
$ ocagent
```

You should be able to see the traces in Stackdriver and Zipkin.
If you stop the ocagent, example application will stop exporting.
If you run it again, it will start exporting again.

## OpenCensus Agent

### Architecture Overview

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

![agent-architecture](image/agent-architecture.png)

To support Agent, Library should have “agent exporters”, similar to the existing exporters to
other backends. There should be 3 separate agent exporters for tracing/stats/metrics
respectively. Agent exporters will be responsible for sending spans/stats/metrics and (possibly)
receiving configuration updates from Agent.

### Communication

Communication between Library and Agent should user a bi-directional gRPC stream. Library should
initiate the connection, since there’s only one dedicated port for Agent, while there could be
multiple processes with Library running.
By default, Agent is available on port 55678.

### Protocol Workflow

1. Library will try to directly establish connections for Config and Export streams.
2. As the first message in each stream, Library must sent its identifier. Each identifier should
uniquely identify Library within the VM/container. Identifier is no longer needed once the streams
are established.
3. If streams were disconnected and retries failed, the Library identifier would be considered
expired on Agent side. Library needs to start a new connection with a unique identifier
(MAY be different than the previous one).

### Implementation details of Agent Server

This section describes the in-process implementation details of OC-Agent.

![agent-implementation](image/agent-implementation.png)

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

#### Interceptors

Each interceptor can be connected with multiple instrumentation libraries. The
communication protocol between interceptors and libraries is the one we described in the
proto files (for example trace_service.proto). When a library opens the connection with the
corresponding interceptor, the first message it sends must have the `Node` identifier. The
interceptor will then cache the `Node` for each library, and `Node` is not required for
the subsequent messages from libraries.

#### Agent Core

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

#### Exporters

Once in a while, Agent Core will push `SpanProto` with `Node` to each exporter. After
receiving them, each exporter will translate `SpanProto` to the format supported by the
backend (e.g Jaeger Thrift Span), and then push them to corresponding backend or service.

## OpenCensus Collector

TODO: add content about Collector.
