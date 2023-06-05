# Internal collector errors

This package includes helpers propagating errors from exporters or processors
back to receivers. Error propagation is helpful in a setup where the Collector
acts as a simple gateway between clients and other Collector instances or
backends.

Error propagation only works when there's synchronous processing of a pipeline.
Concretely, this means there should be no batch processors, sending queues for
exporters and potentially other resiliency mechanisms should be disabled. When
the pipeline is async, as it is the case when the batch processor is part of it,
the errors being returned by the backends used by the exporters are returned
only after the Collector's client has received its response already.

Processors doing data transformation should be treated with care, perhaps even
avoided for sync pipelines. The reason is that when a failure happens _because_
of those transformations, the Collector's client isn't at fault: a 400 returned
by a backend should NOT propagate back to the client.

Shared receivers might also fail in different ways than the outcome of the
exporters.
