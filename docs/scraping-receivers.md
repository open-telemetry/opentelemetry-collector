# Scraping Metrics Receivers

Scraping metrics receivers are receivers that pull data from external sources at regular intervals and translate it
into [pdata](../pdata/README.md) which is sent further in the pipeline. The external source of metrics usually is a
monitored system providing data about itself in some arbitrary format. There are two types of scraping metrics
receivers:

- **Generic scraping metrics receivers:** The set of metrics emitted by this type of receiver fully depends on the 
  state of the external source and/or the user settings. Examples:
  - [Prometheus Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver)
  - [SQL Query Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/sqlqueryreceiver)

- **Built-in scraping metrics receivers:** Receivers of this type emit a predefined set of metrics. However, the 
  metrics themselves are configurable via user settings. Examples of scrapings metrics receivers:
  - [Redis Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/redisreceiver)
  - [Zookeeper Receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/zookeeperreceiver)

This document covers built-in scraping metrics receivers. It defines which metrics these receivers can emit, 
defines stability guarantees and provides guidelines for metric updates.

## Defining emitted metrics 

Each built-in scraping metrics receiver has a `metadata.yaml` file that MUST define all the metrics emitted by the 
receiver. The file is being used to generate an API for metrics recording, user settings to customize the emitted 
metrics and user documentation. The file schema is defined in 
https://github.com/open-telemetry/opentelemetry-collector/blob/main/cmd/mdatagen/metadata-schema.yaml
Defining a metric in `metadata.yaml` DOES NOT guarantee that the metric will always be produced by the receiver. In
some cases it may be impossible to fetch particular metrics from a system in a particular state.

There are two categories of the metrics emitted by scraping receivers:

- **Default metrics**: emitted by default, but can be disabled in user settings.

- **Optional metrics**: not emitted by default, but can be enabled in user settings.

See also [Semantic Convention compatibility](./coding-guidelines.md#semantic-conventions-compatibility) guidance.

### How to identify if new metric should be default or optional?

There is no strict rule to differentiate default metrics from optional. As a rule of thumb, default metrics SHOULD be 
treated as metrics essential to determine healthiness of a particular monitored system. Optional metrics on the 
other hand SHOULD be treated as a source of additional information about the monitored system. 

Additionally, if any of the following conditions can be applied to a metric, it MUST be marked as optional:

- **It is redundant with another metric.** For example system CPU usage can be emitted as two different metrics:
  `system.cpu.time` (CPU time reported as cumulative sum in seconds) or `system.cpu.utilization` (fraction of CPU 
  time spent in different states reported as gauge), one of them has to be marked as option.
- **It creates a disproportionately high cardinality of resources and/or data points.**
- **There is a notable expected performance impact.**
- **The source must be configured in an unusual way.**
- **It requires dedicated configuration in the receiver.**

## Stability levels of scraping receivers

All the requirements defined for components in [the Collector's README](../README.md#stability-levels) are
applicable to the scraping receivers as well. In addition, the following rules applied specifically to scraping
metrics receivers:

### Development

The receiver is not ready for use. All the metrics emitted by the receiver are not finalized and can change in any way.

### Alpha

The receiver is ready for limited non-critical workloads. The list of emitted default metrics SHOULD be 
considered as complete, but any changes to the `metadata.yaml` still MAY be applied.

### Beta

The receiver is ready for non-critical production workloads. The list of emitted default metrics MUST be
considered as complete. Breaking changes to the emitted metrics SHOULD be applied following [the deprecation 
process](#changing-the-emitted-metrics).

### Stable

The receiver is ready for production workloads. Breaking changes to the emitted metrics SHOULD be avoided.
Nevertheless, metrics that are emitted by default MUST be always kept up-to-date with the latest stable version of the 
monitored system. Given that, occasional breaking changes in the emitted metrics are expected even in the stable 
receivers. Any breaking change MUST be applied following [the deprecation process](#changing-the-emitted-metrics).

## Stability levels for metrics and attributes

See [Collector's Telemetry Stability levels](./coding-guidelines.md#telemetry-stability-levels)

## Changing the emitted metrics

Some changes are not considered breaking and can be applied to metrics emitted by scraping receivers of any 
stability level:

- Adding a new optional metric.

Most of other changes to the emitted metrics are considered breaking and MUST be handled according to the stability 
level of the receiver. Each type of breaking change defines a set of steps that MUST (or SHOULD) be applied across 
several releases for a Stable (or Beta) components. At least 3 versions SHOULD be kept between the steps to give 
users time to prepare, e.g. if the first step is released in v0.62.0, the second step SHOULD be released not earlier
than 0.65.0. Any warnings SHOULD include the version starting from which the next step will take effect. If a 
breaking change is more complicated and many metrics are involved in the change, feature gates SHOULD be used instead.

### Removing an optional metric

Steps to remove an optional metric:

1. Mark the metric as deprecated in `metadata.yaml` by adding "[DEPRECATED]" in its description. Show a warning that 
   the metric will be removed if the `enabled` option is set explicitly to `true` in user settings. 
2. Remove the metric.

### Removing a default metric

Steps to remove a default metric:

1. Mark the metric as deprecated in `metadata.yaml` by adding "[DEPRECATED]" in its description. Show a warning that
   the metric will be removed if the `enabled` option is not explicitly set to `false` in user settings.
2. Make the metric optional. Show a warning that the metric will be removed if the `enabled` option is set to `true`
   in user settings.
3. Remove the metric.

### Making a default metric optional

Steps to turn a metric from default to optional:

1. Add a warning that the metric will be turned into optional if `enabled` field is not set explicitly to any value in
   user settings. Warning example: "WARNING: Metric `foo.bar` will be disabled by default in v0.65.0. If you want to
   keep it, please enable it explicitly in the receiver settings."
2. Remove the warning and update `metadata.yaml` to make the metric optional.

### Adding a new default metric or turning an existing optional metric into default

Adding a new default metric is a breaking change for a scraping receiver because it introduces an unexpected output
for users and additional load on metric backends. Steps to apply such a change:

1. If the metric doesn't exist yet, add one as an optional metric. Add a warning that the metric will be turned into 
   default if the `enabled` option is not set explicitly to any value in user settings. A warning example: "WARNING: 
   Metric `foo.bar` will be enabled by default in v0.65.0. If you don't want the metric to be emitted, please
   disable it in the receiver settings."
2. Remove the warning and update `metadata.yaml` to make the metric default.

### Other changes

Other breaking changes SHOULD follow similar strategies inspecting presence of `enabled` field in user settings. For
example, if a metric has to be renamed for any reason, the guidelines for "Removing an optional metric" and "Adding a  
new default metric" SHOULD be followed simultaneously.

Breaking changes that cannot be done through enabling/disabling metrics (e.g. removing or adding an extra attribute)
SHOULD be applied using a feature gate with the following steps:

1. Add a feature gate that is disabled by default. Enabling the feature gate changes the metrics behavior in a desired 
   way. For example, if several metrics emitted by host metrics receiver need to be updated to have an additional 
   `direction` attribute, the following feature gate can be used:
  `receiver.hostmetricsreceiver.emitMetricsWithDirectionAttribute`. Show user a warning if the feature gate is not 
   enabled explicitly, for example: "[WARNING] Metrics `system.network.packets` and `system.network.errors` will be 
   changed in v0.65.0 to emit an additional `direction` attribute, enable a feature gate
   `receiver.hostmetricsreceiver.emitMetricsWithDirectionAttribute` to apply and test the upcoming changes earlier".
2. Enable the feature gate by default and update the warning thrown if user disables the feature gate explicitly. 
3. Remove the feature gate along with the old behavior.
