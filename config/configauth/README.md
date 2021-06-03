# Authentication configuration for receivers

This module allows server types, such as gRPC and HTTP, to be configured to perform authentication for requests and/or RPCs. Each server type is responsible for getting the request/RPC metadata and passing down to the authenticator.

The currently known authenticators:

- [oidc](../../extension/oidcauthextension)

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

receivers:
  otlp/with_auth:
    protocols:
      grpc:
        endpoint: localhost:4318
        tls_settings:
          cert_file: /tmp/certs/cert.pem
          key_file: /tmp/certs/cert-key.pem
        auth:
          ## oidc is the extension name to use as the authenticator for this receiver
          authenticator: oidc
```

## Creating an authenticator

New authenticators can be added by creating a new extension that also implements the `configauth.ServerAuthenticator` extension. Generic authenticators that may be used by a good number of users might be accepted as part of the core distribution, or as part of the contrib distribution. If you have interest in contributing one authenticator, open an issue with your proposal.

For other cases, you'll need to include your custom authenticator as part of your custom OpenTelemetry Collector, perhaps being built using the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector-builder).
