// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionmiddlewaretest // import "go.opentelemetry.io/collector/extension/extensionmiddleware/extensionmiddlewaretest"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionmiddleware"
)

var (
	_ extension.Extension            = (*nopClient)(nil)
	_ extensionmiddleware.HTTPClient = (*nopClient)(nil)
	_ extensionmiddleware.GRPCClient = (*nopClient)(nil)
)

type nopClient struct {
	component.StartFunc
	component.ShutdownFunc
	extensionmiddleware.GetHTTPRoundTripperFunc
	extensionmiddleware.GetGRPCClientOptionsFunc
}

// NewNopClient returns a new [extension.Extension] that implements
// the [extensionmiddleware.HTTPClient] and
// [extensionmiddleware.GRPCClient].  For HTTP requests it returns the
// base RoundTripper and for gRPC requests it returns an empty slice.
func NewNopClient() extension.Extension {
	return &nopClient{}
}
