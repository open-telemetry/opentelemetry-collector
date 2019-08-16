# OpenTelemetry Service Architecture

This document describes the architecture design and implementation of
OpenTelemetry Service.

## Summary

OpenTelemetry Service is an executable that allows to receive telemetry data, optionally transform it and send the data further.

The Service supports several popular open-source protocols for telemetry data receiving and sending as well as offering a pluggable architecture for adding more protocols.

Data receiving, transformation and sending is done using Pipelines. The Service can be configured to have one or more Pipeline. Each Pipeline includes a set of Receivers that receive the data, a series of optional Processors that get the data from receivers and transform it and a set of Exporters which get the data from the Processors and send it further outside the Service. The same receiver can feed data to multiple Pipelines and multiple pipelines can feed data into the same Exporter.

## Pipelines

Pipeline defines a path the data follows in the Service starting from reception, then further processing or modification and finally exiting the Service via exporters.

Pipelines can operate on 2 telemetry data types: traces and metrics. The data type the pipeline operates is a property of the pipeline defined by its configuration. A pipeline can be depicted the following way:

![Pipelines](images/design-pipelines.png)

There can be one or more receivers in a pipeline. Data from all receivers is pushed to the first processor, which performs a processing on it and then pushes it to the next processor (or it may drop the data, e.g. if it is a “sampling” processor) and so on until the last processor in the pipeline pushes the data to the exporters. Each exporter gets a copy of each data element. The last processor uses a multiconsumer to fanout the data to multiple exporters.

The pipeline is constructed during Service startup based on pipeline definition in the config file.

A pipeline configuration typically looks like this:

```yaml
pipelines: # section that can contain multiple subsections, one per pipeline
  traces:  # type of the pipeline 
    receivers: [opencensus, jaeger, zipkin]
    processors: [tags, tail-sampling, batch, queued-retry]
    exporters: [opencensus, jaeger, stackdriver, zipkin]
```

The above example defines a pipeline for “traces” type of telemetry data, with 3 receivers, 4 processors and 4 exporters. 

For details of config file format see [this document](https://docs.google.com/document/d/1GWOzV0H0RTN1adiwo7fTmkjfCATDDFGuOB4jp3ldCc8/edit#).

### Receivers

Receivers typically listen on a network port and receive telemetry data. Usually one receiver is configured to send received data to one pipeline, however it is also possible to configure the same receiver to send the same received data to multiple pipelines. This can be done by simply listing the same receiver in the “receivers” key of several pipelines:

```yaml
receivers:
  opencensus:
    endpoint: "127.0.0.1:55678”

pipelines:
  traces:  # a pipeline of “traces” type
    receivers: [opencensus]
    processors: [tags, tail-sampling, batch, queued-retry]
    exporters: [jaeger]
  traces/2:  # another pipeline of “traces” type
    receivers: [opencensus]
    processors: [batch]
    exporters: [opencensus]
```

In the above example “opencensus” receiver will send the same data to pipeline “traces” and to pipeline “traces/2”. (Note: the configuration uses composite key names in the form of `type[/name]` as defined in this [this document](https://docs.google.com/document/d/1GWOzV0H0RTN1adiwo7fTmkjfCATDDFGuOB4jp3ldCc8/edit#)).

When the Service loads this config the result will look like this (part of processors and exporters are omitted from the diagram for brevity):

![Receivers](images/design-receivers.png)

Important: when the same receiver is referenced in more than one pipeline the Service will create only one receiver instance at runtime that will send the data to fanoutprocessor which in turn will send the data to the first processor of each pipeline. The data propagation from receiver to fanoutprocessor and then to processors is via synchronous function call. This means that if one processor blocks the call the other pipelines that are attached to this receiver will be blocked from receiving the same data and the receiver itself will stop processing and forwarding newly received data.

### Exporters

Exporters typically forward the data they get to a destination on a network (but they can also send it elsewhere, e.g “logging” exporter writes the telemetry data to a local file). 

The configuration allows to have multiple exporters of the same type, even in the same pipeline. For example one can have 2 “jaeger” exporters defined each one sending to a different jaeger endpoint, e.g.:

```yaml
exporters:
  jaeger/1:
    protocols:
      grpc:
        endpoint: "example.com:14250”
  jaeger/2:
    protocols:
      grpc:
        endpoint: "127.0.0.1:14250”
```

Usually an exporter gets the data from one pipeline, however it is possible to configure multiple pipelines to send data to the same exporter, e.g.:

```yaml
exporters:
  jaeger:
    protocols:
      grpc:
        endpoint: "127.0.0.1:14250”

pipelines:
  traces:  # a pipeline of “traces” type
    receivers: [zipkin]
    processors: [tags, tail-sampling, batch, queued-retry]
    exporters: [jaeger]
  traces/2:  # another pipeline of “traces” type
    receivers: [opencensus]
    processors: [batch]
    exporters: [jaeger]
```

In the above example “jaeger” exporter will get data from pipeline “traces” and from pipeline “traces/2”. When the Service loads this config the result will look like this (part of processors and receivers are omitted from the diagram for brevity):

![Exporters](images/design-exporters.png)

### Processors

A pipeline can contain one or more sequentially connected processors. The first processor gets the data from one or more receivers that are configured for the pipeline, the last processor sends the data to one or more exporters that are configured for the pipeline. All processors between the first and last receive the data strictly only from one preceding processor and send data strictly only to the succeeding processor.

Processors can transform the data before forwarding it (i.e. add or remove attributes from spans), they can drop the data simply by deciding not to forward it (this is for example how “sampling” processor works), they can also generate new data (this is how for example how a “persistent-queue” processor can work after Service restarts by reading previously saved data from a local file and forwarding it on the pipeline).

The same name of the processor can be referenced in the “processors” key of multiple pipelines. In this case the same configuration will be used for each of these processors however each pipeline will always gets its own instance of the processor. Each of these processors will have its own state, the processors are never shared between pipelines. For example if “queued-retry” processor is used several pipelines each pipeline will have its own queue (although the queues will be configured exactly the same way if the reference the same key in the config file). As an example, given the following config:

```yaml
processors:
  queued-retry:
    size: 50
    per-exporter: true
    enabled: true

pipelines:
  traces:  # a pipeline of “traces” type
    receivers: [zipkin]
    processors: [queued-retry]
    exporters: [jaeger]
  traces/2:  # another pipeline of “traces” type
    receivers: [opencensus]
    processors: [queued-retry]
    exporters: [opencensus]
```

When the Service loads this config the result will look like this:

![Processors](images/design-processors.png)

Note that each “queued-retry” processor is an independent instance, although both are configured the same way, i.e. each have a size of 50.

## <a name="opentelemetry-agent"></a>Running as an Agent

On a typical VM/container, there are user applications running in some
processes/pods with OpenTelemetry Library (Library). Previously, Library did
all the recording, collecting, sampling and aggregation on spans/stats/metrics,
and exported them to other persistent storage backends via the Library
exporters, or displayed them on local zpages. This pattern has several
drawbacks, for example:

1. For each OpenTelemetry Library, exporters/zpages need to be re-implemented
   in native languages.
2. In some programming languages (e.g Ruby, PHP), it is difficult to do the
   stats aggregation in process.
3. To enable exporting OpenTelemetry spans/stats/metrics, application users
   need to manually add library exporters and redeploy their binaries. This is
   especially difficult when there’s already an incident and users want to use
   OpenTelemetry to investigate what’s going on right away.
4. Application users need to take the responsibility in configuring and
   initializing exporters. This is error-prone (e.g they may not set up the
   correct credentials\monitored resources), and users may be reluctant to
   “pollute” their code with OpenTelemetry.

To resolve the issues above, you can run OpenTelemetry Service as an Agent.
The Agent runs as a daemon in the VM/container and can be deployed independent
of Library. Once Agent is deployed and running, it should be able to retrieve
spans/stats/metrics from Library, export them to other backends. We MAY also
give Agent the ability to push configurations (e.g sampling probability) to
Library. For those languages that cannot do stats aggregation in process, they
should also be able to send raw measurements and have Agent do the aggregation.

TODO: update the diagram below.

![agent-architecture](https://user-images.githubusercontent.com/10536136/48792454-2a69b900-eca9-11e8-96eb-c65b2b1e4e83.png)

For developers/maintainers of other libraries: Agent can also 
accept spans/stats/metrics from other tracing/monitoring libraries, such as
Zipkin, Prometheus, etc. This is done by adding specific receivers. See
[Receivers](#receivers) for details.

## <a name="opentelemetry-collector"></a>Running as a Collector

The OpenTelemetry Service can run as a standalone Collector instance and receives spans
and metrics exported by one or more Agents or Libraries, or by
tasks/agents that emit in one of the supported protocols. The Collector is
configured to send data to the configured exporter(s). The following figure
summarizes the deployment architecture:

TODO: update the diagram below.

![OpenTelemetry Collector Architecture](https://user-images.githubusercontent.com/10536136/46637070-65f05f80-cb0f-11e8-96e6-bc56468486b3.png "OpenTelemetry Collector Architecture")

The OpenTelemetry Collector can also be deployed in other configurations, such
as receiving data from other agents or clients in one of the formats supported
by its receivers.
