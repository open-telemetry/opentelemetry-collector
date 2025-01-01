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
file passed to the Collector MUST be validated prior to being loaded. If an invalid
configuration is detected, the Collector MUST fail to start as a protective
mechanism.

The configuration drives the Collector's behavior and care should be taken to
ensure the configuration only enables the minimum set of capabilities and as
such exposes the minimum set of required ports. In addition, any incoming or
outgoing communication SHOULD leverage TLS and authentication.

The Collector keeps the configuration in memory, but where the configuration is
loaded from at start time depends on the packaging used. For example, in
Kubernetes secrets and configmaps CAN be leveraged. In comparison, the Docker
image embeds the configuration in the container where is it not stored in an
encrypted manner by default.

The configuration MAY contain sensitive information including:

- Authentication information such as API tokens
- TLS certificates including private keys

Sensitive information SHOULD be stored securely such as on an encrypted
filesystem or secret store. Environment variables CAN be used to handle
sensitive and non-sensitive data as the Collector MUST support environment
variable expansion.

> For more information on environment variable expansion, see
> [this](https://opentelemetry.io/docs/collector/configuration/#environment-variables)
> documentation.

When defining Go structs for configuration data that may contain sensitive
information, use the `configopaque` package to define fields with the
`configopaque.String` type. This ensures that the data is masked when serialized
to prevent accidental exposure.

> For more information, see the
> [configopaque](https://pkg.go.dev/go.opentelemetry.io/collector/config/configopaque)
> documentation.

Component developers MUST get configuration information from the Collector's
configuration file. Component developers SHOULD leverage
[configuration helper functions](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config).

More information about configuration is provided in the following sections.

## Permissions

The Collector supports running as a custom user and SHOULD NOT be run as a
root/admin user. For the majority of use-cases, the Collector SHOULD NOT require
privileged access to function. Some components MAY require privileged access and
care should be taken before enabling these components. Collector components MAY
require external permissions including network access or RBAC.

Component developers SHOULD minimize privileged access requirements and MUST
document what requires privileged access and why.

More information about permissions is provided in the following sections.

## Receivers and Exporters

Receivers and Exporters can be either push or pull-based. In either case, the
connection established SHOULD be over a secure and authenticated channel. Unused
receivers and exporters SHOULD be disabled to minimize the attack vector of the
Collector.

Receivers and Exporters MAY expose buffer, queue, payload, and/or worker
settings via configuration parameters. If these settings are available,
end-users should proceed with caution before modifying the default values.
Improperly setting these values may expose the Collector to additional attack
vectors including resource exhaustion.

> It is possible that a receiver MAY require the Collector run in a privileged
> mode in order to operate, which could be a security concern, but today this is
> not the case.

Component developers MUST default to encrypted connections (via the
`insecure: false` configuration setting) and SHOULD leverage
[gRPC](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc)
and
[http](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/confighttp)
helper functions.

## Extensions

While receivers, processors, and exporters handle telemetry data directly,
extensions typical serve different needs.

### Health and Telemetry

Component developers SHOULD NOT expose health or telemetry data outside the
Collector by default.
