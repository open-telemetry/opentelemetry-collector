# Authentication configuration

This module defines necessary interfaces to implement 

- Server type authenticators for gRPC and HTTP receivers to perform authentication for requests and/or RPCs. Each server type is responsible for getting the request/RPC metadata and passing down to the authenticator.
- Client type authenticators such as gRPC and HTTP exporters to perform client side authentication. 

Few such authenticators that are implemented currently:

- Server Authenticators
  - [oidc](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/oidcauthextension)

- Client Authenticators
  - [oauth2](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/oauth2clientauthextension)
  - [BearerToken](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/bearertokenauthextension)

Examples:
```yaml
extensions:
  oidc:
    # see the blog post on securing the otelcol for information
    # on how to setup an OIDC server and how to generate the TLS certs
    # required for this example
    # https://medium.com/opentelemetry/securing-your-opentelemetry-collector-1a4f9fa5bd6f
    issuer_url: http://localhost:8080/auth/realms/opentelemetry
    audience: account

  oauth2client:
    client_id: someclientid
    client_secret: someclientsecret
    token_url: https://example.com/oauth2/default/v1/token
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
  otlp/with_auth:
    protocols:
      grpc:
        endpoint: localhost:4318
        tls:
          cert_file: /tmp/certs/cert.pem
          key_file: /tmp/certs/cert-key.pem
        auth:
          ## oidc is the extension name to use as the authenticator for this receiver
          authenticator: oidc

  otlphttp/withauth:
    endpoint: http://localhost:9000
    auth:
      authenticator: oauth2client

```

## Creating an authenticator

New server side authenticators can be added by creating a new extension that also implements the `configauth.ServerAuthenticator` extension and similarly a client side authenticator can be added by implementing `configauth.ClientAuthenticator` interface.

Generic authenticators that may be used by a good number of users might be accepted as part of the core distribution, or as part of the contrib distribution. If you have interest in contributing one authenticator, open an issue with your proposal. For other cases, you'll need to include your custom authenticator as part of your custom OpenTelemetry Collector, perhaps being built using the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector-builder).
