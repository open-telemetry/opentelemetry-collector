# Authentication configuration for receivers

This module allows server types, such as gRPC and HTTP, to be configured to perform authentication for requests and/or RPCs. Each server type is responsible for getting the request/RPC metadata and passing down to the authenticator. Currently, only bearer token authentication is supported, although the module is ready to accept new authenticators.

Examples:
```yaml
receivers:
  somereceiver:
    grpc:
      authentication:
        attribute: authorization
        oidc:
          issuer_url: https://auth.example.com/
          issuer_ca_path: /etc/pki/tls/cert.pem
          client_id: my-oidc-client
          username_claim: email
```
