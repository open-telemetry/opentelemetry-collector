# Prometheus Receiver

Receives metric data in [Prometheus](https://prometheus.io/) format. See the
[Design](DESIGN.md) for additional information on this receiver.

Supported pipeline types: metrics

## ⚠️ Warning

Note: This component is currently work in progress. It has several limitations
and please don't use it if the following limitations is a concern:

* Collector cannot auto-scale the scraping yet when multiple replicas of the
  collector is run. 
* When running multiple replicas of the collector with the same config, it will
  scrape the targets multiple times.
* Users need to configure each replica with different scraping configuration
  if they want to manually shard the scraping.
* The Prometheus receiver is a stateful component.

## Unsupported features
The Prometheus receiver is meant to minimally be a drop-in replacement for Prometheus. However,
there are advanced features of Prometheus that we don't support and thus explicitly will return
an error for if the receiver's configuration YAML/code contains any of the following

- [x] alert_config.alertmanagers
- [x] alert_config.relabel_configs
- [x] remote_read
- [x] remote_write
- [x] rule_files


## Getting Started

This receiver is a drop-in replacement for getting Prometheus to scrape your
services. It supports [the full set of Prometheus configuration in `scrape_config`][sc],
including service discovery. Just like you would write in a YAML configuration
file before starting Prometheus, such as with:

**Note**: Since the collector configuration supports env variable substitution
`$` characters in your prometheus configuration are interpreted as environment
variables.  If you want to use $ characters in your prometheus configuration,
you must escape them using `$$`.

```shell
prometheus --config.file=prom.yaml
```

You can copy and paste that same configuration under:

```yaml
receivers:
  prometheus:
    config:
```

For example:

```yaml
receivers:
    prometheus:
      config:
        scrape_configs:
          - job_name: 'otel-collector'
            scrape_interval: 5s
            static_configs:
              - targets: ['0.0.0.0:8888']
          - job_name: k8s
            kubernetes_sd_configs:
            - role: pod
            relabel_configs:
            - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
              regex: "true"
              action: keep
            metric_relabel_configs:
            - source_labels: [__name__]
              regex: "(request_duration_seconds.*|response_duration_seconds.*)"
              action: keep
```

[sc]: https://github.com/prometheus/prometheus/blob/v2.28.1/docs/configuration/configuration.md#scrape_config
