# OIDC

This extension enables bearer token authorization for GRPC receivers.

The following settings are required:

- `issuer_url`: The base URL for the Config provider.
- `audience`: Audience of the token, used during the verification
- `attribute` (default: `authorization`): The attribute (header name) to look for auth data

Example:

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
    groups_claim: group
    attribute: authorization

service:
  extensions: [oidc]
  pipeline:
    traces/toSomewhere:
      recievers: [somereceiver]
```

The full list of settings exposed for this exporter is documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).