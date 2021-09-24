# Collector Feature Gates

This package provides a mechanism that allows operators to enable and disable
experimental or transitional features at deployment time. These flags should
be able to govern the behavior of the application starting as early as possible
and should be available to every component such that decisions may be made
based on flags at the component level.

## Usage

A component that needs a feature gate that will not be shared with other 
components should define a `Gate` as an unexported package variable and add 
it to the global registry in an `init()` function.

```go
var fancyNewFeatureGate = &configgate.Gate{
	ID:          "namespaced.uniqueIdentifier",
	Description: "A brief description of what the gate controls",
	Enabled:     false,
}

func init() {
	err := configgates.GetRegistry().Add(fancyNewFeatureGate)
	if err != nil {
		panic(err)
	}
}
```

The `Enabled` property of the feature gate may then later be checked to 
determine whether to enable the gated feature:

```go
if fancyNewFeatureGate.Enabled {
	setupNewFeature()
	doThingWithNewFeature()
} else {
	doThingWithOldFeature()
}
```

For components that need to query the state of globally-applicable feature 
gates a read-only implementation of the `configgates.Gates` interface is 
available in the `*CreateSettings` structs passed to component factories. It 
exposes an `IsEnabled(id string) bool` method that can be used to query the 
status of any `Gate`.

```go
if settings.Features.IsEnabled("global.featureGate") {
	setupNewFeature()
}
```

## Controlling Gates

Feature gates can be enabled or disabled via the CLI, with the 
`--feature-gates` flag, or via configuration. When using the CLI flag, gate 
identifiers must be presented as a comma-delimited list.  In either location 
prefixing a gate identifier with `-` will disable the gate and prefixing 
with `+` or with no prefix will enable the gate.

```shell
otelcol --config=config.yaml --feature-gates=gate1,-gate2
```

This will enable `gate1` and disable `gate2`.  The same can be achieved with 
the following `service` configuration:

```yaml
service:
  features:
    - gate1
    - -gate2
```

## Feature Lifecycle

Features controlled by a `Gate` should follow a three-stage lifecycle, 
modeled after the [system used by Kubernetes](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages):

1. An `alpha` stage where the feature is disabled by default and must be enabled 
   through a `Gate`.
2. A `beta` stage where the feature has been well tested and is enabled by 
   default but can be disabled through a `Gate`.
3. A generally available stage where the feature is permanently enabled and 
   the `Gate` is no longer operative.

Features that prove unworkable in the `alpha` stage may be discontinued 
without proceeding to the `beta` stage.  Features that make it to the `beta` 
stage will not be dropped and will eventually reach general availability 
where the `Gate` that allowed them to be disabled during the `beta` stage 
will be removed.