# Security

The OpenTelemetry Collector defaults to operating in a secure manner, but is
configuration driven. This document captures important security aspects and
considerations for the Collector. This document is intended for both end-users
and component developers. It assumes at least a basic understanding of the
Collector architecture and functionality.

> Note: Please review the [configuration
> documentation](https://opentelemetry.io/docs/collector/configuration/)
> prior to this security document.

## TL;DR

### End-users

- Configuration
  - SHOULD only enable the minimum required components
  - SHOULD ensure sensitive configuration information is stored securely
- Permissions
  - SHOULD not run Collector as root/admin user
  - MAY require privileged access for some components
- Receivers/Exporters
  - SHOULD use encryption and authentication
  - SHOULD limit exposure of servers to authorized users
  - MAY pose a security risk if configuration parameters are modified improperly
- Processors
  - SHOULD configure obfuscation/scrubbing of sensitive metadata
  - SHOULD configure recommended processors
- Extensions
  - SHOULD NOT expose sensitive health or telemetry data

> For more information about securing the OpenTelemetry Collector, see
> [this](https://medium.com/opentelemetry/securing-your-opentelemetry-collector-1a4f9fa5bd6f)
> blog post.

### Component Developers

- Configuration
  - MUST come from the central configuration file
  - SHOULD use configuration helpers
- Permissions
  - SHOULD minimize privileged access
  - MUST document what required privileged access and why
- Receivers/Exporters
  - MUST default to encrypted connections
  - SHOULD leverage helper functions
- Extensions
  - SHOULD NOT expose sensitive health or telemetry data by default

## Configuration

The Collector binary does not contain an embedded or default configuration and
MUST NOT start without a configuration file being specified. The configuration
file passed to the Collector MUST be validated prior to be loaded. If an
invalid configuration is detected, the Collector MUST fail to start as a
protective mechanism.

> Note: Issue
> [#886](https://github.com/open-telemetry/opentelemetry-collector/issues/886)
> proposes adding a default configuration to the binary.

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
> [this](https://opentelemetry.io/docs/collector/configuration/#configuration-environment-variables)
> documentation.

When defining Go structs for configuration data that may contain sensitive information, use the `configopaque` package to define fields with the `configopaque.String` type. This ensures that the data is masked when serialized to prevent accidental exposure.

> For more information, see the [configopaque](https://pkg.go.dev/go.opentelemetry.io/collector/config/configopaque) documentation.

Component developers MUST get configuration information from the Collector's
configuration file. Component developers SHOULD leverage [configuration helper
functions](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config).

More information about configuration is provided in the following sections.

## Permissions

The Collector supports running as a custom user and SHOULD NOT be run as a
root/admin user. For the majority of use-cases, the Collector SHOULD NOT require
privileged access to function. Some components MAY require privileged access
and care should be taken before enabling these components. Collector components
MAY require external permissions including network access or RBAC.

Component developers SHOULD minimize privileged access requirements and MUST
document what requires privileged access and why.

More information about permissions is provided in the following sections.

## Receivers and Exporters

Receivers and Exporters can be either push or pull-based. In either case, the
connection established SHOULD be over a secure and authenticated channel.
Unused receivers and exporters SHOULD be disabled to minimize the attack vector
of the Collector.

Receivers and Exporters MAY expose buffer, queue, payload, and/or worker
settings via configuration parameters. If these settings are available,
end-users should proceed with caution before modifying the default values.
Improperly setting these values may expose the Collector to additional attack
vectors including resource exhaustion.

> It is possible that a receiver MAY require the Collector run in a privileged
> mode in order to operate, which could be a security concern, but today this
> is not the case.

Component developers MUST default to encrypted connections (via the `insecure:
false` configuration setting) and SHOULD leverage
[gRPC](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc)
and
[http](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/confighttp)
helper functions.

### Safeguards against denial of service attacks

Users SHOULD bind receivers' servers to addresses that limit connections to authorized users.
For example, if the OTLP receiver OTLP/gRPC server only has local clients, the `endpoint` setting SHOULD be bound to `localhost`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: localhost:4317
```

Generally, `localhost`-like addresses should be preferred over the 0.0.0.0 address.
For more information, see [CWE-1327](https://cwe.mitre.org/data/definitions/1327.html).

To change the default endpoint to be `localhost`-bound in all components, enable the `component.UseLocalHostAsDefaultHost` feature gate. This feature gate will be enabled by default in the Collector in a future release.


If `localhost` resolves to a different IP due to your DNS settings then explicitly use the loopback IP instead: `127.0.0.1` for IPv4 or `::1` for IPv6. In IPv6 setups, ensure your system supports both IPv4 and IPv6 loopback addresses to avoid issues.

Using `localhost` may not work in environments like Docker, Kubernetes, and other environments that have non-standard networking setups.  We've documented a few working example setups for the OTLP receiver gRPC endpoint below, but other receivers and other Collector components may need similar configuration.

#### Docker
You can run the Collector in Docker by binding to the correct address.  An OTLP exporter in Docker might look something like this:

Collector config file 

`config.yaml`:
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: my-hostname:4317 # the same hostname from your docker run command
```
Docker run command:
`docker run --hostname my-hostname --name container-name -p 127.0.0.1:4567:4317 otel/opentelemetry-collector:0.104.0`

The key here is using the `--hostname` argument - that allows the collector to bind to the `my-hostname` address.
You could access it from outside that Docker network (for example on a regular program running on the host) by connecting to `127.0.0.1:4567`.

#### Docker Compose
Similarly to plain Docker, you can run the Collector in Docker by binding to the correct address.

`compose.yaml`:
```yaml 
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.104.0
    ports:
      - "4567:4317"
```

Collector config file:

`config.yaml`:
```yaml
receivers:
  otlp:
    protocols:
      grpc:
         endpoint: otel-collector:4317 # Using the service name from your Docker compose file
```

You can connect to this Collector from another Docker container running in the same network by connecting to `otel-collector:4317`.  You could access it from outside that Docker network (for example on a regular program running on the host) by connecting to `127.0.0.1:4567`.

#### Kubernetes
If you run the Collector as a `Daemonset`, you can use a configuration like below:
```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: collector
spec:
  selector:
    matchLabels:
      name: collector
  template:
    metadata:
      labels:
        name: collector
    spec:
      containers:
        - name: collector
          image: otel/opentelemetry-collector:0.104.0
          ports:
            - containerPort: 4317
              hostPort: 4317
              protocol: TCP
              name: otlp-grpc
            - containerPort: 4318
              hostPort: 4318
              protocol: TCP
              name: otlp-http
          env:
            - name: MY_POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP

```
In this example, we use the [Kubernetes Downward API](https://kubernetes.io/docs/concepts/workloads/pods/downward-api/) to get your own Pod IP, then bind to that network interface.  Then, we use the `hostPort` option to ensure that the Collector is exposed on the host.  The Collector's config should look something like:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: ${env:MY_POD_IP}:4317
      http:
        endpoint: ${env:MY_POD_IP}:4318
```

You can send OTLP data to this Collector from any Pod on the Node by accessing `${MY_HOST_IP}:4317` to send OTLP over gRPC and `${MY_HOST_IP}:4318`  to send OTLP over HTTP, where `MY_HOST_IP` is the Node's IP address.  You can get this IP from the Downwards API:

```yaml
env:
  - name: MY_HOST_IP
    valueFrom:
      fieldRef:
        fieldPath: status.hostIP
```

## Processors

Processors sit between receivers and exporters. They are responsible for
processing the data in some way. From a security perspective, they are useful
in a couple ways.

### Scrubbing sensitive data

It is common for a Collector to be used to scrub sensitive data before
exporting it to a backend. This is especially important when sending the data
to a third-party backend. The Collector SHOULD be configured to obfuscate or
scrub sensitive data before exporting.

> Note: Issue
> [#2466](https://github.com/open-telemetry/opentelemetry-collector/issues/2466)
> proposes adding default obfuscation or scrubbing of known sensitive metadata.

### Safeguards around resource utilization

In addition, processors offer safeguards around resource utilization. The
`batch` and especially `memory_limiter` processor help ensure that the
Collector is resource efficient and does not run out of memory when overloaded. At
least these two processors SHOULD be enabled on every defined pipeline.

> For more information on recommended processors and order, see
> [this](https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor)
> documentation.

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

Component developers SHOULD NOT expose health or telemetry data outside the
Collector by default.

### Forwarding

A forwarding extension is typically used when some telemetry data not natively
supported by the Collector needs to be collected. For example, the
`http_forwarder` extension can receive and forward HTTP payloads. Forwarding
extensions are similar to receivers and exporters so the same security
considerations apply.

### Observers

An observer is capable of performing service discovery of endpoints. Other
components of the collector such as receivers MAY subscribe to these extensions
to be notified of endpoints coming or going. Observers MAY require certain
permissions in order to perform service discovery. For example, the
`k8s_observer` requires certain RBAC permissions in Kubernetes, while the
`host_observer` requires the Collector to run in privileged mode.

### Subprocesses

Extensions may also be used to run subprocesses. This can be useful when
collection mechanisms that cannot natively be run by the Collector (e.g.
FluentBit). Subprocesses expose a completely separate attack vector that would
depend on the subprocess itself. In general, care should be taken before
running any subprocesses alongside the Collector.