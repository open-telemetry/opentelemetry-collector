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

## Possible mechanisms

The following are the possible mechanisms for configuring Providers. They are
not exclusive from each other and could all be implemented if desired. This
section does not comment on which of these options will be implemented or how
they will look when implemented.

### Reuse the Collector's config file

This option would add extra top-level keys to the Collector config to allow configuring providers.

**Requires a breaking change**: Probably not, would be extra config options that
could be added later. Nested config resolution would have to be carefully
considered to prevent breaking changes.

**Requirements fulfilled**: (1), (2), and (3)

### Configure Providers through a separate config file/structure

This option is an alternative to the previous one and would instead configure
providers in a separate file that is loaded before the Collector's config.

**Requires a breaking change**: No, this could be added separately.

**Requirements fulfilled**: (1), (2), and (3)

### Configure Providers through URIs

This would be done by using a part of the URI (e.g. URI fragments) to configure
URIs per invocation. Settings could be grouped through nested URI invocations.

**Requires a breaking change**: Yes, though this could likely be done without
breaking many users by using infrequently-used characters as delimeters between
URIs and options.

**Requirements fulfilled**: (2) and (3)

### Configure Providers through an in-band configuration mechanism

This would be done within files by adding to the currently-supported in-file
provider syntax (`${...}`), e.g. through adding square brackets to the end
(`${...}[...]`). Settings could be grouped through nested URI invocation inside
the options.

**Requires a breaking change**: Yes, in-file configuration would change due to
syntax additions, and confmap APIs would need to be updated to accept options
alongside a URI to resolve.

**Requirements fulfilled**: (2) and (3)
