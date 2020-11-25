# Authentication configuration for receivers

This module allows server types, such as gRPC and HTTP, to be configured to perform authentication for requests and/or RPCs. Each server type is responsible for getting the request/RPC metadata and passing down to the authenticator.

Examples:
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
