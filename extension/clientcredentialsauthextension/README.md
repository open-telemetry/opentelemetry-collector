# Authenticator - OAuth2 Client Credentials

This extension implements a `configauth.ClientAuthenticator` and is to be used by HTTP and gRPC based exporters.
The authenticator type has to be set to `oauth2`.

## Configuration

```yaml
extensions:
  oauth2clientcredentials:
    client_id: someclientid
    client_secret: someclientsecret
    token_url: https://someserver.com/oauth2/default/v1/token
    scopes: ["api.metrics"]
    # tls settings for the token client
    tls:
        insecure: true
        ca_file: /var/lib/mycert.pem
        cert_file: certfile
        key_file: keyfile
    # timeout for the token client
    timeout: 2s
    
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
      authenticator: oauth2clientcredentials
      
  otlp/withauth:
    endpoint: 0.0.0.0:5000
    ca_file: /tmp/certs/ca.pem
    auth:
      authenticator: oauth2clientcredentials

service:
  extensions: [oauth2clientcredentials]
  pipelines:
    metrics:
      receivers: [hostmetrics]
      processors: []
      exporters: [otlphttp/withauth, otlp/withauth]
```

For more information on client side tls settings, see [configtls README](../../config/configtls/README.md).