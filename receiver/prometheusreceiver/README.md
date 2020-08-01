# Prometheus Receiver

Receives metric data in [Prometheus](https://prometheus.io/) format. See the
[Design](DESIGN.md) for additional information on this receiver.

This receiver is a drop-in replacement for getting Prometheus to scrape your
services. It supports the full set of Prometheus configuration, including 
service discovery. Just like you would write in a YAML configuration file before
starting Prometheus, such as with:
```shell
prometheus --config.file=prom.yaml
```

You can copy and paste that same configuration under section
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

### Include Filter
Include Filter provides ability to filter scraping metrics per target. If a
filter is specified for a target then only those metrics which exactly matches
one of the metrics specified in the `Include Filter` list will be scraped. Rest
of the metrics from the targets will be dropped.

#### Syntax
* Endpoint should be double quoted.
* Metrics should be specified in form of a list.

#### Example
```yaml
receivers:
    prometheus:
      include_filter: {
        "0.0.0.0:9777" : [http/server/server_latency, custom_metric1],
        "0.0.0.0:9778" : [http/client/roundtrip_latency],
      }
      config:
        scrape_configs:
          ...
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
