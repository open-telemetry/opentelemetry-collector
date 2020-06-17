# Changelog

## Unreleased

## v0.4.0 Beta

Released 2020-06-16

## 🛑 Breaking changes 🛑

- `isEnabled` configuration option removed (#909) 
- `thrift_tchannel` protocol moved from `jaeger` receiver to `jaeger_legacy` in contrib (#636) 

## ⚠️ Major changes ⚠️

- Switch from `localhost` to `0.0.0.0` by default for all receivers (#1006)
- Internal API Changes (only impacts contributors)
  - Add context to `Start` and `Stop` methods in the component (#790)
  - Rename `AttributeValue` and `AttributeMap` method names (#781)
(other breaking changes in the internal trace data types)
  - Change entire repo to use the new vanityurl go.opentelemetry.io/collector (#977)

## 🚀 New components 🚀

- Receivers
  - `hostmetrics` receiver with CPU (#862), disk (#921), load (#974), filesystem (#926), memory (#911), network (#930), and virtual memory (#989) support
- Processors
  - `batch` for batching received metrics (#1060) 
  - `filter` for filtering (dropping) received metrics (#1001) 

## 💡 Enhancements 💡

- `otlp` receiver implement HTTP X-Protobuf (#1021)
- Exporters: Support mTLS in gRPC exporters (#927) 
- Extensions: Add `zpages` for service (servicez, pipelinez, extensions) (#894) 

## 🧰 Bug fixes 🧰

- Add missing logging for metrics at `debug` level (#1108) 
- Fix setting internal status code in `jaeger` receivers (#1105) 
- `zipkin` export fails on span without timestamp when used with `queued_retry` (#1068) 
- Fix `zipkin` receiver status code conversion (#996) 
- Remove extra send/receive annotations with using `zipkin` v1 (#960)
- Fix resource attribute mutation bug when exporting in `jaeger` proto (#907) 
- Fix metric/spans count, add tests for nil entries in the slices (#787) 


## 🧩 Components 🧩

### Traces

| Receivers | Processors | Exporters |
|:----------:|:-----------:|:----------:|
| Jaeger | Attributes | File |
| OpenCensus | Batch | Jaeger |
| OTLP | Memory Limiter | Logging |
| Zipkin | Queued Retry | OpenCensus |
| | Resource | OTLP |
| | Sampling | Zipkin |
| | Span ||

### Metrics

| Receivers | Processors | Exporters |
|:----------:|:-----------:|:----------:|
| HostMetrics | Batch | File |
| OpenCensus | Filter | Logging |
| OTLP | Memory Limiter | OpenCensus |
| Prometheus || OTLP |
| VM Metrics || Prometheus |

### Extensions

- Health Check
- Performance Profiler
- zPages


## v0.3.0 Beta

Released 2020-03-30

### Breaking changes

-  Make prometheus receiver config loading strict. #697 
Prometheus receiver will now fail fast if the config contains unused keys in it.

### Changes and fixes

- Enable best effort serve by default of Prometheus Exporter (https://github.com/orijtech/prometheus-go-metrics-exporter/pull/6)
- Fix null pointer exception in the logging exporter #743 
- Remove unnecessary condition to have at least one processor #744 

### Components

| Receivers / Exporters | Processors | Extensions |
|:---------------------:|:-----------:|:-----------:|
| Jaeger | Attributes | Health Check |
| OpenCensus | Batch | Performance Profiler |
| OpenTelemetry | Memory Limiter | zPages |
| Zipkin | Queued Retry | |
| | Resource | |
| | Sampling | |
| | Span | |


## v0.2.8 Alpha

Alpha v0.2.8 of OpenTelemetry Collector

- Implemented OTLP receiver and exporter.
- Added ability to pass config to the service programmatically (useful for custom builds).
- Improved own metrics / observability.
- Refactored component and factory interface definitions (breaking change #683) 


## v0.2.7 Alpha

Alpha v0.2.7 of OpenTelemetry Collector

- Improved error handling on shutdown
- Partial implementation of new metrics (new obsreport package)
- Include resource labels for Zipkin exporter
- New `HASH` action to attribute processor



## v0.2.6 Alpha

Alpha v0.2.6 of OpenTelemetry Collector.
- Update metrics prefix to `otelcol` and expose command line argument to modify the prefix value.
- Extend Span processor to have include/exclude span logic.
- Batch dropped span now emits zero when no spans are dropped.


## v0.2.5 Alpha

Alpha v0.2.5 of OpenTelemetry Collector.

- Regexp-based filtering of spans based on service names.
- Ability to choose strict or regexp matching for include/exclude filters.


## v0.2.4 Alpha

Alpha v0.2.4 of OpenTelemetry Collector.

- Regexp-based filtering of span names.
- Ability to extract attributes from span names and rename span.
- File exporter for debugging.
- Span processor is now enabled by default.


## v0.2.3 Alpha

Alpha v0.2.3 of OpenTelemetry Collector.

Changes:
21a70d6 Add a memory limiter processor (#498)
9778b16 Refactor Jaeger Receiver config (#490)
ec4ad0c Remove workers from OpenCensus receiver implementation (#497)
4e01fa3 Update k8s config to use opentelemetry docker image and configuration (#459)


## v0.2.2 Alpha

Alpha v0.2.2 of OpenTelemetry Collector.

Main changes visible to users since previous release:

- Improved Testbed and added more E2E tests.
- Made component interfaces more uniform (this is a breaking change).

Note: v0.2.1 never existed and is skipped since it was tainted in some dependencies.


## v0.2.0 Alpha

Alpha v0.2 of OpenTelemetry Collector.

Docker image: omnition/opentelemetry-collector:v0.2.0 (we are working on getting this under an OpenTelemetry org)

Main changes visible to users since previous release:

* Rename from `service` to `collector`, the binary is now named `otelcol`

* Configuration reorganized and using strict mode

* Concurrency issues for pipelines transforming data addressed

Commits:

```terminal
0e505d5 Refactor config: pipelines now under service (#376)
402b80c Add Capabilities to Processor and use for Fanout cloning decision (#374)
b27d824 Use strict mode to read config (#375)
d769eb5 Fix concurrency handling when data is fanned out (#367)
dc6b290 Rename all github paths from opentelemtry-service to opentelemetry-collector (#371)
d038801 Rename otelsvc to otelcol (#365)
c264e0e Add Include/Exclude logic for Attributes Processor (#363)
8ce427a Pin a commit for Prometheus dependency in go.mod (#364)
2393774 Bump Jaeger version to 1.14.0 (latest) (#349)
63362d5 Update testbed modules (#360)
c0e2a27 Change dashes to underscores to separate words in config files (#357)
7609eaa Rename OpenTelemetry Service to Collector in docs and comments (#354)
bc5b299 Add common gRPC configuration settings (#340)
b38505c Remove network access popups on macos (#348)
f7727d1 Fixed loop variable pointer bug in jaeger translator (#341)
958beed Ensure that ConsumeMetricsData() is not passed empty metrics in the Prometheus receiver (#345)
0be295f Change log statement in Prometheus receiver from info to debug. (#344)
d205393 Add Owais to codeowners (#339)
8fa6afe Translate OC resource labels to Jaeger process tags (#325)
```


## v0.0.2 Alpha

Alpha release of OpenTelemetry Service.

Docker image: omnition/opentelemetry-service:v0.0.2 (we are working on getting this under an OpenTelemetry org)

Main changes visible to users since previous release:

```terminal
8fa6afe Translate OC resource labels to Jaeger process tags (#325)
047b0f3 Allow environment variables in config (#334)
96c24a3 Add exclude/include spans option to attributes processor (#311)
4db0414 Allow metric processors to be specified in pipelines (#332)
c277569 Add observability instrumentation for Prometheus receiver (#327)
f47aa79 Add common configuration for receiver tls (#288)
a493765 Refactor extensions to new config format (#310)
41a7afa Add Span Processor logic
97a71b3 Use full name for the metrics and spans created for observability (#316)
fed4ed2 Add support to record metrics for metricsexporter (#315)
5edca32 Add include_filter configuration to prometheus receiver (#298)
0068d0a Passthrough CORS allowed origins (#260)
```


## v0.0.1 Alpha

This is the first alpha release of OpenTelemetry Service.

Docker image: omnition/opentelemetry-service:v0.0.1


[v0.3.0]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.10...v0.3.0
[v0.2.10]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.8...v0.2.10
[v0.2.8]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.7...v0.2.8
[v0.2.7]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.6...v0.2.7
[v0.2.6]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.5...v0.2.6
[v0.2.5]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.4...v0.2.5
[v0.2.4]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.3...v0.2.4
[v0.2.3]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.2...v0.2.3
[v0.2.2]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.2.0...v0.2.2
[v0.2.0]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.0.2...v0.2.0
[v0.0.2]: https://github.com/open-telemetry/opentelemetry-collector/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.0.1