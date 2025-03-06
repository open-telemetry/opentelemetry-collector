// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var errMockError = errors.New("mock Error")

var (
	_ extension.Extension      = (*errorClient)(nil)
	_ extensionauth.HTTPClient = (*errorClient)(nil)
	_ extensionauth.GRPCClient = (*errorClient)(nil)
)

type errorClient struct{}

// PerRPCCredentials implements extensionauth.GRPCClient.
func (e *errorClient) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	return nil, errMockError
}

// RoundTripper implements extensionauth.HTTPClient.
func (e *errorClient) RoundTripper(http.RoundTripper) (http.RoundTripper, error) {
	return nil, errMockError
}

// Shutdown implements extension.Extension.
func (e *errorClient) Shutdown(context.Context) error {
	return nil
}

// Start implements extension.Extension.
func (e *errorClient) Start(context.Context, component.Host) error {
	return nil
}

// NewErrorClient returns a new [extension.Extension] that implements the [extensionauth.HTTPClient] and [extensionauth.GRPCClient] and always returns an error on both methods.
func NewErrorClient() extension.Extension {
	return &errorClient{}
}
