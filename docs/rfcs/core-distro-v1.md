# A v1 core distro by KubeCon EU 2027

**Authors**: [@mx-psi](https://github.com/mx-psi) and [@codeboten](https://github.com/codeboten)

An earlier version of this RFC was shared with other Collector SIG approvers and maintainers who also helped shape it.

## Overview

We propose to

1. mark the OpenTelemetry Collector [core
   distro](https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol#opentelemetry-collector-core-distro)
   and [OTLP
   distro](https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-otlp#opentelemetry-collector-otlp-distro)
   (a subset of the core distro) as v1
2. mark all [Phase 1 high priority Collector components as
   v1](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/44130)

by [KubeCon Europe 2027](https://events.linuxfoundation.org/kubecon-cloudnativecon-europe/) (March
2027). This means some items on the original [v1 OTLP distro
roadmap](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/ga-roadmap.md) and
on the original vision for stability of these components will be postponed for a future major
version of the Collector or of these components.

To make the message easy to understand, we would focus on releasing the core distro as v1: a distro
with only vendor-agnostic components that are useful in a wider variety of use cases than the OTLP
distro. This paves the way for this to be our 'default' distro as opposed to the contrib distro.

## Motivation

In April 2024, the Collector SIG approved a [GA roadmap for getting an OTLP distro to
v1](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/ga-roadmap.md). This
roadmap has been [reviewed and approved by the OTel
TC](https://github.com/open-telemetry/opentelemetry-collector/issues/11499). About 80% of the
[high-level items](https://github.com/open-telemetry/opentelemetry-collector/issues/9375) are done.

In November 2025, the Collector SIG approved a list of [Phase 1 high priority individual opentelemetry-collector-contrib
components](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/44130) to be
marked as v1 to address TOC feedback for OpenTelemetry graduation. One of the seven components is
marked as v1, with work ongoing for another four.

In both efforts, we have struggled to make progress on some subitems, due to reasons including:

1. The Collector SIG has so far taken a more strict interpretation of semantic versioning and a more
   feature-rich approach to its suite of capabilities than other projects for its v1 version. This
   is stricter than other parts of OpenTelemetry itself and other CNCF graduated projects (e.g.
   [Kubernetes](https://kubernetes.io/docs/reference/deprecation-policy/)).
2. Unlike some SDK SIGs, the Collector SIG has waited for stability in other parts of the project
   (e.g. semantic conventions), even though users adopt current semantic conventions in production.
3. Since OCB consumes Collector components as Go modules, Go API versioning and application
   versioning, even if most end users of binary distributions do not care about Go API stability.

This is hurting the perception of OpenTelemetry and the Collector which are sometimes seen as:

* unstable or of beta quality (see [TOC feedback during
  graduation](https://github.com/cncf/toc/issues/1739#issuecomment-3386269224)) despite being [used
  widely in production](https://opentelemetry.io/blog/2024/collector-roadmap/)
* slow to progress (we’ve been trying to get to v1 for 5 years)

The outcome of both projects is also difficult to explain to end users succinctly: users think about
the Collector holistically, struggling to understand the nuances of what v1 of subcomponents or Go
modules mean.

We therefore want to scope down and simplify the current roadmaps.

## Goals

The goals of this reframing are as follows:

* **Simple messaging**. It should be easy to understand what is promised and what is now stable.
* **Relevant for a wide range of end users**. Completing this effort should improve stability,
  reliability and user perception for users on a wide number of use cases.
* **Stabilize what is used in production**. We treat current behavior and Go API as the desired v1
  behavior by default for these specific components.
* **Uphold existing component stability commitments**. Existing [commitments for testing and
  documentation
  remain](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#stable).

## What parts do we keep on prioritizing after this RFC?

There are three main parts that we need to fulfill for this.

### Validation of stability requirements for high priority components

We continue validating testing, benchmarking, documentation and observability [requirements for
stable
components](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#stable).
This means pursuing graduation for each of these components using the process we have [successfully
followed for the k8s attributes
processor](https://opentelemetry.io/blog/2026/k8s-attributes-processor-v1/).

### Prepare the core distro for being v1

We should:

1. audit the core distro component list, dropping components that are not widely used or are
   replaceable by other components (e.g. transform processor replaces the attributes or the resource
   processor). This may mean rebranding the core distro as a "slim, production-ready distro". It may
   still have some components that are outside of the high-priority list
2. add high-priority components to it that are not already in its manifest.
3. Add a [mechanism to the core distro to disable unstable components by
   default](https://github.com/open-telemetry/opentelemetry-collector/issues/14064) and have to
   explicitly opt into using unstable components

### Some parts of the v1 OTLP distro roadmap

We should continue to pursue stabilization of modules required for API stability. This would include
stabilizing or removing dependencies on these remaining modules:

* [`go.opentelemetry.io/collector/config/confighttp`](https://pkg.go.dev/go.opentelemetry.io/collector/config/confighttp), used by the OTLP receiver.
* [`go.opentelemetry.io/collector/filter`](https://pkg.go.dev/go.opentelemetry.io/collector/filter), used by the hostmetrics receiver.
* [`go.opentelemetry.io/collector/exporter/exporterhelper`](https://pkg.go.dev/go.opentelemetry.io/collector/exporter/exporterhelper), whose configuration is used by the OTLP exporters.

We continue with the batching migration roadmap until at least [Phase
3](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/batching-migration.md#phase-3).
Other parts (see below) are deprioritized.

## What we are **not** prioritizing after this RFC

Completing this by KubeCon Europe 2027 means not prioritizing some things, especially those items
that have been contentious or difficult to get consensus on or those that we can justifiably
postpone. In a general sense, we should aim to postpone anything that can be justified by the goals
above.

We take deprioritization to mean that we don't see these as *required* for tagging as v1. We may
still do some of these if the SIG or others find additional time, and since OpenTelemetry is a
community-led project, parts of the community may work on these regardless of what is listed here.

A non-exhaustive list of items that were listed before or that we have been working on in practice
and that we won't prioritize anymore is as follows:

1. **Semconv stabilization**. Existing semconv used by the host metrics receiver or the resource
   detection processor are widely used despite not being stable. We will allow components to be
   marked as v1 with their current semantics per the ['stable by default'
   guidance](https://opentelemetry.io/blog/2025/stability-proposal-announcement/#instrumentation-and-convention-goals).
   Components may adopt the latest version advised by the Semantic Conventions documentation in an
   opt-in way following [our usual migration
   strategy](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/semconv-feature-gates.md).
2. [**`consumererror`
   stabilization**](https://github.com/open-telemetry/opentelemetry-collector/issues/12984). We will
   stop requiring `consumererror` stabilization to be able to tag v1
3. **Migration to new pipeline telemetry**. We will not require having migrated to [new pipeline
   telemetry](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/component-universal-telemetry.md)
   or changing internal metrics
4. **Stabilizing components that are not high priority but that are in the core distro**. The core
   distro will be marked as v1 even if it has unstable components. These components may be
   stabilized later. These components will NOT be available by default, having to use [some
   mechanism to opt into using
   them](https://github.com/open-telemetry/opentelemetry-collector/issues/14064). This is consistent
   with our [versioning
   guidance](https://github.com/open-telemetry/opentelemetry-collector/blob/main/VERSIONING.md#general-considerations).
5. **Improvements to 'subcomponents' such as OTTL functions, stanza operators or resource detection
   processors**. We will not work on behavior improvements on stanza operators or resource detection
   processors or other subcomponents. If a subcomponent has significant drawbacks or is very likely
   to change drastically or be removed in the future, we will use a feature gate to 'gate' it.
6. **Parts marked as explicitly out of scope from the v1 OTLP distro roadmap and Phase 1 high
   priority components list**. This includes stabilizing items such as OCB, profiling, other Go APIs
   such as the component helper, service or otelcol modules or other component kinds such as
   connectors or extensions.

## Open questions

This is a list of open questions that we have not agreed on; we do not need to agree on these to
approve the RFC.

1. What minor version do we use for tagging the distribution? (do we reset the minor version or do we use the same one used in opentelemetry-collector or opentelemetry-collector-contrib)
2. Do we align minor versions across opentelemetry-collector and opentelemetry-collector-contrib modules?
3. What's the release cadence after v1? Do we keep on "every other week" cadence or do we make it
   monthly?
4. Do we want to release a 'stable-only' distro?
5. How does versioning and support work in practice after v2? Do we need to update our existing
   versioning guidance?
6. How does the mechanism to opt into unstable components work?
