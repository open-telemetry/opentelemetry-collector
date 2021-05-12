# Authenticator - Bearer

This extension implements `configauth.Authenticator`, to be used in grpc receivers inside the `auth` settings as a means
to embed a static token for every rpc call that will be made.

The authenticator type has to be set to `bearer`.

## Configuration

```yaml
extensions:
  bearer:
    bearer_token: "sometoken"

receivers:
  hostmetrics:

exporters:
  otlp:
    endpoint: localhost:1234
    authentication:
      authenticator: bearer

processors:

service:
  extensions: [bearer]
  pipelines:
    traces:
      receivers: [hostmetrics]
      processors: []
      exporters: [otlp]
```