# Migration for `batchprocessor` and exporterhelper batching

## Overview

The Collector now offers two batching mechanisms:

1. The `batchprocessor` component, used in pipelines for many years and
   widely deployed, notably as a default in the OpenTelemetry Helm
   chart and other distributions.
2. The `exporterhelper` combined queue/batch sender, a replacement for
   the batch processor, is ready to replace it.

We find there are a small number of users and known use-cases that
require a batch processor, for example:

- When multiple exporters are in use, the processor form uses less memory
  because the exporters share batched requests.
- Aggregation processors (e.g., `groupbyattrs`) benefit from larger
  batches and recommend applying the batch processor first.

At the same time, the existing batch processor has several known
defects (e.g., concurrency limit, error propagation, trace context),
and fixing the defects require breaking changes.

Our goal is to migrate most users away from the batch processor and
enable batching by default for all exporterhelper users. This will
require a careful sequence of steps for a successful migration.

## Background

This supersedes an unmerged draft in
[#11947](https://github.com/open-telemetry/opentelemetry-collector/pull/11947)
and answers
[#15047](https://github.com/open-telemetry/opentelemetry-collector/issues/15047),
[#13766](https://github.com/open-telemetry/opentelemetry-collector/issues/13766),
[#13582](https://github.com/open-telemetry/opentelemetry-collector/issues/13582),
[#13583](https://github.com/open-telemetry/opentelemetry-collector/pull/13583),
[#14038](https://github.com/open-telemetry/opentelemetry-collector/issues/14038),
[#12022](https://github.com/open-telemetry/opentelemetry-collector/issues/12022),
[#8122](https://github.com/open-telemetry/opentelemetry-collector/issues/8122).

### `batchprocessor` defects

The processor is widely used. It has well-documented defects:

- Suppresses errors (always returns `nil`)
- Returns to the caller before the export has completed
- Single thread concurrency
- Sizes batches only by item count
- Interrupts trace context.

The single thread concurrency problem is particularly a problem for
error propagation. Performance can suffer when the downstream
components block waiting for export. If the exporter is synchronous,
as by exporterhelper's `wait_for_result: true`, the batch processor
creates a single-export bottleneck.

Even while suppressing errors, batchprocessor creates
backpressure. This operational detail must be preserved as we
deprecate this component. Below, we propose to make exporterhelper
`block_on_overflow: true` the default behavior to address this
concern.

### Exporterhelper replacement

The `exporterhelper` queue/batch sender resolves the known
batchprocessor defects and integrates new features:

- Choice of sizing logic by items, bytes, and requests
- Optional synchronous (`wait_for_result`) or asynchronous behavior
- Optional persistent storage extension
- Optional blocking (`block_on_overflow`).

The `exporterhelper` queue/batch sender has feature parity with
`batchprocessor`. However, exporterhelper default settings remain
aligned and compatible with `batchprocessor`:

- `wait_for_result: false` asynchronous export
- `block_on_overflow: false` drop when full
- `batch::enabled: false` batching disabled.

The exporterhelper supports opt-in persistent storage, and when no
storage extension is configured it uses an in-memory queue with
`wait_for_result: false`, meaning the default behavior is to accept
data and immediately return success. This is what `batchprocessor`
expects, and this is also how `batchprocessor` behaves, returning
success, not waiting for potential errors.

The exporterhelper's built-in batching support is critical for
vendor-specific components, because it gives exporters a way to control request size in
terms of bytes, even considering custom export protocols. The ideal
final state is that we enable exporterhelper batching by default. Many
exporters already do this, but not all, the choice is made
per-component.

### What success looks like

The `batchprocessor` is removed from the core repository.

A re-implemented `queuebatchprocessor` will be created (in the core)
using exporterhelper as the implementation. This will use the combined
QueueBatchConfig struct and support all batching modes; it will use
the same defaults as exporterhelper recommends for exporters.

OpenTelemetry documentation will not refer to `batchprocessor`.

The OpenTelemetry Helm charts will not refer to `batchprocessor`.

#### Recommendation: no error propagation by default

The `wait_for_result` feature will remain off by default. The
Collector's default position is to accept data in-memory and return
success.

Note that configuring `wait_for_result: true` together with a
persistent storage extension is not supported, and should become
checked as an invalid configuration (tracked under "Additional work
items" below).

#### Recommendation: block on overflow by default

The `block_on_overflow` feature will become enabled by default.

This is a user visible change. Receivers that set a timeout will cause
all exporters by default to block when the queue is full.  Collector's
default position is to wait for memory to become available.

A feature flag for this change will be defined,
`exporterhelper.exporterQueueBlockOnOverflow`. The default
queue configuration will use `BlockOnOverflow` set to the value
of the feature, becoming true by default in "beta".

#### Recommendation: exporterhelper batching enabled by default

Currently, components have a mixture of `configoptional.Some`,
`configoptional.Default`, and `configoptional.None`, usually passing
`exporterhelper.DefaultQueueBatchConfig()`. However, many of these
configurations are historical, set to `Default` or `None` because the
exporterhelper was evolving.

Not all exporters want queue/batching enabled by default. Concrete
examples raised during review include:

- **Pull-based exporters** (e.g., `prometheus`) have no concept of an
  outbound queue or batch at all.
- **Order-sensitive exporters** (e.g., `prometheusremotewrite`) cannot
  use the standard queue because it does not guarantee in-order
  delivery; they typically run with `num_consumers: 1` and no batching.
- **Exporters whose client library already implements retry/queueing**
  (e.g., `googlecloud`) disable retries and may not want a second
  layer of queue/batching on top.

Even so, we believe many exporters will choose to enable batching by
default now that the feature is stable. This RFC calls for an audit of
the components, evaluating for every exporter whether a `Default` or
`None` should become `Some` in each exporter's default queue config.

A feature flag for this change will be defined,
`exporterhelper.exporterQueueBatchEnabled`.

#### Recommended final default queue configuration

The `NewDefaultQueueConfig` function will begin being influenced by
two feature flags.

```go
func NewDefaultQueueConfig() queuebatch.Config {
	return queuebatch.Config{
		Sizer:           request.SizerTypeRequests,
		NumConsumers:    10,
		QueueSize:       1_000,
        // CHANGED: Blocking is becoming enabled by default.
		BlockOnOverflow: exporterQueueBlockOnOverflow.IsEnabled(),
        WaitForResult:   false,
		Batch: configoptional.DefaultOrSome(
            // CHANGED: Batching is becoming enabled by default.
            exporterQueueBatchEnabled.IsEnabled(),
            queuebatch.BatchConfig{
			    FlushTimeout: 200 * time.Millisecond,
			    Sizer:        request.SizerTypeItems,
			    MinSize:      8192,
            }),
	}
}
```

The above makes use of an hypothetical `configoptional` helper:

```
// DefaultOrSome ties a feature flag to a default-to-some transition.
func DefaultOrSome[T any](feature bool, value T) Optional[T] {
	if feature {
		return Some(value)
	}
	return Default(value)		
}
```

### Double-batching problem

One concern preventing the migration is that we may unintentionally
apply multiple batching processes in a single pipeline. This is a
coordination problem. If a pipeline with a default- or user-configured
batch processor has settings that are out of line with the new
exporterhelper defaults, users will pay the cost of batching and then
re-batching.

Users have 6 releases (~3 months) during which the batch processor
will print a warning that users should upgrade. During this window
the new exporterhelper defaults are still off, so unmodified pipelines
do not double-batch. Adopters who manually enable the Beta-on values
to provide feedback (see Phase 2 exit criteria) are protected by the
startup warning introduced in Phase 1, step 5.

## Risks

The phased timeline below is designed around the following risks:

- **Custom exporters that do not use exporterhelper lose batching
  entirely** once `batchprocessor` is removed. The Phase 2 deprecation
  window gives these users time to adopt `queuebatchprocessor` or move
  to exporterhelper.
- **`block_on_overflow: true` changes failure mode.** Pipelines that
  previously dropped silently on overflow will apply backpressure;
  receivers with timeouts may begin returning errors upstream. Must be
  highlighted in release notes.
- **Batching-on-by-default adds latency.** Exporters adopting the new
  defaults introduce up to `FlushTimeout` of latency where there was
  none before.
- **Silent double batching during Phase 2** when users keep
  `batchprocessor` alongside an exporter with batching enabled.
  Mitigated by the startup warning under "Additional work items".
- **Distribution drift.** Non-OpenTelemetry distros and vendor docs
  referencing `batchprocessor` become invalid on removal. Mitigated by
  the Phase 1 advisory and the Phase 2 Helm chart switch.
- **Insufficient real-world feedback.** Mitigated by the Phase 2 exit
  criteria requiring confirmed early adopters.

## Migration timeline

We treat the batchprocessor deprecation as separate from the change of
exporterhelper default with an interleaved timeline. That is:

1. Batch processor will become deprecated with printed instructions
   and documentation for users to explicitly set exporterhelper
   queue/batch configuration instead, explaining that the defaults are
   undergoing migration.
2. Exporterhelper will introduce two feature flags and proceed to enable
   `block_on_overflow: true` and batching `enabled: true`.

The sequence of events is ordered in phases.  The following table
summarizes what an operator running a typical pipeline observes at
each phase:

| Phase | `batchprocessor` in config                                   | exporterhelper defaults                                                    | Net behavior                                                                                                                                                               | Release version  |
|-------|--------------------------------------------------------------|----------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------|
| <1    | OK                                                           | `batch::enabled: false`, `block_on_overflow: false`                        | Documented behavior                                                                                                                                                        |               |
| 1     | OK                                                           | Unchanged; feature gates are Alpha (default off)                           | No change for users; new `queuebatchprocessor` and docs available for opt-in migration.                                                                                    |               |
| 2     | Works, prints deprecation warning pointing to migration docs | Unchanged; gates still Alpha/off                                           | Users see warnings and have at least six releases to migrate; Helm chart switches to the new path while `batchprocessor` still works. Exit gated on early-adopter success. | v0.158.0    |
| 3     | Removed, Collector fails to start if referenced              | Gates Beta (default on): `batch::enabled: true`, `block_on_overflow: true` | Pipelines that didn't migrate fail fast; exporters batch and apply backpressure by default; users can still opt out via gate for six releases.                             | v0.164.0  |
| 4     | n/a                                                          | Gates Stable and removed; new defaults permanent                           | Users choose `exporterhelper` or `queuebatchprocessor` for batching                                                                                                        | v0.170.0 |

### Phase 1

These are loosely dependent,

1. Ensure all exporters in OpenTelemetry repos define `queue_sender` having
   type `configoptional.Optional[exporterhelper.QueueBatchConfig]`.
   Add an mdatagen-generated check that fails when an exporter's default
   config omits `queue_sender` or sets it to a non-standard value.
   Exporters that legitimately need different defaults (e.g., pull-based
   exporters) must declare the opt-out in their `metadata.yaml`.
2. Introduce the two feature gates at **Alpha** stability (default off).
3. Support a configurable metrics prefix to distinguish processor
   batching from exporter batching
   ([#14038](https://github.com/open-telemetry/opentelemetry-collector/issues/14038)).
4. Implement `queuebatchprocessor`
   ([#13583](https://github.com/open-telemetry/opentelemetry-collector/pull/13583)).
5. Detect pipelines that combine `batchprocessor` with an
   exporterhelper that has batching enabled, and emit a startup warning
   identifying the affected pipeline and exporter. This protects users
   from silently paying for double batching during Phase 2, when
   adopters may enable the new defaults manually while `batchprocessor`
   is still present.
6. Update documentation explaining `batchprocessor` deprecation with
   instructions to adopt explicit exporterhelper `queue_sender`
   settings or the new `queuebatchprocessor`. Issue release notes and
   advisories to distros to remove `batchprocessor` from
   documentation. Create a blog post about what's happening to
   `batchprocessor`, get it approved.

### Phase 2

7. Deprecate `batchprocessor`. Release the blog post. The first
   release where this lands is tentatively **v0.158.0** (August 2026).
8. Full audit of exporters and default settings. Exporters can opt-in
   to the migration here by electing `Some` instead of `Default` or
   `None`. Exporters that do not want default blocking/batching will
   override and/or validate that their exporterhelper queue/batch
   configuration are valid; they must stop using the
   `NewDefaultQueueConfig()`.
9. Update various documentation pointing to `queuebatchprocessor`,
    which will advise users to configure exporter batching in most
    cases. ([#13766](https://github.com/open-telemetry/opentelemetry-collector/issues/13766))

After the batch processor is deprecated, we will wait at least +6
releases (~3 months) with `batchprocessor` printing warnings and
referring to the documentation, which will guide them to fill explicit
exporterhelper settings. During this window we actively solicit
early-adopter feedback on the two feature gates set to their Beta-on
values.

#### Exit criteria for Phase 2

Phase 2 does not end purely on a release-count schedule. Before the
Helm chart switch (step 10) and the transition to Phase 3, we must
have:

- At least three production adopters of the two feature gates (set to
  the Beta-on values manually) who have reported back, of which at
  least one operates the Collector at non-trivial scale. Adopters will
  be tracked in a running list maintained alongside this RFC (e.g., in
  the deprecation tracking issue).
- No open regressions reported by those adopters against the new
  defaults.

If these criteria are not met after +6 releases, Phase 2 is extended
in 3-release increments until they are met.

Once the exit criteria are satisfied:

10. Update the standard OpenTelemetry Helm chart to no longer reference
   `batchprocessor`. The chart instead configures exporterhelper
   `queue_sender` with batching enabled (or uses `queuebatchprocessor`).
   `batchprocessor` is still available as a fallback for users who have
   not yet migrated. Shipping the updated chart is the signal that we
   are ready to enter Phase 3.

### Phase 3

In a single release cycle:

11. Remove `batchprocessor`.
12. Promote both feature gates to **Beta** (default on). Exporterhelper
    now blocks on overflow and enables batching by default; users can
    still opt out via the gate for one cycle.

### Phase 4

We will wait another +6 releases (~3 months).

13. Promote both feature gates to **Stable** and remove them. The new
    exporterhelper defaults become permanent and can no longer be
    toggled via the gate.

## Additional work items

Items noted in this document that should be tracked separately,
accomplished during Phase 1:

- Reject `wait_for_result: true` combined with a configured storage
  extension at configuration validation time (see "Recommendation: no
  error propagation by default" above).

## Conclusion

At the end, we can close a whole bunch of issues going back to
[#8122](https://github.com/open-telemetry/opentelemetry-collector/issues/8122).
