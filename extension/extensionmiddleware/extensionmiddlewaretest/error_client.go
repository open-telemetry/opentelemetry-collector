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
	_ extension.Extension            = (*errClient)(nil)
	_ extensionmiddleware.HTTPClient = (*errClient)(nil)
	_ extensionmiddleware.GRPCClient = (*errClient)(nil)
)

type errClient struct {
	component.StartFunc
	component.ShutdownFunc
	extensionmiddleware.GetHTTPRoundTripperFunc
	extensionmiddleware.GetGRPCClientOptionsFunc
}

// NewErr returns a new [extension.Extension] that implements the [extensionmiddleware.HTTPClient] and [extensionmiddleware.GRPCClient] and always returns an error on both methods.
func NewErr(err error) extension.Extension {
	return &errClient{
		GetHTTPRoundTripperFunc: func(http.RoundTripper) (http.RoundTripper, error) {
			return nil, err
		},
		GetGRPCClientOptionsFunc: func() ([]grpc.DialOption, error) {
			return nil, err
		},
	}
}
