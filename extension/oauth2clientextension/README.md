# Authenticator - OIDC

This extension implements a `configauth.Authenticator`, to be used in exporter inside the `auth` settings. The authenticator type has to be set to `oauth2`.

## Configuration

```yaml
extensions:
  oauth2:
    client_id: someclientidentifier
    client_secret: someclientsecret
    token_url: https://autz.server/oauth2/default/v1/token
    scopes: ["some.resource.read"]

receivers:
  otlp:
    protocols:
      grpc:

processors:

exporter:
  otlphttp/withauth:
    authentication:
      authenticator: oauth2

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [otlphttp/withauth]
```