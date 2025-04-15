# OpenTelemetry Collector Middleware Extension API

This package implements interfaces for injecting middleware behavior
in OpenTelemetry Collector exporters and receivers.  See the
[associated `configmiddleware` package](../../config/configmiddleware/README.md)
for referring to middleware
extensions in component configurations.

## Overview

Middleware extensions can be configured on gRPC and HTTP connections,
on both the client and server side.  The term "middleware" is defined
broadly to cover many ways of intercepting, acting on, and observing
requests as they enter and exit and RPC system.

Middleware details and capabilities are specific to each protocol.  In
some cases, these interfaces permit configuring behavior other than
middleware.  Users have to place a trust in the extensions they
configure, since they are capable of subverting security and other RPC
configuration.

Middleware is generally configured at a level in the code where:

1. the identity of the calling component is not known, because
   `confighttp` and `configgrpc` interfaces likewise are not configured
   with the identify of the calling component.
2. the signal type in use is not known, because a single connection
   serves multiple signals.

## Interfaces

Each interface has a single function to configure middleware for a
protocol on the client or server side.  An error is returned if the
extension cannot be configured.

New protocols and new ways to configure middleware can be introduced
by adding new interfaces.  Note that for each interface, there is a
corresponding method to locate a named middleware extension that
satisfies the interface in
[the `configmiddleware` package](../../config/configmiddleware/README.md) .

### HTTP

Interface methods are called once per request to construct a client-
or server-side middleware object.

- **HTTPClient**: The extension returns a function to create new `http.RoundTripper`s.
- **HTTPServer**: The extension returns a function to create new `http.Handler`s.

### GRPC

Interface methods are called once at setup to configure the client- or
server-side middleware object.

- **GRPCClient**: The extension returns `[]grpc.DialOption`.
- **GRPCServer**: The extension returns `[]grpc.ServerOption`.
