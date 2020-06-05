# Probabilistic Sampling Processor

Supported pipeline types: traces

The probabilistic sampler supports two types of sampling:

1. `sampling.priority` [semantic
convention](https://github.com/opentracing/specification/blob/master/semantic_conventions.md#span-tags-table)
as defined by OpenTracing
2. Trace ID hashing

The `sampling.priority` semantic convention takes priority over trace ID hashing. As the name
implies, trace ID hashing samples based on hash values determined by trace IDs. In order for
trace ID hashing to work, all collectors for a given tier (e.g. behind the same load balancer)
must have the same `hash_seed`. It is also possible to leverage a different `hash_seed` at
different collector tiers to support additional sampling requirements. Please refer to
[config.go](./config.go) for the config spec.

The following configuration options can be modified:
- `hash_seed` (no default): An integer used to compute the hash algorithm. Note that all collectors for a given tier (e.g. behind the same load balancer) should have the same hash_seed.
- `sampling_percentage` (default = 0): Percentage at which traces are sampled; >= 100 samples all traces

Examples:

```yaml
processors:
  probabilistic_sampler:
    hash_seed: 22
    sampling_percentage: 15.3
```

Refer to [config.yaml](./testdata/config.yaml) for detailed
examples on using the processor.
