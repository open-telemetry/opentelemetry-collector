// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest // import "go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"

import (
	"net/http"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	_ extension.Extension            = (*baseExtension)(nil)
	_ extensionmiddleware.HTTPClient = (*baseExtension)(nil)
	_ extensionmiddleware.GRPCClient = (*baseExtension)(nil)
	_ extensionmiddleware.HTTPServer = (*baseExtension)(nil)
	_ extensionmiddleware.GRPCServer = (*baseExtension)(nil)
)

type baseExtension struct {
	component.StartFunc
	component.ShutdownFunc
	extensionmiddleware.GetHTTPHandlerFunc
	extensionmiddleware.GetGRPCServerOptionsFunc
	extensionmiddleware.GetHTTPRoundTripperFunc
	extensionmiddleware.GetGRPCClientOptionsFunc
}

// NewErr returns a new [extension.Extension] that implements all
// extensionmiddleware interface and always returns an error.
func NewErr(err error) extension.Extension {
	return &baseExtension{
		GetHTTPRoundTripperFunc: func(http.RoundTripper) (http.RoundTripper, error) {
			return nil, err
		},
		GetGRPCClientOptionsFunc: func() ([]grpc.DialOption, error) {
			return nil, err
		},
		GetHTTPHandlerFunc: func(http.Handler) (http.Handler, error) {
			return nil, err
		},
		GetGRPCServerOptionsFunc: func() ([]grpc.ServerOption, error) {
			return nil, err
		},
	}
}
