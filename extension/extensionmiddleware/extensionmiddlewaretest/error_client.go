// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest // import "go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"

import (
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
	"google.golang.org/grpc"
)

var (
	_ extension.Extension            = (*errorClient)(nil)
	_ extensionmiddleware.HTTPClient = (*errorClient)(nil)
	_ extensionmiddleware.GRPCClient = (*errorClient)(nil)
)

type errorClient struct {
	component.StartFunc
	component.ShutdownFunc
	extensionmiddleware.GetHTTPRoundTripperFunc
	extensionmiddleware.GetGRPCClientOptionsFunc
}

// NewErrorClient returns a new [extension.Extension] that implements the [extensionmiddleware.HTTPClient] and [extensionmiddleware.GRPCClient] and always returns an error on both methods.
func NewErrorClient(err error) extension.Extension {
	return &errorClient{
		GetHTTPRoundTripperFunc: func(http.RoundTripper) (http.RoundTripper, error) {
			return nil, err
		},
		GetGRPCClientOptionsFunc: func() ([]grpc.DialOption, error) {
			return nil, err
		},
	}
}
