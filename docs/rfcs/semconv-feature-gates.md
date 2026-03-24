# Semantic conventions migrations in the Collector

## Overview

The OpenTelemetry Collector components emit telemetry that often conforms to semantic conventions.
Semantic conventions have [varying levels of stability][1] and often have an SDK-focused migration
guide.

This RFC defines how migration should be handled in Collector components that have
semantic conventions that migrate to a stable version, in a Collector-native way.

## Scope and goals

This RFC provides general guidelines for semantic convention-mandated migrations of telemetry created by Collector components (usually receivers) and output into the Collector's pipeline. It explicitly does not attempt to cover:
- telemetry created by an application and forwarded by a Collector receiver;
- internal telemetry of Collector components;
- guidelines for the migration of specific semantic conventions.

The migration mechanism should have the following characteristics:

1. **Collector native**: the mechanism should work in a similar way to other Collector migrations
   and should feel natural and intuitive to users. 
2. **Simple**: a user should have to make a small number of changes to their Collector deployment to
   migrate to a new set of conventions.
3. **Easy to understand**: It should be easy to understand how to migrate a particular set of
   conventions.
5. **Flexible (double publish)**: The mechanism should allow you to 'double publish' v0 and v1
   conventions
6. **Flexible (other conventions)**: The mechanism should still allow for evolution of other
   semantic conventions that are not being migrated.

## Background 

### Setup

We want to write guidance for when we have a component that emits telemetry from a common
`area` that is undergoing a migration mandated by the Semantic Conventions SIG. In the rest of this
document we refer to the **v0** conventions and the **v1** conventions, which are the conventions
in this area before and after the migration.

When the semantic conventions are specific to a component we use 
- `kind` to refer to the component kind (receiver, exporter...)
- `id` for the component id (e.g. `hostmetrics`)

### What does the semconv spec say?

The semantic conventions specification defines an environment variable named
`OTEL_SEMCONV_STABILITY_OPT_IN` that, for each area, takes two possible values:
1. One value representing the new semantic conventions (e.g. `http`, `gen_ai_latest_experimental`)
2. Once mature enough, a second value ending in `/dup` that emits both the old conventions and the
   new ones.

This is not specified in a generic way, but it is a consistent pattern across all semantic
conventions areas that are being actively worked on:

<details>

<summary> Example 1: HTTP compatibility warning </summary>

Taken from [semconv v1.38.0][2]:

> **Warning**
> Existing HTTP instrumentations that are using
> [v1.20.0 of this document](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.20.0/specification/trace/semantic_conventions/http.md)
> (or prior):
>
> * SHOULD NOT change the version of the HTTP or networking conventions that they emit
>   until the HTTP semantic conventions are marked stable (HTTP stabilization will
>   include stabilization of a core set of networking conventions which are also used
>   in HTTP instrumentations). Conventions include, but are not limited to, attributes,
>   metric and span names, and unit of measure.
> * SHOULD introduce an environment variable `OTEL_SEMCONV_STABILITY_OPT_IN`
>   in the existing major version which is a comma-separated list of values.
>   The only values defined so far are:
>   * `http` - emit the new, stable HTTP and networking conventions,
>     and stop emitting the old experimental HTTP and networking conventions
>     that the instrumentation emitted previously.
>   * `http/dup` - emit both the old and the stable HTTP and networking conventions,
>     allowing for a seamless transition.
>   * The default behavior (in the absence of one of these values) is to continue
>     emitting whatever version of the old experimental HTTP and networking conventions
>     the instrumentation was emitting previously.
>   * Note: `http/dup` has higher precedence than `http` in case both values are present
> * SHOULD maintain (security patching at a minimum) the existing major version
>   for at least six months after it starts emitting both sets of conventions.
> * SHOULD drop the environment variable in the next major version (stable
>   next major version SHOULD NOT be released prior to October 1, 2023).

</details>

<details>

<summary> Example 2: GenAI compatibility warning </summary>

From [semconv v1.38.0][3]:

> [!Warning]
>
> Existing GenAI instrumentations that are using
> [v1.36.0 of this document](https://github.com/open-telemetry/semantic-conventions/blob/v1.36.0/docs/gen-ai/README.md)
> (or prior):
>
> * SHOULD NOT change the version of the GenAI conventions that they emit by default.
>   Conventions include, but are not limited to, attributes, metric, span and event names,
>   span kind and unit of measure.
> * SHOULD introduce an environment variable `OTEL_SEMCONV_STABILITY_OPT_IN`
>   as a comma-separated list of category-specific values. The list of values
>   includes:
>   * `gen_ai_latest_experimental` - emit the latest experimental version of
>     GenAI conventions (supported by the instrumentation) and do not emit the
>     old one (v1.36.0 or prior).
>   * The default behavior is to continue emitting whatever version of the GenAI
>     conventions the instrumentation was emitting (1.36.0 or prior).
>
> This transition plan will be updated to include stable version before the
> GenAI conventions are marked as stable.

</details>

<details>

<summary> Example 3: K8s compatibility warning </summary>

> From [semconv v1.38.0][3]:

> When existing K8s instrumentations published by OpenTelemetry are
> updated to the stable K8s semantic conventions, they:
> 
> - SHOULD introduce an environment variable `OTEL_SEMCONV_STABILITY_OPT_IN` in
>   their existing major version, which accepts:
>   - `k8s` - emit the stable k8s conventions, and stop emitting
>     the old k8s conventions that the instrumentation emitted previously.
>   - `k8s/dup` - emit both the old and the stable k8s conventions,
>     allowing for a phased rollout of the stable semantic conventions.
>   - The default behavior (in the absence of one of these values) is to continue
>     emitting whatever version of the old k8s conventions the
>     instrumentation was emitting previously.
> - Need to maintain (security patching at a minimum) their existing major version
>   for at least six months after it starts emitting both sets of conventions.
> - May drop the environment variable in their next major version and emit only
>   the stable k8s conventions.

> Specifically for the Opentelemetry Collector:

> The transition will happen through two different feature gates.
> One for enabling the new schema called `semconv.k8s.enableStable`,
> and one for disabling the old schema called `semconv.k8s.disableLegacy`. Then:

> - On alpha the old schema is enabled by default (`semconv.k8s.disableLegacy` defaults to false),
>   while the new schema is disabled by default (`semconv.k8s.enableStable` defaults to false).
> - On beta/stable the old schema is disabled by default (`semconv.k8s.disableLegacy` defaults to true),
>   while the new is enabled by default (`semconv.k8s.enableStable` defaults to true).
> - It is an error to disable both schemas
> - Both schemas can be enabled with `--feature-gates=-semconv.k8s.disableLegacy,+semconv.k8s.enableStable`.

</details>

## Proposed mechanism

Suppose the `<id>` (e.g. `hostmetrics`) `kind` (e.g. `receiver`) component is migrating from v0 to
v1 semantic conventions on the area `area` (e.g. `process`). The semantic conventions specification
defines the set of conventions that are in scope for a particular migration.

To support this migration, the component defines two feature gates: `<kind>.<id>.EmitV1<Area>Conventions` (e.g.
`receiver.hostmetrics.EmitV1ProcessConventions`) and `<kind>.<id>.DontEmitV0<Area>Conventions`
(e.g. `receiver.hostmetrics.DontEmitV0ProcessConventions`). These feature gates work as follows:

| `<kind>.<id>.EmitV1<Area>Conventions` status | `<kind>.<id>.DontEmitV0<Area>Conventions` status    | Resulting behavior                                        |
|-----------------------------------------------|-------------------------------------------------------|-----------------------------------------------------------|
| Disabled                                      | Disabled                                              | Emit telemetry under the 'v0' conventions                |
| Disabled                                      | Enabled                                               | Error at startup since this would not emit any telemetry  |
| Enabled                                       | Disabled                                              | Emit telemetry under both the v0 and the v1 conventions |
| Enabled                                       | Enabled                                               | Emit telemetry under the v1 conventions                  |

Both feature gates evolve at the same pace through the feature gate stages, so that the progression
is as follows:
1. Initially both are at **alpha** stage (disabled by default). This means that the default behavior
   is to emit only the 'v0' conventions. Users can opt-in to emit the v1 conventions alongside the
   v0 conventions or to emit only the v1 conventions. A warning message must be logged by the component at startup indicating the upcoming change.
2. Whenever there is a semantic conventions release that marks these as stable, the feature gates are promoted to the
   **beta** stage on the same Collector release. The new default behavior is therefore to emit only the
   'v1' conventions. Users can opt-out to emit the v1 conventions alongside the v0 conventions or
   to emit only the v0 conventions.
3. After 4 minor releases, the feature gates are promoted to the **stable** stage. At this point users
   can only use the v1 conventions.
4. After additional 4 minor releases, the feature gates are removed.

This mechanism does not cover any sort of transition for experimental semantic conventions. These
presumably would be covered by separate feature gates or some other mechanism.

## Handling conflicts during double-publishing

During the double-publishing phase (when both `<kind>.<id>.EmitV1<Area>Conventions` is enabled and `<kind>.<id>.DontEmitV0<Area>Conventions` is disabled), components typically emit both v0 and v1 telemetry. For metrics, this usually means emitting two separate metrics with different names (e.g., `http.server.duration` for v0 and `http.server.request.duration` for v1).

However, in some cases v0 and v1 conventions may use the same metric name but with different characteristics (e.g., different attributes or metric types). Two metrics with identical names must not be emitted, as this would produce an invalid OpenTelemetry dataset and cause issues on backends. Below are examples illustrating how such conflicts can be resolved:

**Different attributes:** If a metric name stays the same but an attribute is renamed, emit a single metric with both the v0 and v1 attributes present. For instance, if `process.cpu.time` uses `process.owner` in v0 and `process.owner.name` in v1, emit one metric with both attributes.

**Different metric type:** If a metric name stays the same but the type changes (e.g., Gauge to UpDownCounter), emit a single metric with the v1 type, effectively prioritizing the new convention. For instance, if `system.memory.usage` changes from Gauge to UpDownCounter, emit it as an UpDownCounter.

## Alternative mechanisms

There are some other possibilities:

### Environment variable

We could just use the `OTEL_SEMCONV_STABILITY_OPT_IN` mechanism. However, this does not feel
"Collector native": Collector users expect experimental features to be controlled via feature gates
and as such this could be a surprising mechanism. In particular, users would expect that they are
able to 'roll back' to the previous behavior even after a Collector upgrade, something that the
environment variable mechanism explicitly does not support.

### More granular feature gate pairs

The granularity of the feature gates described could be changed: we could have a pair per convention
or even a pair for the whole Collector. I argue 'per component' strikes the right balance between
simplicity and flexibility: 
- per convention would lead to dozens of feature gates on some of the areas we want to stabilize. It
  would also be unclear how these interact on edge cases (semantic conventions may only make sense
  holistically)
- a single pair of feature gates would effectively be forever unstable and would not be flexible
  enough to allow people to migrate on a per dashboard basis

### Meta feature gate

We could have both a feature gate pair per component and a meta target feature gate pair that allows
you to enable/disable all v1 conventions at the same time. This is effectively a superset of the
proposed mechanism, so I argue we can postpone this for later: if users ask for it, we can always
add it in the future.

## Open questions and future possibilities

This document does not cover how to deal with experimental semantic conventions after the 'big'
migration has been completed in one particular area. What to do here in part depends on the
[stabilization changes][4]. Quoting the blogpost:
> Instrumentation stability should be decoupled from semantic convention stability. We have a lot of
> stable instrumentation that is safe to run in production, but has data that may change in the
> future. Users have told us that conflating these two levels of stability is confusing and limits
> their options.

How to deal with these remains an open question that should be tackled in OTEPs first.

As mentioned above, the 'Meta feature gate' remains a possibility even when adopting this mechanism.

[1]: https://opentelemetry.io/docs/specs/semconv/general/semantic-convention-groups/#group-stability
[2]: https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/http/README.md
[3]: https://github.com/open-telemetry/semantic-conventions/blob/v1.38.0/docs/gen-ai/README.md
[4]: https://opentelemetry.io/blog/2025/stability-proposal-announcement/
