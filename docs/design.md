# OpenTelemetry Collector Architecture

This document describes the architecture design and implementation of the
OpenTelemetry Collector.

## Summary

OpenTelemetry Collector is an executable that can receive telemetry data, optionally process it, and export it further.

The Collector supports several popular open-source protocols for receiving and sending telemetry data as well as offering a pluggable architecture for adding more protocols.

Data receiving, processing, and exporting is done using [Pipelines](#pipelines). The Collector can be configured to have one or more pipelines. Each pipeline includes:

- a set of [Receivers](#receivers) that receive the data
- a series of optional [Processors](#processors) that get the data from receivers and process it
- a set of [Exporters](#exporters) which get the data from processors and send it further outside the Collector.

The same receiver can be included in multiple pipelines and multiple pipelines can include the same Exporter.

## Pipelines

A pipeline defines a path the data follows in the Collector starting from reception, then further processing or modification and finally exiting the Collector via exporters.

Pipelines can operate on 3 telemetry data types: traces, metrics, and logs. The data type is a property of the pipeline defined by its configuration. Receivers, processors, and exporters used in a pipeline must support the particular data type otherwise `ErrDataTypeIsNotSupported` will be reported when the configuration is loaded. A pipeline can be depicted the following way:

![Pipelines](images/design-pipelines.png)

There can be one or more receivers in a pipeline. Data from all receivers is pushed to the first processor, which performs a processing on it and then pushes it to the next processor (or it may drop the data, e.g. if it is a “sampling” processor) and so on until the last processor in the pipeline pushes the data to the exporters. Each exporter gets a copy of each data element. The last processor uses a `fanoutconsumer` to fan out the data to multiple exporters.

The pipeline is constructed during Collector startup based on pipeline definition in the configuration.

A pipeline configuration typically looks like this:

```yaml
service:
  pipelines: # section that can contain multiple subsections, one per pipeline
    traces:  # type of the pipeline
      receivers: [otlp, jaeger, zipkin]
      processors: [memory_limiter, batch]
      exporters: [otlp, jaeger, zipkin]
```

The above example defines a pipeline for “traces” type of telemetry data, with 3 receivers, 2 processors and 3 exporters.

For details of config file format see [this document](https://docs.google.com/document/d/1NeheFG7DmcUYo_h2vLtNRlia9x5wOJMlV4QKEK05FhQ/edit#).

### Receivers

Receivers typically listen on a network port and receive telemetry data. Usually one receiver is configured to send received data to one pipeline, however it is also possible to configure the same receiver to send the same received data to multiple pipelines. This can be done by simply listing the same receiver in the “receivers” key of several pipelines:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:4317

service:
  pipelines:
    traces:  # a pipeline of “traces” type
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [jaeger]
    traces/2:  # another pipeline of “traces” type
      receivers: [otlp]
      processors: [batch]
      exporters: [opencensus]
```

In the above example `otlp` receiver will send the same data to pipeline `traces` and to pipeline `traces/2`. (Note: the configuration uses composite key names in the form of `type[/name]` as defined in [this document](https://docs.google.com/document/d/1NeheFG7DmcUYo_h2vLtNRlia9x5wOJMlV4QKEK05FhQ/edit#)).

When the Collector loads this config the result will look like this (part of processors and exporters are omitted from the diagram for brevity):


![Receivers](images/design-receivers.png)

Important: when the same receiver is referenced in more than one pipeline the Collector will create only one receiver instance at runtime that will send the data to a fan out consumer, which in turn will send the data to the first processor of each pipeline. The data propagation from receiver to the fan out consumer and then to processors is done via a synchronous function call. This means that if one processor blocks the call the other pipelines that are attached to this receiver will be blocked from receiving the same data and the receiver itself will stop processing and forwarding newly received data.

### Exporters

Exporters typically forward the data they get to a destination on a network (but they can also send it elsewhere, e.g `logging` exporter writes the telemetry data to the logging destination).

The configuration allows to have multiple exporters of the same type, even in the same pipeline. For example one can have 2 `otlp` exporters defined each one sending to a different OTLP endpoint, e.g.:

```yaml
exporters:
  otlp/1:
    endpoint: example.com:4317
  otlp/2:
    endpoint: localhost:14317
```

Usually an exporter gets the data from one pipeline, however it is possible to configure multiple pipelines to send data to the same exporter, e.g.:

```yaml
exporters:
  jaeger:
    protocols:
      grpc:
        endpoint: localhost:14250

service:
  pipelines:
    traces:  # a pipeline of “traces” type
      receivers: [zipkin]
      processors: [memory_limiter]
      exporters: [jaeger]
    traces/2:  # another pipeline of “traces” type
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger]
```

In the above example `jaeger` exporter will get data from pipeline `traces` and from pipeline `traces/2`. When the Collector loads this config the result will look like this (part of processors and receivers are omitted from the diagram for brevity):


![Exporters](images/design-exporters.png)

### Processors

A pipeline can contain sequentially connected processors. The first processor gets the data from one or more receivers that are configured for the pipeline, the last processor sends the data to one or more exporters that are configured for the pipeline. All processors between the first and last receive the data strictly only from one preceding processor and send data strictly only to the succeeding processor.

Processors can transform the data before forwarding it (i.e. add or remove attributes from spans), they can drop the data simply by deciding not to forward it (this is for example how the `probabilisticsampler` processor works), they can also generate new data. This is how a `spanmetrics` processor can produce metrics for spans processed by the pipeline.

The same name of the processor can be referenced in the `processors` key of multiple pipelines. In this case the same configuration will be used for each of these processors however each pipeline will always get its own instance of the processor. Each of these processors will have its own state, the processors are never shared between pipelines. For example if `batch` processor is used in several pipelines each pipeline will have its own batch processor (although each batch processor will be configured exactly the same way if they reference the same key in the configuration). As an example, given the following configuration:

```yaml
processors:
  batch:
    send_batch_size: 10000
    timeout: 10s

service:
  pipelines:
    traces:  # a pipeline of “traces” type
      receivers: [zipkin]
      processors: [batch]
      exporters: [jaeger]
    traces/2:  # another pipeline of “traces” type
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
```

When the Collector loads this config the result will look like this:


![Processors](images/design-processors.png)

Note that each `batch` processor is an independent instance, although both are configured the same way, i.e. each have a `send_batch_size` of 10000.

## <a name="opentelemetry-agent"></a>Running as an Agent

On a typical VM/container, there are user applications running in some
processes/pods with OpenTelemetry Library (Library). Previously, Library did
all the recording, collecting, sampling and aggregation on traces/metrics/logs,
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

To resolve the issues above, you can run OpenTelemetry Collector as an Agent.
The Agent runs as a daemon in the VM/container and can be deployed independent
of Library. Once Agent is deployed and running, it should be able to retrieve
traces/metrics/logs from Library, export them to other backends. We MAY also
give Agent the ability to push configurations (e.g sampling probability) to
Library. For those languages that cannot do stats aggregation in process, they
should also be able to send raw measurements and have Agent do the aggregation.


![agent-architecture](images/design-collector-agent.png)

For developers/maintainers of other libraries: Agent can also
accept traces/metrics/logs from other tracing/monitoring libraries, such as
Zipkin, Prometheus, etc. This is done by adding specific receivers. See
[Receivers](#receivers) for details.

## <a name="opentelemetry-collector"></a>Running as a Gateway

The OpenTelemetry Collector can run as a Gateway instance and receives spans
and metrics exported by one or more Agents or Libraries, or by
tasks/agents that emit in one of the supported protocols. The Collector is
configured to send data to the configured exporter(s). The following figure
summarizes the deployment architecture:


![OpenTelemetry Collector Architecture](images/design-collector-service.png "OpenTelemetry Collector Architecture")

The OpenTelemetry Collector can also be deployed in other configurations, such
as receiving data from other agents or clients in one of the formats supported
by its receivers.
