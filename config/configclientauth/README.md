# AuthX Settings for Exporters

This module provides HTTP/GRPC client types to be configured to perform authorization/authentication for requests using the following.

## OAuth2 Client Credentials. (HTTP/gRPC)

OAuth2 Client Credentials library exposes a [variety of settings](https://pkg.go.dev/golang.org/x/oauth2/clientcredentials)
and implements the OAuth2.0 ["client credentials"](https://tools.ietf.org/html/rfc6749#section-1.3.4)
token flow, also known as the ["two-legged OAuth 2.0" or machine to machine authorization](https://tools.ietf.org/html/rfc6749#section-4.4).

This should be used when the client (exporter) is acting on its own behalf.


When enabled as a part of an HTTP/gRPC exporter configuration, the corresponding Client with OAuth2 supported authorization transport is returned.
The client manages the token auto refresh

### OAuth2 Client Credentials Configuration

> Please review the Collector's [security
> documentation](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/security.md),
> which contains recommendations on securing sensitive information such as the
> API key required by this configuration.

- [`client_id`](https://tools.ietf.org/html/rfc6749#section-2.2): The unique identifier issued by authorization server to the registered client.
- `client_secret`: Registered client's secret.
- [`token_url`](https://tools.ietf.org/html/rfc6749#section-3.2): endpoint which is used by the client to obtain an access token by
  presenting its authorization grant or refresh token
- `scopes`: optional list of requested resource permissions


Example:
```yaml
exporters:
  otlphttp:
    endpoint: http://localhost:9000
    oauth2:
      client_id: someclientidentifier
      client_secret: someclientsecret
      token_url: https://autz.server.com/oauth2/default/v1/token
      scopes: ["some.resource.read"]
```

## Bearer Token (gRPC)

This is to be used when a Bearer token needs to be attached every RPC directly. No token refresh will be 
done in this case.

> Please review the Collector's [security
> documentation](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/security.md),
> which contains recommendations on securing sensitive information such as the
> API key required by this configuration.

### Bearer Token settings 
```yaml
exporters:
  otlp:
    endpoint: some-grpc:9090
    per_rpc_auth:
      type: bearer
      bearer_token: somelonglivedbearertoken
```
