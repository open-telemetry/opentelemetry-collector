// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"context"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var (
	_ extension.Extension      = (*nopClient)(nil)
	_ extensionauth.HTTPClient = (*nopClient)(nil)
	_ extensionauth.GRPCClient = (*nopClient)(nil)
)

type nopClient struct{}

// PerRPCCredentials implements [extensionauth.GRPCClient].
func (n *nopClient) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	return nil, nil
}

// RoundTripper implements [extensionauth.HTTPClient].
func (n *nopClient) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	return base, nil
}

// Shutdown implements [extension.Extension].
func (n *nopClient) Shutdown(context.Context) error {
	return nil
}

// Start implements [extension.Extension].
func (n *nopClient) Start(context.Context, component.Host) error {
	return nil
}

// NewNopClient returns a new [extension.Extension] that implements the [extensionauth.HTTPClient] and [extensionauth.GRPCClient].
// For HTTP requests it returns the base RoundTripper and for gRPC requests it returns a nil [credentials.PerRPCCredentials].
func NewNopClient() extension.Extension {
	return &nopClient{}
}
