# Migration for `batchprocessor` and exporterhelper batching

## Overview

The Collector now offers two batching mechanisms:

1. The `batchprocessor` component, used in pipelines for many years and
   widely deployed, notably as a default in the OpenTelemetry Helm
   chart and other distributions.
2. The `exporterhelper` combined queue/batch sender, a replacement for
   the batch processor, is ready to replace it.

We find there are a small number of users and known use-cases which
require a batch processor. At the same time, the batch processor has
several known defects, and fixing the defects will be a breaking
change.

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
- Sizes batches by item count
- It interrupts trace context and (without `metadata_keys`) drops
  client metadata.

The single thread concurrency problem is particularly a problem for
error propagation. With only one thread exporting batches, performance
suffers unless the downstream component acts the same. This explains
why the exporterhelper's `wait_for_result` queue sender setting must
remain off by default. Batch processor stands in the way of error
propagation with concurrency, so most users do not propagate errors.

### Exporterhelper replacement

The `exporterhelper` queue/batch sender resolves the known
batchprocessor defects and integrates new features:

- Choice of sizing logic by items, bytes, and requests
- Optional synchronous or asynchronous behavior
- Optional persistent storage extension
- Optional blocking

At this time, the `exporterhelper` queue/batch sender has feature
parity with `batchprocessor`. However, its default settings are still
aligned with the batch processor:

- `wait_for_result: false` we do not propagate errors
- `batch::enabled: false` we disable exporter batching

The exporterhelper supports opt-in persistent storage, but the default
in-memory queue combined with `wait_for_result: false` establishes
a potential to drop data while suppressing error propagation.

The exporterhelper's built-in batching support is critical for
vendors, because it gives exporters a way to control request size in
terms of bytes, considering their own protocol. The ideal final state
is that we enable exporterhelper batching by default. Many exporters
already do this, but not all, the choice is made per-component.

#### Recommendation: error propagation if no persistent store

We aim to set `wait_for_result: true` when there is no storage
extension configured. This flag is already only applicable when the
in-memory queue is used, storage extensions already ignore this mode.

A feature flag for this change will be defined,
`exporterhelper.memoryQueueWaitForResult`.

#### Recommendation: exporterhelper batching enabled by default

Currently, components have a mixture of `configoptional.Some`,
`configoptional.Default`, and `configoptional.None`, usually passing
`exporterhelper.DefaultQueueBatchConfig()` which is defined:

```go
func NewDefaultQueueConfig() queuebatch.Config {
	return queuebatch.Config{
		Sizer:           request.SizerTypeRequests,
		NumConsumers:    10,
		QueueSize:       1_000,
		BlockOnOverflow: false,
        WaitForResult:   false,
		Batch: configoptional.Default(queuebatch.BatchConfig{
			FlushTimeout: 200 * time.Millisecond,
			Sizer:        request.SizerTypeItems,
			MinSize:      8192,
		}),
	}
}
```

The `configoptional.Default` is responsible for disabling batching for
most users. To complete the migration, in the final state, we will have
a single standard constructor where `wait_for_result` is true and batch
is enabled by default:

```go
func StandardQueueConfig() configoptional.Optional[queuebatch.Config] {
	return queuebatch.Config{
        // ...
        WaitForResult:   true,
		Batch: configoptional.Some(queuebatch.BatchConfig{
            // ...
		}),
	}
}
```

The implementation will be determined by the feature flags discussed
in this document. There are two levels of `configoptional` here, one
for the queue and one for the batcher: both will become enabled by
default at the end of this migration.

A feature flag for this change will be defined,
`exporterhelper.exporterQueueBatchEnabled`.

### Double-batching problem

One concern preventing the migration is that we may unintentionally
apply multiple batching processes in a single pipeline. This is a
coordination problem. If a pipeline with a default- or user-configured
batch processor has settings that are out of line with the new
exporterhelper defaults, users will pay the cost of batching and then
re-batching.

#### Recommendation: coordinated batch processor phase-out

Using a single feature flag, we will enable exporterhelper
queue/batching by default, at the same time we disable the batch
processor by rendering it a no-op. First, we will deprecate the batch
processor: users will be guided in two directions:

1. In simple pipelines, the user will remove the batch processor from
   their configuration and configure the queue_sender::batch instead.
2. In complex pipelines, a new core processor will be introduced as a
   from-scratch replacement, called the `queuebatchprocessor`. The new
   core processor will be implemented by exporterhelper internally, as
   demonstrated in
   [#13583](https://github.com/open-telemetry/opentelemetry-collector/pull/13583). This
   component will support configuration identical with `queue_sender`
   but for use as a processor.

At the same time, an audit is required to clean up the many various
initial conditions across exporters. We will define an Go package
`batchmigration` with an enum to capture all the common initial
settings,

```go
// This will be deprecated and removed after the migration.
type Existing int

const (
    // Default means the caller used configoptional.Default(NewDefaultQueueConfig())
    Default Case = iota

    // Default means the caller used configoptional.Some(NewDefaultQueueConfig())
    Some Case

    // Default means the caller used configoptional.None[...]()
    None Case

    // The callsite enables queue, not batch.
    SomeQueueNoneBatch,

    // ...
)
```

Exporters will be able opt-out of the standard migration path by
rejecting the proposed changes during an audit step, where most callers
will be migrated using a new function,

```go
func MigrateQueueConfig(migrate batchmigration.Existing) configoptional.Optional[queuebatch.Config] {
    // Each case will reproduce exactly the behavior of a callsite
    // somewhere in core or collector-contrib.
}
```

A number of exporters today have custom settings because the default
settings today cause data loss. After the migration is complete, these
components should prefer to switch to standard queue config. For these
callers (e.g., `otelarrowexporter`),

```go
func MigrateSpecifyQueueConfig(original queuebatch.Config) configoptional.Optional[queuebatch.Config] {
    // This will return the original configuration until the final
    // feature flag enabling wait_for_result for in-memory queues
    // with batching enabled by default. After this point, the original
    // configuration will be ignored.
}
```

Two feature flags have been introduced so far. They are logically
independent, but defined to advance in step with each other so that
users have independent control over `wait_for_result` and default
batching behavior. The feature flag state of these two will remain
equal.

## Full timeline

The sequence of events is ordered in phases.

### Phase 1

These are loosely dependent,

1. Define `StandardQueueConfig()` function.
2. Ensure all exporters in OpenTelemetry repos define `queue_sender` having
   type `configoptional.Optional[exporterhelper.QueueBatchConfig]`
3. Define the two feature flags.
4. Support a configurable metrics prefix to distinguish processor
   batching from exporter batching
   ([#14038](https://github.com/open-telemetry/opentelemetry-collector/issues/14038)).
5. Implement `queuebatchprocessor`
   ([#13583](https://github.com/open-telemetry/opentelemetry-collector/pull/13583)).
6. Implement `batchmigration` and two migration functions for
   non-standard callers.

### Phase 2

7. Deprecate `batchprocessor`.
8. Full audit of exporters and default settings. Exporters can opt-out
   of the migration here, replace callsites use of `configoptional`
   with one of `StandardQueueConfig`, `MigrateQueueConfig`, and
   `MigrateSpecifyQueueConfig`.
9. Implement coordinated feature behavior. The default changes
   recommended above will be applied through the three functions named
   above. The batch processor will be programmed to act as a
   pass-through when the `exporterhelper.exporterQueueBatchEnabled` is
   enabled.

After the batch processor is deprecated, we will wait 6 months for
users to choose a migration path through configuration.

### Phase 3

10. Update various documentation pointing to `queuebatchprocessor`,
    which will advise users to configure exporter batching in most
    cases. ([#13766](https://github.com/open-telemetry/opentelemetry-collector/issues/13766))
11. After waiting the specified period, we will change both feature
    flags to Stable. At this time, the behavior of
    `StandardQueueConfig`, `MigrateQueueConfig`,
    `MigrateSpecifyQueueConfig`, and `NewDefaultQueueConfig()` will
    change while `batchprocessor` is simultaneously disabled. All
    exporters have `wait_for_result: true` by default, which is
    overridden by defining a storage extension.
12. Update standard Helm charts. Simply remove the batch processor,
    because exporters batch by default excepting those that opt out.
13. Deprecate `MigrateQueueConfig`, `MigrateSpecifyQueueConfig`, and
    the `batchmigration` package. Users of the migration helpers
    will switch to `StandardQueueConfig` in most cases.

### Phase 4

We will wait another 6 months before completing these steps.

14. Remove the `batchprocessor` from the core distribution.
15. Remove the two feature flags.
16. Remove the deprecated migration package and functions.

## Conclusion

At the end, we can close a whole bunch of issues going back to
[#8122](https://github.com/open-telemetry/opentelemetry-collector/issues/8122).
