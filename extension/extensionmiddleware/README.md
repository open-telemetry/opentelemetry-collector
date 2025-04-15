# OpenTelemetry Collector Middleware Extension API

This package implements interfaces for injecting middleware behavior
in OpenTelemetry Collector exporters and receivers.  See [the
associated `configmiddleware` package](../../config/configmiddleware/README.md) 
for referring to middleware extensions in component configurations.

## Overview

Middleware extensions can be configured on gRPC and HTTP connections,
on both the client and server side.

## Interfaces

1. **HTTPClient**: The extension returns a function to create new `http.RoundTripper`s.

2. **HTTPServer**: The extension returns a function to create new `http.Handler`s.

3. **GRPCClient**: The extension returns `[]grpc.DialOption`.

4. **GRPCServer**: The extension returns `[]grpc.ServerOption`.

Each interface has a single function, which is typically called when a
component configures its network connection.  An error is returned if
the named extension cannot be configured.

Note that this module is tested through `configmiddleware`.
