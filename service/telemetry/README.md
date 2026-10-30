# Service Telemetry

Package `telemetry` defines the interface the Collector service uses to create
its internal telemetry providers (logger, meter and tracer). The default
implementation, based on `otelconf`, lives in the `otelconftelemetry`
subpackage.

Components do not use this package directly. They receive a logger, meter and
tracer through `component.TelemetrySettings` and report their own telemetry
using the `ObsReport` helpers described below.

## Instrumenting components with obsreport

`obsreport` is the framework components use to emit consistent internal
telemetry (metrics and traces) about the data they handle. How much of it you
need to wire up depends on the component type:

| Component | What you need to do |
|-----------|---------------------|
| Receiver  | Create a `receiverhelper.ObsReport` and wrap each receive operation in a `Start<Signal>Op` / `End<Signal>Op` pair. |
| Exporter  | Nothing. `exporterhelper` instruments exports for you. |
| Scraper   | Nothing. `scraperhelper` instruments scrapes for you. |

### Receivers

Create one `ObsReport` per receiver instance, using keyed fields:

```go
obsreport, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
	ReceiverID:             set.ID,
	Transport:              "grpc", // or "http", "tcp", ...
	ReceiverCreateSettings: set,
})
if err != nil {
	return err
}
```

Set `LongLivedCtx: true` if the context you pass in outlives a single receive
operation, for example a gRPC stream that carries many batches.

For each request received from a client, call the `Start`/`End` pair that
matches the signal: `StartTracesOp`/`EndTracesOp`, `StartMetricsOp`/`EndMetricsOp`
or `StartLogsOp`/`EndLogsOp`. Pass the context returned by `Start*Op` to the
next consumer and to `End*Op`:

```go
func (r *Receiver) Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	td := req.Traces()
	numSpans := td.SpanCount()
	if numSpans == 0 {
		return ptraceotlp.NewExportResponse(), nil
	}

	ctx = r.obsreport.StartTracesOp(ctx)
	err := r.nextConsumer.ConsumeTraces(ctx, td)
	r.obsreport.EndTracesOp(ctx, "protobuf", numSpans, err)
	// ...
}
```

`End*Op` takes the data format name, the number of items received, and the
error returned by the next consumer. Always call it, including on error paths.
See the OTLP receiver (`receiver/otlpreceiver`) for a complete example.

### Guidelines

1. Count items in the native format of the receiver, not the Collector's
   internal representation, so numbers can be reconciled with clients and
   backends. For example, count spans, metric data points and log records.
2. Report data loss only when the component itself removes data from the
   pipeline. A receiver returning an error to a client is a refused request
   (the client will usually retry), not data loss.
3. Data intentionally filtered out should have its own metric and must not be
   reported as dropped.

### Metrics

The metrics emitted are defined in the helper's `metadata.yaml` and documented
in the generated `documentation.md` next to it, for example
`receiver/receiverhelper/documentation.md`.
