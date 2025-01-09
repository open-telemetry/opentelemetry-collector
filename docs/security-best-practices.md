# Security

The OpenTelemetry Collector defaults to operating in a secure manner but is
configuration driven. This document captures important security aspects and
considerations for the Collector. This document is intended for component
developers. It assumes at least a basic understanding of the Collector
architecture and functionality.

> Note: Please review the
> [configuration documentation](https://opentelemetry.io/docs/collector/configuration/)
> prior to this security document.

Security documentation for end users can be found on the OpenTelemetry
documentation website:

- [Collector configuration best practices](https://opentelemetry.io/docs/security/config-best-practices/)
- [Collector hosting best practices](https://opentelemetry.io/docs/security/hosting-best-practices/)

## TL;DR

- Configuration
  - MUST come from the central configuration file
  - SHOULD use configuration helpers
- Permissions
  - SHOULD minimize privileged access
  - MUST document what requires privileged access and why
- Receivers/Exporters
  - MUST default to encrypted connections
  - SHOULD leverage helper functions
- Extensions
  - SHOULD NOT expose sensitive health or telemetry data by default

> For more information about securing the OpenTelemetry Collector, see
> [this blog post](https://medium.com/opentelemetry/securing-your-opentelemetry-collector-1a4f9fa5bd6f).

## Configuration

The Collector binary does not contain an embedded or default configuration and
MUST NOT start without a configuration file being specified. The configuration
file passed to the Collector MUST be validated prior to being loaded. If an
invalid configuration is detected, the Collector MUST fail to start as a
protective mechanism.

Component developers MUST get configuration information from the Collector's
configuration file. Component developers SHOULD leverage
[configuration helper functions](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config).

When defining Go structs for configuration data that may contain sensitive
information, use the `configopaque` package to define fields with the
`configopaque.String` type. This ensures that the data is masked when serialized
to prevent accidental exposure.

> For more information, see the
> [configopaque](https://pkg.go.dev/go.opentelemetry.io/collector/config/configopaque)
> documentation.

## Permissions

The Collector supports running as a custom user and SHOULD NOT be run as a
root/admin user. For the majority of use-cases, the Collector SHOULD NOT require
privileged access to function. Some components MAY require privileged access or
external permissions, including network access or RBAC.

Component developers SHOULD minimize privileged access requirements and MUST
document what requires privileged access and why.

## Receivers and Exporters

Receivers and Exporters can be either push or pull-based. In either case, the
connection established SHOULD be over a secure and authenticated channel.

Component developers MUST default to encrypted connections (using the
`insecure: false` configuration setting) and SHOULD leverage
[gRPC](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc)
and
[http](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/confighttp)
helper functions.

## Safeguards against denial of service attacks

See the [Collector configuration security documentation](https://opentelemetry.io/docs/security/config-best-practices/#protect-against-denial-of-service-attacks) to learn how to safeguard against denial of service attacks. 

## Extensions

Component developers SHOULD NOT expose health or telemetry data outside the
Collector by default.
