# Authenticator - OIDC

This extension implements a `configauth.Authenticator`, to be used in exporter inside the `auth` settings. The authenticator type has to be set to `oauth2`.

## Configuration

```yaml
extensions:
  oauth2:
    client_id: someclientid
    client_secret: someclientsecret
    token_url: https://someserver.com/oauth2/default/v1/token
    scopes: ["api.metrics"]

receivers:
  hostmetrics:
    scrapers:
      memory:
  otlp:
    protocols:
      grpc:

exporters:
  otlphttp/withauth:
    endpoint: http://localhost:9000
    authentication:
      authenticator: oauth2

service:
  extensions: [oauth2]
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: []
      exporters: [otlphttp/withauth]
```