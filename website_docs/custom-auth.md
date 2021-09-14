---
title: "Building a custom authenticator"
weight: 30
---

The OpenTelemetry Collector allows receivers and exporters to be connected to authenticators, providing a way to both authenticate incoming connections at the receiver's side, as well as adding authentication data to outgoing requests at the exporter's side.

This mechanism is implemented on top of the [`extensions`](https://pkg.go.dev/go.opentelemetry.io/collector/component#Extension) framework and this document will guide you on implementing your own authenticators. If you are looking for documentation on how to use an existing authenticator, refer to the Getting Started page and to your authenticator's documentation. You can find a list of existing authenticators in this website's registry.

Use this guide for general directions on how to build a custom authenticator and refer to the up-to-date [API Reference Guide](https://pkg.go.dev/go.opentelemetry.io/collector/config/configauth) for the actual semantics of each type and function.

If at anytime you need assistance, join the [#opentelemetry-collector](https://cloud-native.slack.com/archives/C01N6P7KR6W) room at the [CNCF Slack workspace](https://slack.cncf.io).

## Architecture

Authenticators are regular extensions that also satisfy one or more interfaces related to the authentication mechanism:

- [go.opentelemetry.io/collector/config/configauth/ServerAuthenticator](https://pkg.go.dev/go.opentelemetry.io/collector/config/configauth#ServerAuthenticator)
- [go.opentelemetry.io/collector/config/configauth/GRPCClientAuthenticator](https://pkg.go.dev/go.opentelemetry.io/collector/config/configauth#GRPCClientAuthenticator)
- [go.opentelemetry.io/collector/config/configauth/HTTPClientAuthenticator](https://pkg.go.dev/go.opentelemetry.io/collector/config/configauth#HTTPClientAuthenticator)

Server authenticators are used with receivers, and are able to intercept HTTP and gRPC requests, while client authenticators are used with exporters, able to add authentication data to HTTP and gRPC requests. It is possible for authenticators to implement both interfaces at the same time, allowing a single instance of the extension to be used both for the incoming and outgoing requests. Note that users might still want to have different authenticators for the incoming and outgoing requests, so, don't make your authenticator required to be used at both ends.

Once an authenticator extension is available in the collector distribution, it can be referenced in the configuration file as a regular extension:

```yaml
extensions:
  oidc:
receivers:
processors:
exporters:
service:
  extensions:
    - oidc
  pipelines:
    traces:
      receivers:  []
      processors: []
      exporters:  []
```

However, an authenticator will need to be referenced by a consuming component to be effective. The following example shows the same extension as above, now being used by a receiver named `otlp/auth`:

```yaml
extensions:
  oidc:
receivers:
  otlp/auth:
    protocols:
      grpc:
        auth:
          authenticator: oidc
processors:
exporters:
service:
  extensions:
    - oidc
  pipelines:
    traces:
      receivers:
        - otlp/auth
      processors: []
      exporters:  []
```

When multiple instances of a given authenticator are needed, they can have different names:

```yaml
extensions:
  oidc/some-provider:
  oidc/another-provider:
receivers:
  otlp/auth:
    protocols:
      grpc:
        auth:
          authenticator: oidc/some-provider
processors:
exporters:
service:
  extensions:
    - oidc/some-provider
    - oidc/another-provider
  pipelines:
    traces:
      receivers:
        - otlp/auth
      processors: []
      exporters:  []
```

### Server authenticators

A server authenticator is essentially an extension with an `Authenticate` function, receiving the payload headers as parameter. If the authenticator is able to authenticate the incoming connection, it should return a `nil` error, or the concrete error if it can't. As an extension, the authenticator should make sure to initialize all the resources it needs during the [`Start`](https://pkg.go.dev/go.opentelemetry.io/collector/component#Component) phase, and is expected to clean them up upon `Shutdown`.

The `Authenticate` call is part of the hot path for incoming requests and will block the pipeline, so make sure to properly handle any blocking operations you need to make. Concretely, respect the deadline set by the context, in case one is provided. Also make sure to add enough observability to your extension, especially in the form of metrics and traces, so that users can get setup a notification system in case error rates go up beyond a certain level and can debug specific failures.

### Client authenticators

A client authenticator is one that implements one or more of the following interfaces:

- [go.opentelemetry.io/collector/config/configauth/GRPCClientAuthenticator](https://pkg.go.dev/go.opentelemetry.io/collector/config/configauth#GRPCClientAuthenticator)
- [go.opentelemetry.io/collector/config/configauth/HTTPClientAuthenticator](https://pkg.go.dev/go.opentelemetry.io/collector/config/configauth#HTTPClientAuthenticator)

Similar to server authenticators, they are essentially extensions with extra functions, each receiving an object that gives the authenticator an opportunity to inject the authentication data into. For instance, the HTTP client authenticator provides an [`http.RoundTripper`](https://pkg.go.dev/net/http#RoundTripper), while the gRPC client authenticator can produce a [`credentials.PerRPCCredentials`](https://pkg.go.dev/google.golang.org/grpc/credentials#PerRPCCredentials).

## Adding your custom authenticator to a distribution

Custom authenticators have to be part of the same binary as the main collector. When building your own authenticator, you'll likely have to build a custom distribution as well, or provide means for your users to consume your extension as part of their own distributions. Fortunately, building a custom distribution can be done using the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector-builder) utility.
