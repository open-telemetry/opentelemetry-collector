# Host Metrics Receiver

The Host Metrics receiver generates metrics about the host system. This is intended to be used when the collector is
deployed as an agent.

The categories of metrics scraped can be configured under the `scrapers` key, e.g. to scrape cpu, memory and disk metrics
(note not all of these are implemented yet):

```
hostmetrics:
  default_collection_interval: 10s
  scrapers:
    cpu:
      report_per_process: true
    memory:
    disk:
```

## OS Support

The initial implementation of the Host Metrics receiver supports Windows only. It's intended that future iterations
of this receiver will support the collection of identical metrics (where possible) on other operating systems.
