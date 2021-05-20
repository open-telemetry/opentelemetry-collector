# Authenticator - OAuth2 Client Credentials

This extension implements a `configauth.Authenticator`, to be used based exporter (HTTP and gRPC based)
inside the `auth` settings. The authenticator type has to be set to `oauth2`.

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
    auth:
      authenticator: oauth2
      
  otlp/withauth:
    endpoint: 0.0.0.0:5000
    ca_file: /tmp/certs/ca.pem
    auth:
      authenticator: oauth2

service:
  extensions: [oauth2]
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: []
      exporters: [otlphttp/withauth, otlp/withauth]
```