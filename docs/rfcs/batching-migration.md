# Coexistence of `batchprocessor` and exporterhelper batching

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

This RFC supersedes the unmerged draft in
[#11947](https://github.com/open-telemetry/opentelemetry-collector/pull/11947)
and gives a more focused answer to the questions raised in
[#15047](https://github.com/open-telemetry/opentelemetry-collector/issues/15047),
[#13766](https://github.com/open-telemetry/opentelemetry-collector/issues/13766),
[#13582](https://github.com/open-telemetry/opentelemetry-collector/issues/13582),
[#13583](https://github.com/open-telemetry/opentelemetry-collector/pull/13583),
[#14038](https://github.com/open-telemetry/opentelemetry-collector/issues/14038),
[#12022](https://github.com/open-telemetry/opentelemetry-collector/issues/12022),
and [#8122](https://github.com/open-telemetry/opentelemetry-collector/issues/8122).

### `batchprocessor` defects

The processor is widely used. It has well-documented defects:

- Suppresses errors (always returns `nil`)
- Returns to the caller before the export has completed
- Single-export concurrency
- Sizes batches by item count
- It interrupts trace context and (without `metadata_keys`) drops
  client metadata.

### The exporterhelper batcher is good, and not yet a drop-in replacement

The `exporterhelper` queue/batch sender resolves the known
batchprocessor defects and integrates new features:

- Choice of sizing logic by items, bytes, and requests
- Optional synchronous or asynchronous behavior
- Optional persistent storage extension
- Optional blocking

At this time, the `exporterhelper` queue/batch sender has feature
parity with `batchprocessor`.

### The double-batching problem

Once `exporterhelper` batching is enabled by default in stable
exporters, every pipeline that also contains a `batchprocessor` will
apply batching twice.

Double batching wastes CPU and can hurt latency. This prevents the
project from turning on exporterhelper batching by default while
`batchprocessor` is still common in user configurations.

We have explored several escapes, none satisfying:

- **Deprecate `batchprocessor`** ([#12022](https://github.com/open-telemetry/opentelemetry-collector/issues/12022),
  [#13766](https://github.com/open-telemetry/opentelemetry-collector/issues/13766)):
  technically unblocked now that `metadata_keys` exists in
  `exporterhelper`, but still impractical given the size of the
  installed base — a hard removal would break many deployed pipelines.
- **Reimplement `batchprocessor` on top of `exporterhelper`**
  ([#13583](https://github.com/open-telemetry/opentelemetry-collector/pull/13583)):
  feature-correct, but inherits exporter-shaped metric names
  ([#14038](https://github.com/open-telemetry/opentelemetry-collector/issues/14038))
  and still leaves the double-batching problem when defaults change.
- **Rename to `inlinebatchprocessor` / introduce `pipelineprocessor`**
  ([#15047](https://github.com/open-telemetry/opentelemetry-collector/issues/15047)):
  user-facing churn and does not address existing deployments.

## Goals

1. Allow `exporterhelper` batching to become the default for stable
   exporters without silently double-batching existing pipelines.
2. Preserve the `batchprocessor` for users who depend on it.
3. Make the long-term direction reversible if a chosen mechanism
   proves wrong: no breaking renames, no irreversible config changes.
4. Give distribution authors (e.g. the Helm chart) a single switch to
   adopt the new default without rewriting every user's pipeline.

## Proposed mechanisms

We propose adopting **both** of the following. They solve different
parts of the problem and compose cleanly.

### Mechanism 1: a feature gate controlling batch defaults

Introduce a single feature gate, e.g.
`exporter.defaultBatching.enabled`, that controls whether stable
exporters enable the `exporterhelper` batch feature by default when
the user has not explicitly configured `sending_queue.batch`.

| Stage   | Default | Behavior                                   |
|---------|---------|--------------------------------------------|
| Alpha   | off     | Existing behavior. Opt-in only.            |
| Beta    | off     | Documentation, distribution guidance.      |
| Stable  | on      | Exporters batch by default. Users opt out. |
| Removed |         | Default-on permanently.                    |

This is a global setting that will affect all exporters.


An optional companion gate (e.g. `processor.batch.disabled`) MAY be
added to let distributions short-circuit a `batch:` block that exists
in user config but is no longer wanted, without requiring users to
edit their YAML.

Benefits:

- Coordinated, observable rollout following the existing feature-gate
  contract.
- Distribution authors flip one switch.
- Reversible at any pre-stable stage.

Limitations:

- A static gate cannot tell whether *this particular pipeline* already
  has a `batchprocessor`. Operators with mixed pipelines must still
  edit configs to avoid double batching.
- The gate is global; it does not solve the per-pipeline problem on
  its own, which motivates Mechanism 2.

### Mechanism 2: a runtime "already batched" context marker

Define a context value (carried on `client.Context` or as a dedicated
key on the request `context.Context`) that a batching component sets
on the contexts it forwards downstream:

```go
// pseudocode
ctx = batching.MarkBatched(ctx, batching.Source{Component: "batchprocessor", InstanceID: id})
```

The `exporterhelper` batch sender, when it observes this marker on an
incoming request, MUST short-circuit:

- It does not start a new batch timer.
- It does not merge with other in-flight batches.
- It forwards the request to the next sender (queue, retry, timeout,
  exporter) as-is.

In effect, the helper batcher becomes a no-op for any request that has
already been batched upstream. If multiple batchers exist in a
pipeline, only the first one runs.

Benefits:

- Solves double-batching automatically and per-pipeline. A user who
  retains `batchprocessor` in a single pipeline does not pay for two
  batchers, and does not need to edit YAML to opt out.
- Local, conservative, additive change: no config surface changes, no
  renames, no new components.
- Composes with Mechanism 1: turning on default batching is then safe
  even when `batchprocessor` is present.
- Forward-compatible with future batching components (a
  `pipelineprocessor`, a future inline batcher, etc.) — any batcher
  can set the marker.

Limitations:

- Only suppresses the *batching* stage of `exporterhelper`. The queue,
  retry, and timeout senders still apply, which is the desired
  behavior.
- Adds a small contract that batching components must honor (set the
  marker on outgoing contexts; respect the marker on incoming
  contexts). This must be documented as part of the batching API.
- The marker must survive context boundaries that today do not
  preserve metadata. The implementation needs a clear story for which
  context carries it (see Open Questions).

## How the two mechanisms work together

| Pipeline shape | Mechanism 1 only | Mechanism 1 + 2 |
| --- | --- | --- |
| `... → exporter` (no batch processor) | Helper batches (good). | Helper batches (good). |
| `... → batch → exporter` (legacy) | Double-batches (bad). | `batch` runs; helper sees marker and skips (good). |
| `... → batch(metadata_keys) → exporter` | Double-batches (bad). | `batch` runs; helper skips (good). |
| `... → batch → exporter (helper batch explicitly disabled)` | Single-batches (good). | Single-batches (good). |

Mechanism 1 controls *the default*. Mechanism 2 makes that default
*safe* in the presence of existing `batchprocessor` deployments.

## Detailed design notes

### Context marker shape

Two reasonable carriers:

1. A new field on `client.Info` (e.g. `client.Info.Batched []Source`).
   Carried alongside metadata, available wherever the helper already
   reads `client.Info`.
2. A typed value on `context.Context`, accessed via a small package
   (e.g. `pipeline/batching`) that exposes `MarkBatched`,
   `IsBatched`, and `Sources`.

Option 2 is preferred: batching is a pipeline concern, not a
client/transport concern, and `context.Context` is the existing carrier
between consecutive components.

The marker is a list, not a boolean, so that diagnostic tooling can
report which component(s) batched a request.

### Interaction with `MergeSplit`

When the helper batcher receives an oversized request that has already
been batched upstream, it MUST still split it if a maximum size is
configured for the exporter, because that size is an exporter-protocol
constraint, not a batching preference. Splitting in this case happens
without buffering or timer activation; each split request retains the
upstream marker.

### Interaction with the queue sender

The queue sender is unaffected by the marker. The marker only
suppresses the batching step. This keeps queueing, retry, and
timeout semantics intact regardless of where batching happened.

### Metric naming for `batchprocessor`-on-helper

[#14038](https://github.com/open-telemetry/opentelemetry-collector/issues/14038)
tracks fixing the metric names emitted when `batchprocessor` is
reimplemented on top of `exporterhelper`. It is a follow-up rather
than a blocker: this RFC works whether `batchprocessor` keeps its
current implementation or adopts the helper internally, and Mechanism
2 in particular is independent of how `batchprocessor` is built.

## Alternatives considered

- **Rename `batchprocessor` to `inlinebatchprocessor` and introduce
  `pipelineprocessor`** ([#15047](https://github.com/open-telemetry/opentelemetry-collector/issues/15047)):
  rejected for this RFC because it forces user-visible churn and does
  not by itself prevent double batching.
- **Detect `batchprocessor` presence at config-load time and disable
  helper batching for that pipeline**: brittle (must enumerate every
  current and future batching component) and surprising (action at a
  distance based on a sibling component's name).
- **Do nothing and require users to edit configs when defaults
  change**: rejected; the installed base is too large and the cost of
  silent double-batching is too high.

## Open questions

1. Should the marker be advisory or normative? This RFC proposes
   normative ("MUST short-circuit"); a debug/override toggle MAY be
   provided for testing.
2. Where exactly does the marker live (`client.Info` vs.
   `context.Context` package)? Preference here is the latter.
3. Should the marker include the upstream batcher's effective size
   limits, so the helper can decide whether *additional* splitting is
   needed without re-batching? Probably yes, as a `[]Source` with
   optional metadata.
4. What is the deprecation horizon for `batchprocessor`? This RFC
   intentionally does not propose one; it removes the technical
   blockers so that decision can be made later on its own merits.
