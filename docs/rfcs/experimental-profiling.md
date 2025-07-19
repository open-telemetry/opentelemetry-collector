# Experimental support for profiling

## Overview

The OpenTelemetry collector has traces, metrics and logs as stable signals. We
want to start experimenting with support for profiles as an experimental
signal. But we also don't want to introduce breaking changes in packages
otherwise considered stable.

This document describes:

* The approach we intend to take to introduce profiling with no breaking changes
* How the migration will happen once profiling goes stable

### Discarded approaches

#### Refactor everything into per-signal subpackages

A first approach, discussed in [issue
10207](https://github.com/open-telemetry/opentelemetry-collector/issues/10207)
has been discarded.
It aimed to refactor the current packages with per-signal subpackages, so each
subpackage could have its own stability level, like pdata does.

This has been discarded, as the Collector SIG does not want the profiling
signal to impact the road to the collector reaching 1.0.

#### Use build tags

An approach would have been to use build tags to limit the availability of
profiles within packages.

This approach would make the UX very bad though, as most packages are meant to
be imported and not used in a compiled collector. It would therefore not have
been possible to specify the appropriate build tags.

This has been discarded, as the usage would have been too difficult.

## Proposed approach

The proposed approach will consist of two main phases:

* Introduce `experimental` packages for each required module of the collector that needs to be profiles-aware.
	* `consumer`, `receiver`, `connector`, `component`, `processor`
* Mark specific APIs as `experimental` in their godoc for parts that can't be a new package.
	* `service`

### Introduce "experimental" subpackages

Each package that needs to be profiling signal-aware will have its public
methods and interfaces moves into an internal subpackage.

Then, the original package will get similar API methods and interfaces as the
ones currently available on the main branch.

The profiling methods and interfaces will be made available in a `profiles`
subpackage.

See [PR
#10253](https://github.com/open-telemetry/opentelemetry-collector/pull/10253)
for an example.

### Mark specific APIs as `experimental`

In order to boot a functional collector with profiles support, some stable
packages need to be aware of the experimental ones.

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

There are no symbols that would need to be marked as experimental today. If
there ever are then implementers may add an experimental comment to them

#### User specified configuration

The user-specified configuration will let users specify a `profiles` pipeline:

```
service:
	pipelines:
		profiles:
			receivers: [otlp]
			exporters: [otlp]
```

When an experimental signal is being used, the collector will log a warning at
boot.

## Signal status change

If the profiling signal becomes stable, all the experimental packages will be
merged back into their stable counterpart, and the `service` module's imports
will be updated.

If the profiling signal is removed, all the experimental packages will be
removed from the repository, and support for them will be removed in the
`service` module.
