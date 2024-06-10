# Experimental support for profiling

## Overview

The OpenTelemetry collector has traces, metrics and logs as stable signals. We
want to start experimenting with support for profiles as an experimental
signal. But we also don't want to introduce breaking changes in packages
otherwise considered stable.

This document describes:

* The approach we intend to take to introduce profiling with no breaking changes
* How the migration will happen once profiling goes stable too

### Discarded approach

A first approach, discussed in [issue
10207](https://github.com/open-telemetry/opentelemetry-collector/issues/10207)
has been discarded.
It aimed to refactor the current packages with per-signal subpackages, so each
subpackage could have its own stability level, like pdata does.

This has been discarded, as the Collector SIG does not want the profiling
signal to impact the road to the collector reaching 1.0.

## Proposed approach

The proposed approach will consist of two main phases:

* Introduce `experimental` packages for each required module of the collector that needs to be profiles-aware.
	* `consumer`, `receiver`, `connector`, `component`, `processor`
* Mark specific APIs as `experimental` in their godoc for parts that can't be a new package.
	* `service`

### Introduce `experimental` packages

Each package that needs to be signal-aware will have its barebones copied into
a new `XXexperimental` package (eg: `consumerexperimental`) which will be able
to handle the profiling signal.

The stable related stable packages MUST not be aware of their experimental
counterpart.
The experimental package MAY call its stable implementation.

### Mark specific APIs as `experimental`

In order to boot a functional collector with profiles support, the some stable
packages need to be aware of the experimental ones.

The only package concerned here should be `service`.

To support that case, we will mark new APIs as `experimental` with go docs.
Every experimental API will be documented as such:

```golang
// # Experimental
//
// Notice: This method is EXPERIMENTAL and may be changed or removed in a
// later release.
```

As documented, APIs marked as experimental may changed or removed across
releases, without it being considered as a breaking change.

#### User specified configuration

The user-specified configuration will let users specify a `profiles` pipeline:

```
service:
	pipelines:
		profiles:
			receivers: [otlp]
			processors: [batch]
			exporters: [otlp]
```

## Signal status change

If the profiling signal becomes stable, all the experimental packages will be
merged back into their stable counterpart, and the `service` module's imports
will be updated.

If the profiling signal is removed, all the experimental packages will be
removed from the repository, and support for them will be removed in the
`service` module.
