# Resource Processor

Supported pipeline types: metrics, traces

The resource processor can be used to override a resource.
Please refer to [config.go](./config.go) for the config spec.

The following configuration options are required:
- `type`: Resource type to be applied. If specified, this value overrides the
original resource type. Otherwise, the original resource type is kept.
- `labels`: Map of key/value pairs that should be added to the resource.

Examples:

```yaml
processors:
  resource:
    type: "host"
    labels: {
      "cloud.zone": "zone-1",
      "k8s.cluster.name": "k8s-cluster",
      "host.name": "k8s-node",
    }
```

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
