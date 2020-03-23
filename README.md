---

<p align="center">
  <strong>
    <a href="https://opentelemetry.io/docs/collector/about/">Getting Started<a/>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://github.com/open-telemetry/community#agentcollector">Getting Involved<a/>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="https://gitter.im/open-telemetry/opentelemetry-service">Getting In Touch<a/>
  </strong>
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/open-telemetry/opentelemetry-collector">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/open-telemetry/opentelemetry-collector?style=for-the-badge">
  </a>
  <a href="https://travis-ci.org/open-telemetry/opentelemetry-collector">
    <img alt="Build Status" src="https://img.shields.io/travis/open-telemetry/opentelemetry-collector?style=for-the-badge">
  </a>
  <a href="https://codecov.io/gh/open-telemetry/opentelemetry-collector/branch/master/">
    <img alt="Codecov Status" src="https://img.shields.io/codecov/c/github/open-telemetry/opentelemetry-collector?style=for-the-badge">
  </a>
  <a href="releases">
    <img alt="GitHub release (latest by date including pre-releases)" src="https://img.shields.io/github/v/release/open-telemetry/opentelemetry-collector?include_prereleases&style=for-the-badge">
  </a>
</p>

<p align="center">
  <strong>
    <a href="CONTRIBUTING.md">Contributing<a/>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/design.md">Design<a/>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/monitoring.md">Monitoring<a/>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/performance.md">Performance<a/>
    &nbsp;&nbsp;&bull;&nbsp;&nbsp;
    <a href="docs/roadmap.md">Roadmap<a/>
  </strong>
</p>

---


### OpenTelemetry Collector

The OpenTelemetry Collector offers a vendor-agnostic implementation on how to receive, process, and export telemetry data. In addition, it removes the need to run, operate, and maintain multiple agents/collectors in order to support open-source telemetry data formats (e.g. Jaeger, Prometheus, etc.) sending to multiple open-source or commercial back-ends.

Objectives:

- Usable: Reasonable default configuration, supports popular protocols, runs and collects out of the box.
- Performant: Highly stable and performant under varying loads and configurations.
- Observable: An exemplar of an observable service.
- Extensible: Customizable without touching the core code.
- Unified: Single codebase, deployable as an agent or collector with support for traces, metrics, and logs (future).

### Community Roles

Approvers ([@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)):

- [Owais Lone](https://github.com/owais), Splunk
- [Rahul Patel](https://github.com/rghetia), Google
- [Steve Flanders](https://github.com/flands), Splunk
- [Steven Karis](https://github.com/sjkaris), Splunk
- [Yang Song](https://github.com/songy23), Google

Maintainers ([@open-telemetry/collector-maintainers](https://github.com/orgs/open-telemetry/teams/collector-maintainers)):

- [Bogdan Drutu](https://github.com/BogdanDrutu), Splunk
- [Paulo Janotti](https://github.com/pjanotti), Splunk
- [Tigran Najaryan](https://github.com/tigrannajaryan), Splunk

Learn more about roles in the [community repository](https://github.com/open-telemetry/community/blob/master/community-membership.md).
