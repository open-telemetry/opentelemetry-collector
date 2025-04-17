# OpenTelemetry Collector Middleware Configuration

This package implements a configuration struct for referring to
[middleware extensions](../../extension/extensionmiddleware/README.md).

## Overview

The `configmiddleware` package defines a `Config` type that
allows components to configure middleware extensions, typically as
an ordered list.
This support is built in for push-based receivers configured through
`confighttp` and `configgrpc`, as for example in the OTLP receiver:

```yaml
receivers:
  otlp:
    protocols:
	  http:
	    middlewares:
		- id: limitermiddleware
```

## Methods

The package provides four key methods to retrieve appropriate middleware handlers:

1. **GetHTTPClientRoundTripper**: Obtains a function to wrap an HTTP client with a middleware extension via a `http.RoundTripper`.

2. **GetHTTPServerHandler**: Obtains a function to wrap an HTTP server with a middleware extension via a `http.Handler`.

3. **GetGRPCClientOptions**: Obtains a `[]grpc.DialOption` that configure a middleware extension for gRPC clients.

4. **GetGRPCServerOptions**: Obtains a `[]grpc.ServerOption` that configure a middleware extension for gRPC servers.

These functions are typically called during Start() by a component,
passing the `component.Host` extensions.
An error is returned if the named extension cannot be found.
