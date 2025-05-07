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

## Resolution

The `confmap` module APIs will not substantially change for 1.0. The following
steps will be taken to ensure that configuration can be made to work post-1.0:

1. Restrict URIs sufficiently to allow for extension after 1.0, e.g. restricting
   the scheme to allow for things like "named schemes" (`file/auth:`).
2. Stabilize confmap Providers individually, so they can impose any desired
   restrictions on their own.
3. Offer configuration as an optional interface for things like options that are
   applied to all instances of a Provider.

## Possible technical solutions

*NOTE*: This section is speculative and may not reflect the final implementation
for providing options to confmap Providers.

Providers are invoked through passing `--config` flags to the Collector binary
or by using the braces syntax inside a Collector config file (`${scheme:uri}`).
Each invocation contains a scheme specifying how to obtain config and URI
specifying the config to be obtained. A single instance of a Provider is created
for each scheme and is tasked with retrieving config for its scheme for each
corresponding URI passed to the Collector.

With the above in mind, we have the following places where it may make sense to
support specifying options for Providers:

1. Parts of the URI we are requesting.
1. Separate flags to configure Providers per config URI.
1. Use a separate config file that specifies config sources inside a map
   structure.
1. Extend the Collector's config schema to support specifying additional places
   to obtain configuration.

All of the above options are targeted toward configuring how specific URIs are
resolved into config. To configure how a Provider resolves every URI it
receives, we should consider how to extend the above options to be specified
without a URI and to ensure the options are always applied to all URI
resolutions.

### Configure options inside the URI

[RFC 3986](https://datatracker.ietf.org/doc/html/rfc3986#section-3), which
specifies the format of a URI, specifies the different parts of a URI and
suggests two places where we could pass options to Providers: queries and
fragments.

#### Queries

Breaking changes:

- confmap Providers would have breaking changes since they would now consume
  unescaped URI queries. There would be no breaking changes to the confmap API.

Advantages:

- Explicitly intended to specify non-hierarchical data in a URI.
- Often used for this purpose.
- Fits into existing config URIs for URL-based Providers.

Disadvantages:

- Only allows easily specifying key-value pairs.
- Query parameters are somewhat frequently used, which may extend to backend
  requests, and this may cause some churn for users who are unfamiliar that we
  would be consuming them.

#### Fragments

We could specify options in a query parameter-encoded string placed into the URI
fragment.

Breaking changes:

- confmap Providers would have breaking changes since they would now consume
  fragments. There would be no breaking changes to the confmap API.

Advantages:

- Not likely to be used by config backends for any of our supported protocols,
  so has a low chance of conflict when using unescaped fragments.
- Fits into existing config URIs for URL-based Providers.

Disadvantages:

- Even if fragments are likely not useful to backends, we are still preventing
  unescaped use in upstream Providers.
- Doesn't conform to the spirit of how fragments should be used according to RFC
  3986.
- Only allows easily specifying key-value pairs.

We could likely partially circumvent the key-value pair limitation by
recursively calling confmap Providers to resolve files, env vars, HTTP URLs,
etc. For example:

```text
https://config.com/config#refresh-interval=env:REFRESH_INTERVAL&headers=file:headers.yaml
```

Using this strategy would also allow us to more easily get env vars and to get
values from files for things like API tokens.

### Separate flags to configure Providers per config URI

Breaking changes:

- Will need factory options if we provide config through a mechanism similar to
  `component.Factory`, along with making a Provider instance per URI.
- Otherwise will need to break `confmap.Provider` interface to support providing
  options in `Retrieve`.

Advantages:

- Allows us to keep config URIs opaque.
- Options live right next to config URIs on the command line.

Disadvantages:

- The flags would need to be placed in a certain position in the arguments list
  to specify which URI they apply to.
- Configuring URIs present in files requires users to look in two places for
  each URI and is suboptimal UX.
- Complicating the flags like this would be suboptimal UX.

### Specify additional config sources inside the main Collector configuration

This is a variant of providing a separate config source-only configuration file
that instead puts those URIs and their options inside the main configuration
file.

API changes:

- Need a way to specify options, either through a factory option, or an optional
  interface.

Advantages:

- Allows us to keep URIs opaque.
- Map structures are easier to work with than command-line arguments for complex
  config.

Disadvantages:

- There are now two ways to include config inside a Collector config file.
- Complicates the config schema and the config resolution process.
