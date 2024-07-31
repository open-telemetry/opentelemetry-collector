# Configuration of confmap Providers

## Motivation

The `confmap.Provider` interface is used by the Collector to retrieve map
objects representing the Collector's configuration or a subset thereof. Sources
of config may include locally-available information such as files on disk or
environment variables, or may be remotely accessed over the network. In the
process of obtaining configuration from a source, the user may wish to modify
the behavior of how the source is obtained.

For example, consider the case where the Collector obtains configuration over
HTTP from an HTTP endpoint. A user may want to:

1. Poll the HTTP endpoint for configuration at a configurable interval and
   reload the Collector service if the configuration changes.
2. Authenticate the request to get configuration by including a header in the
   request. Additional headers may be necessary as part of this flow.

This would produce a set of options like the following:

- `poll-interval`: Sets an interval for the Provider to check the HTTP endpoint
  for changes. If the config has changed, the service will be reloaded.
- `headers`: Specifies a map of headers to be put into the HTTP request to the
  server.

## Current state

No upstream Providers currently offer any configuration options. The exported
interfaces are still able to change before the `confmap` module is declared
stable, but avoiding breaking changes in the API would be preferable.

## Desired state

We would like the following features available to users to configure Providers:

1. Global configuration of a certain type of Provider (`file`, `http`, etc.).
   This allows for users to express things such as "all files should be watched
   for changes" and "all HTTP requests should include authentication".
2. Named configuration for a certain type of provider that can be applied to
   particular URIs. This will allow users to express things such as "some HTTP
   URLs should be watched for changes with a certain set of settings applied".
3. Configuration options applied to specific URIs.
