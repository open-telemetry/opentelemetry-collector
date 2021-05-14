# Authenticator - Bearer

This extension implements `configauth.GRPCClientAuthenticator` and is to be used in grpc receivers inside the `auth` settings as a means
to embed a static token for every rpc call that will be made.

The authenticator type has to be set to `bearertokenauth`.

## Configuration

```yaml
extensions:
   bearertokenauth:
    token: "sometoken"

receivers:
  hostmetrics:

exporters:
  otlp:
    endpoint: localhost:1234
    auth:
      authenticator: bearer

processors:

service:
  extensions: [bearertokenauth]
  pipelines:
    traces:
      receivers: [hostmetrics]
      processors: []
      exporters: [otlp]
```

The following is the only setting and is required:

- `token`: static authorization token that needs to be sent on every gRPC client call as metadata.
   This token is prepended by "Bearer " before being sent as a value of "authorization" key in
   rpc metadata.
  
