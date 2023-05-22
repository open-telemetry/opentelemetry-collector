# Internal collector errors

This package includes helpers propagating errors from exporters or processors
back to receivers. Error propagation is helpful in a setup where the Collector
acts as a simple gateway between clients and other Collector instances or
backends.

Error propagation only works when there's synchronous processing of a pipeline.
Concretely, this means there should be no batch processors, sending queues for
exporters and other resiliency mechanisms should be disabled.

"Asymmetrical" processors (e.g. Anything that merges, splits, reorganizes,
drops, or adds ptraces/pmetrics/plogs) shouldn't also be included in the
pipeline.

Shared receivers might also fail in different ways than the outcome of the
exporters.
