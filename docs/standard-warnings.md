# Standard Warnings
Some components have scenarios that could cause issues.  Some components require the collector be interacted with in a specific way in order to ensure the component works as intended. This document describes common warnings that may affect a component in the collector. 

Visit a component's README to see if it is affected by any of these standard warnings.

## Unsound Transformations
Incorrect usage of the component may lead to telemetry data that is unsound i.e. not spec-compliant/meaningless.  This would most likely be caused by converting metric data types or creating new metrics from existing metrics.

## Statefulness
The component keeps state related to telemetry data and therefore needs all data from a producer to be sent to the same Collector instance to ensure a correct behavior. Examples of scenarios that require state would be computing/exporting delta metrics, tail-based sampling and grouping telemetry.

## Identity Crisis
The component may change the ['identity' of a metric](https://github.com/open-telemetry/opentelemetry-specification/blob/main//specification/metrics/data-model.md#opentelemetry-protocol-data-model-producer-recommendations).  This could be down either by changing a metrics name, removing attributes, or updating existing attribute values.  Adding attributes to metrics is always safe and does not create an Identity Crisis.

## Orphaned Telemetry
The component modifies the incoming telemetry in such a way that a span becomes orphaned, that is, it contains a `trace_id` or `parent_span_id` that does not exist.  This may occur because the component can modify `span_id`, `trace_id`, or `parent_span_id` or because the component can delete telemetry.  