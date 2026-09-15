# Internal queue and batch telemetry

This package connects queue and batch operations to component-specific metrics.
It is internal because the callbacks are concrete implementation details.
External users should not depend on this arrangement.

Exporterhelper creates one `ObsMetrics` value backed by exporter instruments.
It gives the same value to its queue and sender.

`ObsMetrics` contains a `QueueMetrics` part and a `SendMetrics` part. The queue
receives the queue part. The sender receives the send part.

The queuebatch processor creates an `ObsMetrics` value backed by processor
instruments. It calls `ConfigWithObsMetrics` to attach the value to its
`component.Config`. This avoids adding an internal option to the public
exporterhelper API.

Each exporterhelper constructor removes the wrapper immediately. The original
config continues through normal validation. The callbacks become an internal
exporterhelper option.

Exporterhelper owns the callbacks after its options are applied successfully.
It calls `Shutdown` when construction later fails or when the component shuts
down. Until ownership transfers, the component must call `Shutdown` if
construction fails.

If an external implementation is needed, replace this arrangement with a
public interface designed for that use.
