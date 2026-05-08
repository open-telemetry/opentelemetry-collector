# Count `PartialSuccess` in `exporterhelper`

Author: @braydonk

## Intro

The current `exporterhelper` observability behaviour is to [opaquely recognize any error to mean that the entire batch has failed](https://github.com/open-telemetry/opentelemetry-collector/blob/91b32efbbda0180db8b063262589abaf6a0c9df9/exporter/exporterhelper/internal/obs_report_sender.go#L163), counting the entire size of the current request as being failed. This is not always the case; some backends will implement [the PartialSuccess portion of the OTLP specification](https://opentelemetry.io/docs/specs/otlp/#partial-success) or may [have a similar partial rejection protocol](https://docs.cloud.google.com/logging/docs/reference/v2/rest/v2/entries/write#:~:text=Optional.%20Whether%20a%20batch%27s%20valid%20entries%20should%20be%20written%20even%20if%20some%20other%20entry%20failed%20due%20to%20a%20permanent%20error%20such%20as%20INVALID_ARGUMENT%20or%20PERMISSION_DENIED.%20If%20any%20entry%20failed%2C%20then%20the%20response%20status%20is%20the%20response%20status%20of%20one%20of%20the%20failed%20entries.) that makes it so only some items within a request failed while the rest succeeded. Under the current `exporterhelper` functionality, it is impossible to properly count this scenario.

## Proposal

Two new error APIs will be introduced:

### `Countable` error 

This is an error that can any other error can be wrapped in along with a count of failures. This will allow for cases where producers of an error want to include an amount of failures.

This API may end up being useful in other places than `PartialSuccess`, such as [in `receiverhelper`](https://github.com/open-telemetry/opentelemetry-collector/issues/14440).

### `PartialSuccess` error

A `PartialSuccess` error will wrap an internal error as permanent, implement [GRPC status code resolution](https://pkg.go.dev/google.golang.org/grpc/status#FromError) to code `OK`, and wrap it all as a `Countable` with a count of failures.

#### Why should the internal error be permanent?

A partial success SHOULD NOT be retried. When a server responds according to or similar to the OTLP `PartialSuccess` spec, it is not possible to determine which items exactly failed, and retrying to whole request would end up retrying items that already succeeded. As a result, the rest of the upstream pipeline's behaviour (crucially, the `retry_sender`) does not need to change how it reacts to a `PartialSuccess` compared to any other `PermanentError`.

#### Why does it implement status code `OK`?

A `PartialSuccess` is nominally a success according to the spec. As a result, when `exporterhelper` processes this it needs to know that the Go error it received doesn't actually correspond to a real failure to send data.This will allow accurate setting of the span status (it shouldn't be `Failed` upon `PartialSuccess`) and determining of `error.type` (a new custom type called `Partial_Success` would be introduced).

## Implementation Plan

This will be done as two PRs:

1. [Implementing the new error APIs](https://github.com/open-telemetry/opentelemetry-collector/compare/main...braydonk:opentelemetry-collector:countable_consumer_error)
2. [Implementing the counting in `exporterhelper`'s `obs_report_sender`](https://github.com/braydonk/opentelemetry-collector/compare/countable_consumer_error...braydonk:opentelemetry-collector:exporterhelper_partial_success)
3. Recognize and return `PartialSuccess` errors from `otlp` and `otlp_http` exporters (no draft yet)

## Out of Scope

The following is out of scope for this RFC:

* Making the upstream components respond differently to `PartialSuccess` 
    - I would wager that it's unlikely it ever needs to, but regardless this RFC is scoped completely to self observability entirely within the exporter in this scenario, whereas the rest of the upstream pipeline will only ever respond to this the same as any other permanent error
* Fully counting data drops throughout the pipeline
    - The new `Countable` interface does open the possibility of counting partial failures at various points throughout the pipeline, which might be worth exploring down the line. The only thing I want to address right now is ensuring these are properly counted at the exporter metric level.
* Partial retries
    - This is purely covering scenarios that align with the OTLP spec's definition of `PartialSuccess`, which does not come with any functionality to determine exactly which items failed
