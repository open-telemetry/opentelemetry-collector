# Host Metrics Receiver

The Host Metrics receiver generates metrics about the host system. This is intended to be used when the collector is
deployed as an agent.

The categories of metrics scraped can be configured under the `scrapers` key, e.g. to scrape cpu, memory and disk metrics:

```yaml
hostmetrics:
  collection_interval: 1m
  scrapers:
    cpu:
    memory:
    disk:
```

If you would like to scrape some metrics at a different frequency than others, you can configure multiple hostmetrics
receivers with different collection_interval values, e.g.

```yaml
receivers:
  hostmetrics:
    collection_interval: 30s
    scrapers:
      cpu:
      memory:

  hostmetrics/disk:
    collection_interval: 1m
    scrapers:
      disk:
      filesystem:

service:
  pipelines:
    metrics:
      receivers: [hostmetrics, hostmetrics/disk]
```

## OS Support

The initial implementation of the Host Metrics receiver supports Windows only. It's intended that future iterations
of this receiver will support the collection of identical metrics (where possible) on other operating systems.
