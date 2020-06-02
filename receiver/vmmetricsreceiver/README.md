# VM Metrics Receiver

Collects metrics from the host operating system. This is applicable when the
OpenTelemetry Collector is running as an agent.

```yaml
receivers:
  vmmetrics:
    scrape_interval: 10s
    metric_prefix: "testmetric"
    mount_point: "/proc"
    #process_mount_point: "/data/proc" # Only using when running as an agent / daemonset
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
