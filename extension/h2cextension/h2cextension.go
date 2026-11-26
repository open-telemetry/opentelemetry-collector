// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package h2cextension provides an HTTP server middleware extension for serving
// cleartext HTTP/2 connections.
package h2cextension // import "go.opentelemetry.io/collector/extension/h2cextension"

import (
	"context"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	_ extensionmiddleware.HTTPServer          = (*h2cExtension)(nil)
	_ extensionmiddleware.HTTPServerProtocols = (*h2cExtension)(nil)
	_ component.Component                     = (*h2cExtension)(nil)
)

// h2cExtension asks the HTTP server to accept HTTP/2 with prior knowledge
// over cleartext connections. net/http serves each stream through the
// server's full handler chain and drains the connections when the server
// shuts down.
type h2cExtension struct {
	component.StartFunc
	component.ShutdownFunc
}

// GetHTTPHandler implements extensionmiddleware.HTTPServer. The handler is
// unchanged, because the server itself serves h2c.
func (*h2cExtension) GetHTTPHandler(context.Context) (extensionmiddleware.WrapHTTPHandlerFunc, error) {
	return func(_ context.Context, next http.Handler) (http.Handler, error) {
		return next, nil
	}, nil
}

// GetHTTPServerProtocols implements extensionmiddleware.HTTPServerProtocols.
func (*h2cExtension) GetHTTPServerProtocols(context.Context) (http.Protocols, error) {
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	return protocols, nil
}
