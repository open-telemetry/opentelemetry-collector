---
title: Collector
weight: 10
description: >-
  <img width="35" src="https://raw.github.com/open-telemetry/opentelemetry.io/main/iconography/32x32/Collector.svg" alt="Collector logo"></img>
  Vendor-agnostic way to receive, process and export telemetry data.
cascade:
  github_repo: &repo https://github.com/open-telemetry/opentelemetry-collector
  github_subdir: website_docs
  path_base_for_github_subdir: content/en/docs/collector/
  github_project_repo: *repo
---

<img src="https://raw.github.com/open-telemetry/opentelemetry.io/main/iconography/Otel_Collector.svg" alt="Otel-Collector diagram with Jaeger, OTLP and Prometheus integration"></img>

The OpenTelemetry Collector offers a vendor-agnostic implementation on how to
receive, process and export telemetry data. It removes the need to run,
operate, and maintain multiple agents/collectors. This works with improved scalability and supports
open-source observability data formats (e.g. Jaeger, Prometheus, Fluent Bit,
etc.) sending to one or more open-source or commercial back-ends. The Collector
is the default location instrumentation libraries export their telemetry data.

Objectives:

- *Usability*: Reasonable default configuration, supports popular protocols, runs and collects out of the box.
- *Performance*: Highly stable and performant under varying loads and configurations.
- *Observability*: An exemplar of an observable service.
- *Extensibility*: Customizable without touching the core code.
- *Unification*: Single codebase, deployable as an agent or collector with support for traces, metrics, and logs (future).
