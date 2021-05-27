# Authenticator - OIDC

This extension implements a `configauth.Authenticator`, to be used in receivers inside the `auth` settings. The authenticator type has to be set to `oidc`.

## Configuration

```yaml
extensions:
  oidc:
    issuer_url: https://tenant1.example.com/
    issuer_ca_path: /etc/pki/tls/cert.pem
    client_id: my-oidc-client
    username_claim: email

receivers:
  otlp:
    protocols:
      grpc:
        authentication:
          attribute: authorization
          authenticator: oidc

processors:

exporters:
  logging:
    logLevel: debug

service:
  extensions: [oidc]
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [logging]
```