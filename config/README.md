# High Level Design

This document is work in progress, some concepts are not yet available
(e.g. MapResolver is a private concept in the service for the moment).

## Map

The [Map](configmap.go) represents the raw configuration for a service (e.g. OpenTelemetry Collector).

## MapProvider

The [MapProvider](mapprovider.go) provides configuration, and allows to watch/monitor for changes.

## MapConverter

The [MapConverter](configmap.go) allows implementing conversion logic for the provided configuration. One of the most
common use-case is to migrate/transform the configuration after a backwards incompatible change.

## MapResolver

The `MapResolver` handles the use of multiple [MapProviders](./mapprovider.go) and [Converters](./configmap.go)
simplifying configuration parsing, monitoring for updates, and the overall life-cycle of the used config providers.
The `MapResolver` provides two main functionalities: [Configuration Resolving](#configuration-resolving) and
[Watching for Updates](#watching-for-updates).

### Configuration Resolving

The `MapResolver` receives as input a set of `MapProviders`, a list of `MapConverters`, and a list of configuration identifier
`configUri` that will be used to generate the resulting, or effective, configuration in the form of a `config.Map`,
that can be used by code that is oblivious to the usage of `MapProviders` and `MapConverters`.

```terminal
             MapResolver               MapProvider
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
                 │          MapConverter    │
                 │                │         │
              ┌─ │     Convert    │         │
              │  ├───────────────►│         │
   foreach    │  │                │         │
 MapConverter │  │◄───────────────┤         │
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
configuration retrieved via the `MapProvider` used to retrieve the “initial” configuration and to generate the “effective” one.

```terminal      
        MapResolver          MapProvider
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

The `MapResolver` does that by passing an `onChange` func to each `MapProvider.Retrieve` call and capturing all watch events. 
