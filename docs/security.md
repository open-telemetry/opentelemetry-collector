# Security

The OpenTelemetry Collector defaults to operating in a secure manner, but is
configuration driven. This document captures important security aspects and
considerations for the Collector.

## Configuration

The Collector does not contain an embedded or default configuration and MUST
NOT start without a configuration file being specified. The configuration file
passed to the Collector MUST be validated prior to be loaded. If an invalid
configuration is detected, the Collector MUST fail to start as a protective
mechanism.

The configuration drives the Collector's behavior and care should be taken to
ensure the configuration only enables the minimum set of capabilities and as
such exposes the minimum set of required ports. In addition, any incoming or
outgoing communication SHOULD leverage TLS and/or authentication.

The Collector keeps the configuration in memory, but where the configuration is
loaded from at start time depends on the packaging used. For example, in
Kubernetes secrets and configmaps CAN be leveraged. In comparison, the Docker
image embeds the configuration in the container where is it not stored in an
encrypted manner by default.

The configuration MAY contain sensitive information including:

- Authentication information such as API tokens
- TLS certificates including private keys

Sensitive information SHOULD be stored securely such as on an encrypted
filesystem or secret store. Environment variable CAN be used to handle
sensitive and non-sensitive data as the Collector MUST support environment
variable expansion.

More information about configuration is provided in the following sections.

## Permissions

The Collector supports running as a custom user and SHOULD NOT be run as a
root/admin user. For the majority of use-cases, the Collector SHOULD NOT require
privileged access to function. Some components MAY require privileged access
and care should be taken before enabling these components. Collector components
MAY require external permissions including network access or RBAC access.

More information about permissions is provided in the following sections.

## Receivers and Exporters

Receivers and Exporters can be either push or pull-based. In either case, the
connection established SHOULD be over a secure and authenticated channel.
Unused receivers and exporters SHOULD be disabled to minimize the attack vector
of the Collector. Receivers and exporters MUST default to TLS connections (via
the `insecure: False` configuration setting).

Receivers and Exporters MAY expose buffer, queue, payload, and/or worker
settings. If these settings are available, care should be taken before
modifying the default values. Improperly setting these values may expose the
Collector to additional attack vectors including resource exhaustion.

> It is possible that a receiver MAY require the Collector run in a privileged
> mode in order to operate, which could be a security concern, but today this
> is not the case.

## Processors

Processors sit between receivers and exporters. They are responsible for
processing the data in some way. From a security perspective, they are useful
in a couple ways.

### Scrubbing sensitive data

It is common for a Collector to be used to scrub sensitive data before
exporting it to a backend. This is especially important when sending the data
to a third-party backend.

### Safeguards around resource utilization

In addition, processors offer safeguards around resource utilization. The
`batch` and especially `memory_limiter` processor help ensure that the
Collector is resource efficient and does not OOM when overloaded. At least
these two processors SHOULD be enabled on every defined pipeline.

## Extensions

While receivers, processors, and exporters handle telemetry data directly,
extensions typical serve different needs.

### Health and Telemetry

The initial extensions provided health check information, Collector metrics and
traces, and the ability to generate and collect profiling data. When enabled
with their default settings, all of these extensions except the health check
extension are only accessibly locally to the Collector. Care should be taken
when configuring these extensions for remote access as sensitive information
may be exposed as a result.

### Forwarding

A forwarding extension is typically used when some telemetry data not natively
supported by the Collector needs to be collected. For example, the
`http_forwarder` extension can receive and forward HTTP payloads. Forwarding
extensions are similar to receivers and exporters so the same security
considerations apply.

### Observers

An observer is capable of doing service discovery of endpoints. Other
components of the collector such as receivers MAY subscribe to these extensions
to be notified of endpoints coming or going. Observers require certain
permissions in order to perform service discovery. For example, the
`k8s_observer` requires certain RBAC permissions in Kubernetes, while the
`host_observer` requires the Collector to run in privileged mode.

### Subprocesses

Extensions may also be used to run subprocesses. This can be useful when
collection mechanisms that cannot natively be run by the Collector (e.g.
FluentBit). Subprocesses expose a completely separate attack vendor that would
depend on the subprocess itself. In general, care should be taken before
running any subprocesses alongside the Collector.
