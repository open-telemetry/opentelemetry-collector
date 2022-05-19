# High Level Design

This document is work in progress, some concepts are not yet available
(e.g. MapResolver is a private concept in the service for the moment).

## ConfMap

The [ConfMap](confmap.go) represents the raw configuration for a service (e.g. OpenTelemetry Collector).

## Provider

The [Provider](provider.go) provides configuration, and allows to watch/monitor for changes. Any `Provider`
has a `<scheme>` associated with it, and will provide configs for `configURI` that follow the "<scheme>:<opaque_data>" format.
This format is compatible with the URI definition (see [RFC 3986](https://datatracker.ietf.org/doc/html/rfc3986)).
The `<scheme>` MUST be always included in the `configURI`. The scheme for any `Provider` MUST be at least 2
characters long to avoid conflicting with a driver-letter identifier as specified in
[file URI syntax](https://tools.ietf.org/id/draft-kerwin-file-scheme-07.html#syntax).

## Converter

The [Converter](converter.go) allows implementing conversion logic for the provided configuration. One of the most
common use-case is to migrate/transform the configuration after a backwards incompatible change.

## MapResolver

The `MapResolver` handles the use of multiple [Providers](#provider) and [Converters](#converter)
simplifying configuration parsing, monitoring for updates, and the overall life-cycle of the used config providers.
The `MapResolver` provides two main functionalities: [Configuration Resolving](#configuration-resolving) and
[Watching for Updates](#watching-for-updates).

### Configuration Resolving

The `MapResolver` receives as input a set of `Providers`, a list of `Converters`, and a list of configuration identifier
`configURI` that will be used to generate the resulting, or effective, configuration in the form of a `config.Map`,
that can be used by code that is oblivious to the usage of `Providers` and `Converters`.

```terminal
             MapResolver                 Provider
                 │                          │
   Resolve       │                          │
────────────────►│                          │
                 │                          │
              ┌─ │        Retrieve          │
              │  ├─────────────────────────►│
              │  │                          │
              │  │◄─────────────────────────┤
   foreach    │  │                          │
  configURI   │  ├───┐                      │
              │  │   │Merge                 │
              │  │◄──┘                      │
              └─ │                          │
                 │            Converter     │
                 │                │         │
              ┌─ │     Convert    │         │
              │  ├───────────────►│         │
    foreach   │  │                │         │
   Converter  │  │◄───────────────┤         │
              └─ │                          │
                 │                          │
◄────────────────┤                          │
                 │                          │
```

The `Resolve` method proceeds in the following steps:

1. Start with an empty "result" of `config.Map` type.
2. For each config URI retrieves individual configurations, and merges it into the "result".
2. For each "Converter", call "Convert" for the "result".
4. Return the "result", aka effective, configuration.

### Watching for Updates
After the configuration was processed, the `MapResolver` can be used as a single point to watch for updates in the
configuration retrieved via the `Provider` used to retrieve the “initial” configuration and to generate the “effective” one.

```terminal      
        MapResolver          Provider
            │                     │
   Watch    │                     │
───────────►│                     │
            │                     │
            .                     .
            .                     .
            .                     .
            │      onChange       │
            │◄────────────────────┤
◄───────────┤                     │
```

The `MapResolver` does that by passing an `onChange` func to each `Provider.Retrieve` call and capturing all watch events. 
