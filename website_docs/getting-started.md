---
title: "Getting Started"
weight: 1
---
<!-- markdown-link-check-disable -->
Please be sure to review the [Data Collection
documentation](../../concepts/data-collection) in order to understand the
deployment models, components, and repositories applicable to the OpenTelemetry
Collector.
<!-- markdown-link-check-enable -->
## Deployment

The OpenTelemetry Collector consists of a single binary and two primary deployment methods:

- **Agent:** A Collector instance running with the application or on the same
  host as the application (e.g. binary, sidecar, or daemonset).
- **Gateway:** One or more Collector instances running as a standalone service
  (e.g. container or deployment) typically per cluster, datacenter or region.

### Agent

It is recommended to deploy the Agent on every host within an environment. In
doing so, the Agent is capable of receiving telemetry data (push and pull
based) as well as enhancing telemetry data with metadata such as custom tags or
infrastructure information. In addition, the Agent can offload responsibilities
that client instrumentation would otherwise need to handle including batching,
retry, encryption, compression and more. OpenTelemetry instrumentation
libraries by default export their data assuming a locally running Collector is
available.

### Gateway

Additionally, a Gateway cluster can be deployed in every cluster, datacenter,
or region. A Gateway cluster runs as a standalone service and can offer
advanced capabilities over the Agent including tail-based sampling. In
addition, a Gateway cluster can limit the number of egress points required to
send data as well as consolidate API token management. Each Collector instance
in a Gateway cluster operates independently so it is easy to scale the
architecture based on performance needs with a simple load balancer. If a
gateway cluster is deployed, it usually receives data from Agents deployed
within an environment.

## Getting Started

### Demo

Deploys a load generator, agent and gateway as well as Jaeger, Zipkin and
Prometheus back-ends. More information can be found on the demo
[README.md](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/examples/demo)

```bash
$ git clone git@github.com:open-telemetry/opentelemetry-collector-contrib.git; \
    cd opentelemetry-collector-contrib/examples/demo; \
    docker-compose up -d
```

### Docker

Every release of the Collector is published to Docker Hub and comes with a
default configuration file.

```bash
$ docker run otel/opentelemetry-collector
```

In addition, you can use the local example provided. This example starts a
Docker container of the
[core](https://github.com/open-telemetry/opentelemetry-collector) version of
the Collector with all receivers enabled and exports all the data it receives
locally to a file. Data is sent to the container and the container scrapes its
own Prometheus metrics.

```bash
$ git clone git@github.com:open-telemetry/opentelemetry-collector.git; \
    cd opentelemetry-collector/examples; \
    go build main.go; ./main & pid1="$!";
    docker run --rm -p 13133:13133 -p 14250:14250 -p 14268:14268 \
      -p 55678-55679:55678-55679 -p 4317:4317 -p 8888:8888 -p 9411:9411 \
      -v "${PWD}/otel-local-config.yaml":/otel-local-config.yaml \
      --name otelcol otel/opentelemetry-collector \
      --config otel-local-config.yaml; \
    kill $pid1; docker stop otelcol
```

### Kubernetes

Deploys an agent as a daemonset and a single gateway instance.

```bash
$ kubectl apply -f https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/main/examples/k8s/otel-config.yaml
```

The example above is meant to serve as a starting point, to be extended and
customized before actual production usage.

The [OpenTelemetry
Operator](https://github.com/open-telemetry/opentelemetry-operator) can also be
used to provision and maintain an OpenTelemetry Collector instance, with
features such as automatic upgrade handling, `Service` configuration based on
the OpenTelemetry configuration, automatic sidecar injection into deployments,
among others.

### Nomad

Reference job files to deploy the Collector as an agent, gateway and in the
full demo can be found at
[https://github.com/hashicorp/nomad-open-telemetry-getting-started](https://github.com/hashicorp/nomad-open-telemetry-getting-started).

### Linux Packaging

Every Collector release includes DEB and RPM packaging for Linux amd64/arm64
systems. The packaging includes a default configuration that can be found at
`/etc/otel-collector/config.yaml` post-installation.

> Please note that systemd is require for automatic service configuration

To get started on Debian systems run the following replacing `v0.20.0` with the
version of the Collector you wish to run and `amd64` with the appropriate
architecture.

```bash
$ sudo apt-get update
$ sudo apt-get -y install wget systemctl
$ wget https://github.com/open-telemetry/opentelemetry-collector/releases/download/v0.20.0/otel-collector_0.20.0_amd64.deb
$ dpkg -i otel-collector_0.20.0_amd64.deb
```

To get started on Red Hat systems run the following replacing `v0.20.0` with the
version of the Collector you wish to run and `x86_64` with the appropriate
architecture.

```bash
$ sudo yum update
$ sudo yum -y install wget systemctl
$ wget https://github.com/open-telemetry/opentelemetry-collector/releases/download/v0.20.0/otel-collector_0.20.0-1_x86_64.rpm
$ rpm -ivh otel-collector_0.20.0-1_x86_64.rpm
```

By default, the `otel-collector` systemd service will be started with the
`--config=/etc/otel-collector/config.yaml` option after installation.  To
customize these options, modify the `OTELCOL_OPTIONS` variable in the
`/etc/otel-collector/otel-collector.conf` systemd environment file with the
appropriate command-line options (run `/usr/bin/otelcol --help` to see all
available options).  Additional environment variables can also be passed to the
`otel-collector` service by adding them to this file.

If either the Collector configuration file or
`/etc/otel-collector/otel-collector.conf` are modified, restart the
`otel-collector` service to apply the changes by running:

```bash
$ sudo systemctl restart otel-collector
```

To check the output from the `otel-collector` service, run:

```bash
$ sudo journalctl -u otel-collector
```

### Windows Packaging

Every Collector release includes EXE and MSI packaging for Windows amd64 systems.
The MSI packaging includes a default configuration that can be found at
`\Program Files\OpenTelemetry Collector\config.yaml`.

> Please note the Collector service is not automatically started

The easiest way to get started is to double-click the MSI package and follow
the wizard. Silent installation is also available.

### Local

Builds the latest version of the collector based on the local operating system,
runs the binary with all receivers enabled and exports all the data it receives
locally to a file. Data is sent to the container and the container scrapes its own
Prometheus metrics.

```bash
$ git clone git@github.com:open-telemetry/opentelemetry-collector-contrib.git; \
    cd opentelemetry-collector-contrib/examples/demo; \
    go build client/main.go; ./client/main & pid1="$!"; \
    go build server/main.go; ./server/main & pid2="$!"; \

$ git clone git@github.com:open-telemetry/opentelemetry-collector.git; \
    cd opentelemetry-collector; make install-tools; make otelcol; \
    ./bin/otelcol_$(go env GOOS)_$(go env GOARCH) --config ./examples/local/otel-config.yaml; kill $pid1; kill $pid2
```
