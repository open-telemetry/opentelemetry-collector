# Authentication configuration for receivers

This module allows server types, such as gRPC and HTTP, to be configured to perform authentication for requests and/or RPCs. Each server type is responsible for getting the request/RPC metadata and passing down to the authenticator.

Authenticators are set up as extensions and passed to the server settings for a receiver. When writing a custom authenticator extension, it must call `configauth.AddAuthenticatorToRegistry` from within its createExtension factory method. This allows the receiver to look up the instance of the authenticator that it is configured to use. See [the OIDC extension factory](../../extension/oidcextension/factory.go) as an example.

Config Examples:
```yaml
receivers:
  somereceiver:
    grpc:
      authenticator: oidc

extensions:
  oidc:
    issuer_url: https://auth.example.com/
    issuer_ca_path: /etc/pki/tls/cert.pem
    audience: my-oidc-client
    username_claim: email
    attribute: authorization

service:
  extensions: [oidc]
  pipeline:
    traces/toSomewhere:
      recievers: [somereceiver]
```
