# OAuth2 Client Credential Settings

This module provides HTTP client types to be configured to perform authorization for requests using OAuth2 Client Credentials.

OAuth2 Client Credentials library exposes a [variety of settings](https://pkg.go.dev/golang.org/x/oauth2/clientcredentials)
and implements the OAuth2.0 ["client credentials"](https://tools.ietf.org/html/rfc6749#section-1.3.4)
token flow, also known as the "two-legged OAuth 2.0" or machine to machine authorisation. This should be used when the client is acting on its own behalf.

When enabled as a part of HTTP Client Configuration, an HTTP Client with OAuth2 supported authorization transport is returned. The client 
manages the token auto refresh

## OAuth2 Client Credentials Configuration

- [`client_id`](https://tools.ietf.org/html/rfc6749#section-2.2): The unique identifier issued by authorization server to the registered client.
- `client_secret`: Registered clients secret.
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
      token_url: https://autz.server/oauth2/default/v1/token
      scopes: ["some.resource.read"]
```