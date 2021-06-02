# Authenticator - Bearer

This extension implements `configauth.GRPCClientAuthenticator` and is to be used in gRPC receivers inside the `auth` settings as a means
to embed a static token for every RPC call that will be made.

The authenticator type has to be set to `bearertokenauth`.

## Configuration

The following is the only setting and is required:

- `token`: static authorization token that needs to be sent on every gRPC client call as metadata.
  This token is prepended by "Bearer " before being sent as a value of "authorization" key in
  RPC metadata.
  
  **Note**: bearertokenauth requires transport layer security enabled on the exporter.


```yaml
extensions:
  bearertokenauth:
    token: "somerandomtoken"

receivers:
  hostmetrics:
    scrapers:
      memory:
  otlp:
    protocols:
      grpc:

exporters:
  otlp/withauth:
    endpoint: 0.0.0.0:5000
    ca_file: /tmp/certs/ca.pem
    auth:
      authenticator: bearertokenauth

service:
  extensions: [bearertokenauth]
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: []
      exporters: [otlp/withauth]
```
