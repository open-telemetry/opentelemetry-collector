// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest // import "go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	_ extension.Extension            = (*nopServer)(nil)
	_ extensionmiddleware.HTTPServer = (*nopServer)(nil)
	_ extensionmiddleware.GRPCServer = (*nopServer)(nil)
)

type nopServer struct {
	component.StartFunc
	component.ShutdownFunc
	extensionmiddleware.GetHTTPHandlerFunc
	extensionmiddleware.GetGRPCServerOptionsFunc
}

// NewNopServer returns a new extension.Extension that implements the
// extensionmiddleware.HTTPServer and extensionmiddleware.GRPCServer.
func NewNopServer() extension.Extension {
	return &nopServer{}
}
