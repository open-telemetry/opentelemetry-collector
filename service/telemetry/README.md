# ObsReport

## Description

ObsReport is a framework in order to provide internal telemetry accross components. ObsReport is used to create traces and metrics. 

## ObsReport struct

The `ObsReport` struct must be defined and contain:
- a [trace.Tracer](https://pkg.go.dev/go.opentelemetry.io/otel/trace@v1.31.0#Tracer): to generate traces
- a [metadata.TelemetryBuilder](https://pkg.go.dev/go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata#TelemetryBuilder): to generate metrics
- a [metric.MeasurementOption](https://pkg.go.dev/go.opentelemetry.io/otel/metric@v1.31.0#MeasurementOption): to add attributes on the telemetry

The ObsReport struct can also contain custom fields that will be used when generating the telemetry. This can be a `component.ID` to add component identifying attributes, a span name prefix to use when generating traces ...etc

## Configuration

The ObsReport struct will have a corresponding `ObsReportSettings` struct containing the fields necessary in order to create the ObsReport instance.

- The tracer which is stored within the `ObsReport` struct should be created through the TracerProvider in the service's `TelemetrySettings`. 
- The TelemetryBuilder which is stored within the `ObsReport` should be created via the auto generated `NewTelemetryBuilder` func, by providing the service's `TelemetrySettings`.
- The MeasurementOption which is stored within the `ObsReport` can be created via `go.opentelemetry.io/otel/metric` API with the relevant attributes.

## Interface

The ObsReport provides functions in order to generate the internal telemetry:
- `(or *ObsReport) StartTracesOp(ctx context.Context) context.Context`
- `(or *ObsReport) EndTracesOp(ctx context.Context, numSpans int, err error)`
- `(or *ObsReport) StartMetricsOp(ctx context.Context) context.Context`
- `(or *ObsReport) EndMetricsOp(ctx context.Context, numMetricPoints int, err error)`
- `(or *ObsReport) StartLogsOp(ctx context.Context) context.Context`
- `(or *ObsReport) EndLogsOp(ctx context.Context, numLogRecords int, err error)`

Each `Start<Signal>Op` should be called in conjunction to a `End<Signal>Op`. These typically encapsulate the work being done by the component. A span is created encapsulating the operation. The numSpans/numMetricPoints/numLogRecords is used to set as an attribute on the span, but also to generate metrics on on the sent(exporter)/accepted(receiver) and failed `<signal>`.
