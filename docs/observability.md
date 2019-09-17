# OpenTelemetry Collector Observability

## Goal

The goal of this document is to have a comprehensive description of observability of the Collector and changes needed to achieve observability part of our [vision](vision.md).

## What Needs Observation

The following elements of the Collector need to be observable.

### Current Values

- Resource consumption: CPU, RAM (in the future also IO - if we implement persistent queues) and any other metrics that may be available to Go apps (e.g. garbage size, etc).

- Receiving data rate, broken down by receivers and by data type (traces/metrics).

- Exporting data rate, broken down by exporters and by data type (traces/metrics).

- Data drop rate due to throttling, broken down by data type.

- Data drop rate due to invalid data received, broken down by data type.

- Current throttling state: Not Throttled/Throttled by Downstream/Internally Saturated.

- Incoming connection count, broken down by receiver.

- Incoming connection rate (new connections per second), broken down by receiver.

- In-memory queue size (in bytes and in units). Note: measurements in bytes may be difficult / expensive to obtain and should be used cautiously.

- Persistent queue size (when supported).

- End-to-end latency (from receiver input to exporter output). Note that with multiple receivers/exporters we potentially have NxM data paths, each with different latency (plus different pipelines in the future), so realistically we should likely expose the average of all data paths (perhaps broken down by pipeline).

- Latency broken down by pipeline elements (including exporter network roundtrip latency for request/response protocols).

“Rate” values must reflect the average rate of the last 10 seconds. Rates must exposed in bytes/sec and units/sec (e.g. spans/sec).

Note: some of the current values and rates may be calculated as derivatives of cumulative values in the backend, so it is an open question if we want to expose them separately or no.

### Cumulative Values

- Total received data, broken down by receivers and by data type (traces/metrics).

- Total exported data, broken down by exporters and by data type (traces/metrics).

- Total dropped data due to throttling, broken down by data type.

- Total dropped data due to invalid data received, broken down by data type.

- Total incoming connection count, broken down by receiver.

- Uptime since start.

### Trace or Log on Events

We want to generate the following events (log and/or send as a trace with additional data):

- Collector started/stopped.

- Collector reconfigured (if we support on-the-fly reconfiguration).

- Begin dropping due to throttling (include throttling reason, e.g. local saturation, downstream saturation, downstream unavailable, etc).

- Stop dropping due to throttling.

- Begin dropping due to invalid data (include sample/first invalid data).

- Stop dropping due to invalid data.

- Crash detected (differentiate clean stopping and crash, possibly include crash data if available).

For begin/stop events we need to define an appropriate hysteresis to avoid generating too many events. Note that begin/stop events cannot be detected in the backend simply as derivatives of current rates, the events include additional data that is not present in the current value.

### Host Metrics

The service should collect host resource metrics in addition to service's own process metrics. This may help to understand that the problem that we observe in the service is induced by a different process on the same host.

## How We Expose Metrics/Traces

Collector configuration must allow specifying the target for own metrics/traces (which can be different from the target of collected data). The metrics and traces must be clearly tagged to indicate that they are service’s own metrics (to avoid conflating with collected data in the backend).

### Impact

We need to be able to assess the impact of these observability improvements on the core performance of the Collector.

### Configurable Level of Observability

Some of the metrics/traces can be high volume and may not be desirable to always observe. We should consider adding an observability verboseness “level” that allows configuring the Collector to send more or less observability data (or even finer granularity to allow turning on/off specific metrics).

The default level of observability must be defined in a way that has insignificant performance impact on the service.
