# Zipkin Receiver

This receiver receives spans from Zipkin (V1 and V2).

To get started, all that is required to enable the Zipkin receiver is to
include it in the receiver definitions. This will enable the default values as
specified [here](./factory.go).
The following is an example:

```yaml
receivers:
  zipkin:
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
