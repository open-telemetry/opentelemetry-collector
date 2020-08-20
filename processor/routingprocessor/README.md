# Routing processor

Routes traces to specific exporters.

This processor will read a header from the incoming HTTP request (gRPC or plain HTTP) and direct the trace information to specific exporters based on the attribute's value.

This processor *does not* let traces to continue through the pipeline and will emit a warning in case other processor(s) are defined after this one. Similarly, exporters defined as part of the pipeline are not authoritative: if you add an exporter to the pipeline, make sure you add it to this processor *as well*, otherwise it won't be used at all. All exporters defined as part of this processor *must also* be defined as part of the pipeline's exporters.

The following settings are required:

- `from_attribute`: contains the HTTP header name to look up the route's value. Only the OTLP exporter has been tested in connection with the OTLP gRPC Receiver, but any other gRPC receiver should work fine, as long as the client sends the specified HTTP header.
- `table`: the routing table for this processor.
- `table.value`: a possible value for the attribute specified under FromAttribute.
- `table.exporters`: the list of exporters to use when the value from the FromAttribute field matches this table item.

The following settings can be optionally configured:

- `default_exporters` contains the list of exporters to use when a more specific record can't be found in the routing table.

Example:

```yaml
processors:
  routing:
    from_attribute: X-Tenant
    default_exporters: jaeger
    table:
    - value: acme
      exporters: [jaeger/acme]
exporters:
  jaeger:
    endpoint: localhost:14250
  jaeger/acme:
    endpoint: localhost:24250
```

The full list of settings exposed for this processor are documented [here](./config.go) with detailed sample configuration [here](./testdata/config.yaml).
