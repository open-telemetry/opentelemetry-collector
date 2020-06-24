# Prometheus Receiver

Receives metric data in [Prometheus](https://prometheus.io/) format. See the
[Design](DESIGN.md) for additional information on this receiver.

This receiver is a drop-in replacement for getting Prometheus to scrape your
services. Just like you would write in a YAML configuration file before
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
          - job_name: 'opencensus_service'
            scrape_interval: 5s
            static_configs:
              - targets: ['0.0.0.0:8889']
          - job_name: 'jdbc_apps'
            scrape_interval: 3s
            static_configs:
              - targets: ['0.0.0.0:9777']
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
