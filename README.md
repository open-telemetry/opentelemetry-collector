---

<p align="center">
  <strong>
    <a href="https://opentelemetry.io/docs/collector/getting-started/">Getting Started</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="CONTRIBUTING.md">Getting Involved</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://gitter.im/open-telemetry/opentelemetry-service">Getting In Touch</a>
  </strong>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/open-telemetry/opentelemetry-collector">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/open-telemetry/opentelemetry-collector?style=for-the-badge"></a>
  <a href="https://circleci.com/gh/open-telemetry/opentelemetry-collector">
    <img alt="Build Status" src="https://img.shields.io/circleci/build/github/open-telemetry/opentelemetry-collector?style=for-the-badge"></a>
  <a href="https://codecov.io/gh/open-telemetry/opentelemetry-collector/branch/main/">
    <img alt="Codecov Status" src="https://img.shields.io/codecov/c/github/open-telemetry/opentelemetry-collector?style=for-the-badge"></a>
  <a href="https://github.com/open-telemetry/opentelemetry-collector/releases">
    <img alt="GitHub release (latest by date including pre-releases)" src="https://img.shields.io/github/v/release/open-telemetry/opentelemetry-collector?include_prereleases&style=for-the-badge"></a>
</p>

<p align="center">
  <strong>
    <a href="docs/vision.md">Vision</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/design.md">Design</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/monitoring.md">Monitoring</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/performance.md">Performance</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/security.md">Security</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/roadmap.md">Roadmap</a>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://pkg.go.dev/go.opentelemetry.io/collector">Package</a>
  </strong>
</p>

---

# <img src="https://opentelemetry.io/img/logos/opentelemetry-logo-nav.png" alt="OpenTelemetry Icon" width="45" height=""> OpenTelemetry Collector

The OpenTelemetry Collector offers a vendor-agnostic implementation on how to
receive, process and export telemetry data. In addition, it removes the need
to run, operate and maintain multiple agents/collectors in order to support
open-source telemetry data formats (e.g. Jaeger, Prometheus, etc.) sending to
multiple open-source or commercial back-ends.

Objectives:

- Usable: Reasonable default configuration, supports popular protocols, runs and collects out of the box.
- Performant: Highly stable and performant under varying loads and configurations.
- Observable: An exemplar of an observable service.
- Extensible: Customizable without touching the core code.
- Unified: Single codebase, deployable as an agent or collector with support for traces, metrics and logs.

## Status

The collector components and implementation are in different stages of stability, and usually split between
functionality and configuration:

| Component | Status |
|---|---|
| OTLP traces protocol | Stable |
| OTLP metrics protocol | Stable |
| OTLP logs protocol | Beta |
| OTLP traces receiver functionality | Stable |
| OTLP metrics receiver functionality | Stable |
| OTLP logs receiver functionality | Beta |
| OTLP traces receiver configuration | Beta |
| OTLP metrics receiver configuration | Beta |
| OTLP logs receiver configuration | Beta |
| OTLP traces exporter functionality | Stable |
| OTLP metrics exporter functionality | Stable |
| OTLP logs exporter functionality | Beta |
| OTLP traces exporter configuration | Beta |
| OTLP metrics exporter configuration | Beta |
| OTLP logs exporter configuration | Beta |
| Logging exporter | Experimental |
| OTLP logs receiver functionality | Beta |
| OTLP traces receiver configuration | Beta |
| Batch processor functionality | Beta |
| Batch processor configuration | Beta |
| MemoryLimiter processor functionality | Beta |
| MemoryLimiter processor configuration | Beta |

The collector public APIs are in different stages of stability (see more details about the [versioning](VERSIONING.md)):

| Package | Status |
|---|---|
| go.opentelemetry.io/collector/model | RC |
| go.opentelemetry.io/collector | Beta |

See more details about the status:

* **Experimental** components are under design, and have not been added to the specification.
* **Beta** components are released and available for beta testing.
* **RC** components are released and available for final testing before being marked as stable.
* **Stable** components are backwards compatible and covered under long term support.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

Triagers ([@open-telemetry/collector-triagers](https://github.com/orgs/open-telemetry/teams/collector-triagers)):

- [Alolita Sharma](https://github.com/alolita), AWS
- [Punya Biswal](https://github.com/punya), Google
- [Steve Flanders](https://github.com/flands), Splunk

Approvers ([@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)):

- [Alex Boten](https://github.com/codeboten), Lightstep
- [Anthony Mirabella](https://github.com/Aneurysm9), AWS
- [Dmitrii Anoshin](https://github.com/dmitryax), Splunk
- [Juraci Paixão Kröhling](https://github.com/jpkrohling), Red Hat
- [Owais Lone](https://github.com/owais), Splunk

Maintainers ([@open-telemetry/collector-maintainers](https://github.com/orgs/open-telemetry/teams/collector-maintainers)):

- [Bogdan Drutu](https://github.com/BogdanDrutu), Splunk
- [Tigran Najaryan](https://github.com/tigrannajaryan), Splunk

Learn more about roles in the [community repository](https://github.com/open-telemetry/community/blob/main/community-membership.md).

Thanks to all the people who already contributed!

<a href="https://github.com/open-telemetry/opentelemetry-collector/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-collector" />
</a>
