# Wavefront Agent Exporter

- [Format](#format)
- [Proxy Example](#proxy-example)
- [Direct Ingestion Example](#direct-ingestion-example)

### Configuration
Wavefront exporter supports sending data using Direct Ingestion (`direct_ingestion`) or via Wavefront Proxy (`proxy`).

In the Service's YAML configuration file, under section "exporters" and sub-section "wavefront", please configure these fields. 

### Format
```yaml
exporters:
  wavefront:
    enable_traces: true|false

    # One of "proxy" or "direct_ingestion" is required
    proxy:
      Host: "<Proxy_IP_or_FQDN>"
      MetricsPort: <wf_metrics_port>
      TracingPort: <wf_trace_port>
      DistributionPort: <wf_distribution_port>
    direct_ingestion:
      Server: "<wavefront_url>"
      Token: "<wavefront_token>"

    # The following are optional
    override_source: "<source_name>"
    application_name: "<app_name>"
    service_name: "<service_name>"
    custom_tags:
      - "<key1>" : "<val1>"
      - "<key2>" : "<val2>"
      # ...
    max_queue_size: number
    verbose_logging: true|false
```

### Proxy Example
```yaml
exporters:
  wavefront:
    proxy:
      Host: "localhost"
      TracingPort: 30000
    enable_traces: true
```

### Direct Ingestion Example
```yaml
exporters:
  wavefront:
    direct_ingestion:
      Server: "https://<MYINSTANCE>.wavefront.com"
      Token: "MY_WAVEFRONT_TOKEN_HERE"
    enable_traces: true
```

### References
- [https://github.com/wavefrontHQ/opencensus-exporter](https://github.com/wavefrontHQ/opencensus-exporter) 
- [https://github.com/wavefrontHQ/wavefront-sdk-go](https://github.com/wavefrontHQ/wavefront-sdk-go)