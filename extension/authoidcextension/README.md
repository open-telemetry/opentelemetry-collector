# OIDC

This extension enables bearer token authorization for GRPC receivers.

The following settings are required:

- `issuer_url`: The base URL for the OIDC provider.
- `audience`: Audience of the token, used during the verification

Example:

```yaml
receivers:
  somereceiver:
    grpc:
      auth:
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
      receivers: [somereceiver]
```

The full list of settings exposed for this exporter is documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).