# Collector Feature Gates

This package provides a mechanism that allows operators to enable and disable
experimental or transitional features at deployment time. These flags should
be able to govern the behavior of the application starting as early as possible
and should be available to every component such that decisions may be made
based on flags at the component level.

## Usage

Feature gates must be defined and registered with the global registry in
an `init()` function.  This makes the `Gate` available to be configured and 
queried with a default value of its `Enabled` property.

```go
var fancyNewFeatureGate = featuregate.Gate{
	ID:          "namespaced.uniqueIdentifier",
	Description: "A brief description of what the gate controls",
	Enabled:     false,
}

func init() {
	featuregate.Register(fancyNewFeatureGate)
}
```

A `Gate` registered in this manner cannot be queried directly as 
configuration and CLI settings for feature gates are only applied to a 
frozen copy of the registry that is made available through the `config.
Config` associated with a running `service.service` instance.

For components that are started by a `service.service` instance, a read-only 
implementation of the `featuregate.Gates` interface is available in the 
`*CreateSettings` structs passed to component factories. It exposes an 
`IsEnabled(id string) bool` method that can be used to query the status of 
any `Gate`. 

```go
if settings.Gates.IsEnabled("namespaced.uniqueIdentifier") {
	setupNewFeature()
}
```

## Controlling Gates

Feature gates can be enabled or disabled via the CLI, with the 
`--feature-gates` flag, or via configuration. When using the CLI flag, gate 
identifiers must be presented as a comma-delimited list. Gate identifiers
prefixed with `-` will disable the gate and prefixing with `+` or with no
prefix will enable the gate.

```shell
otelcol --config=config.yaml --feature-gates=gate1,-gate2
```

This will enable `gate1` and disable `gate2`.  The same can be achieved with 
the following `service` configuration, though there a `map[string]bool` must
be used:

```yaml
service:
  gates:
    gate1: true
    gate2: false
```

Setting the status of a feature gate via the CLI will take precedence over 
any setting in the `service` configuration, assuming the default 
`ParserProvider` is used.  As the `ParserProvider` is responsible for 
applying the feature gates CLI flags in the appropriate order, any changes to
the `ParserProvider` used by a `service.Collector` instance may affect the
feature gate configuration.

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

## Control Flow

All `Gates` must be registered with the global registry by calling 
`featuregate.Register(gate)` in an `init()` function.  During `Collector` 
startup, the configured `ParserProvider` is used to obtain a `ConfigMap` 
that is unmarshalled into a `Config` by the configured `ConfigUnmarshaler`.  

The `ParserProvider` is responsible for ensuring that a `map[string]bool` is
available in the `ConfigMap` at the `service::gates` key.  The default 
`ParserProvider` is a composite that will attempt to load values first from 
a file and then from CLI flags.

Once a `ConfigMap` is available, it is presented to the configured
`ConfigUnmarshaler` instance.  The `ConfigUnmarshaler` is responsible for
getting the `map[string]bool` from the `ConfigMap` and using it to call
`featuregate.Apply(gateConfig)`.  This will return a copy of the global gate
registry with the provided configuration applied as a `featuregate.Gates`
implementation.  This `Gates` value can then be set in the `config.Config`
as appropriate, where it will be available for use by the `service.service`
instance created by the `service.Collector`.
