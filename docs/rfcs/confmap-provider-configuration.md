# Configuration of confmap Providers

## Motivation

The `confmap.Provider` interface is used by the Collector to retrieve map
objects representing the Collector's configuration or a subset thereof. Sources
config may include locally-available information such as files on disk or
environment variables, or may be remotely accessed over the network. In the
process of obtaining configuration from a source, the user may wish to modify
the behavior of how the source is obtained.

For example, consider the case where the Collector obtains configuration over
HTTP from an HTTP endpoint. A user may want to:

1. Poll the HTTP endpoint for configuration at a configurable interval and
   reload the Collector service if the configuration changes.
2. Authenticate the request to get configuration by including a header in the
   request.

This would produce a set of options like the following:

- `poll-interval`: Sets an interval for the Provider to check the HTTP endpoint
  for changes. If the config has changed, the service will be reloaded. Defaults
  to `0`, which disables polling.
- `headers`: Specifies a semicolon-delimited set of headers that will be passed
  to the config backend. Header keys and values should be separated by `=`.

Configuration for Providers cannot be stored in the same place as the rest of
the Collector's configuration since the Providers are the ones tasked with
getting that configuration. Therefore, they need a separate configuration
mechanism.

## Current state

No upstream Providers currently offer any configuration options. The exported
interfaces are still able to change before the `confmap` module is declared
stable, but avoiding breaking changes in the API would be preferable.

## Desired state

We would like a mechanism to configure Providers that meets the following
requirements:

1. Broadly applicable to all current upstream providers.
2. Can be used as a suggestion for configuring custom providers.
3. Is consistent between Providers.
4. Is configurable per config URI.

## Solutions

Providers are invoked through passing `--config` flags to the Collector binary
with a scheme and URI to obtain. A single instance of each Provider is then
tasked with retrieving from config from all URIs passed to the Collector for the
scheme the provider is registered to handle.

We have the following places where it may make sense to hook into options for
Providers:

1. Parts of the URI we are requesting.
2. Part of the string passed to `--config`.
3. Separate flags to configure Providers per config URI.
4. Use a separate config file that specifies config sources inside a map
   structure.
5. Extend the Collector's config schema to support specifying additional places
   to obtain configuration

### Configure options inside the URI

[RFC 3986](https://datatracker.ietf.org/doc/html/rfc3986#section-3), which
specifies the format of a URI, specifies the different parts of a URI and
suggests two places where we could pass options to Providers: queries and
fragments.

#### Queries

Advantages:

- Explicitly intended to specify non-hierarchical data in a URI.
- Often used for this purpose.
- Fits into existing config URIs.

Disadvantages:

- Completely consuming the parameters would mean we can't pass them to config
  backends. The frequent use of query parameters in server endpoints would force
  us to limit the types of requests we can make.
- Partially consuming parameters would still cause conflicts if a Provider
  parameter and config backend parameter share the same name. Additionally,
  mixing the two would provide suboptimal UX.
- Only allows easily specifying key-value pairs.

#### Fragments

We could specify options in a query parameter-encoded string placed into the URI
fragment.

Advantages:

- Not likely to be used by config backends for any of our supported protocols,
  so has a low chance of conflict.
- Fits into existing config URIs.

Disadvantages:

- Even if fragments are likely not useful to backends, we are still preventing
  their use in upstream Providers.
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

### Add to the string passed to the config flag

Instead of configuring Providers inside their URIs, we could add an extra
section to the string passed to `--config` and configure how a URI is retrieved
through that config. This approach is undesirable because the config flag
currently only takes valid URIs, and updating the format to accommodate this
would require we adopt an unconventional format.

### Separate flags to configure Providers per config URI

Advantages:

- Allows us to keep config URIs opaque.
- Options live right next to config URIs on the command line.
- Does not require breaking changes.

Disadvantages:

- The flags would need to be placed in a certain position in the arguments list
  to specify which URI they apply to.
- Complicating the flags like this would be suboptimal UX.

### Separate config file

We could allow specifying a config file that contains all URIs to obtain the
Collector's configuration from.

Advantages:

- Allows us to keep URIs opaque.
- Map structures are easier to work with than command-line arguments for complex
  config.
- Shouldn't require any breaking changes.

Disadvantages:

- The Collector's main config file now depends on a separate config file, which
  would be suboptimal UX.

### Specify additional config sources inside the main Collector configuration

This is a variant of providing a separate config source-only configuration file
that instead puts those URIs and their options inside the main configuration
file.

Advantages:

- Compared to the other option, keeps a single configuration file.
- All advantages of the other file option.

Disadvantages:

- There are now two ways to include config inside a Collector config file.
- Using the existing bracket syntax inside config files does not support
  options, and neither does specifying them through the command line.
- Complicates the config schema and the config resolution process.

## Resolution

We will configure providers through URI fragments. These are seldom used in
server-to-server requests since they are generally used to point to a subsection
of a document such as an HTML page. Additionally, they add minimal conceptual
overhead compared to additional flags or files.
