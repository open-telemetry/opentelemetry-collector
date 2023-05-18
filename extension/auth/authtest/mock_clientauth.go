// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package authtest // import "go.opentelemetry.io/collector/extension/auth/authtest"

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/auth"
)

var (
	_            auth.Client = (*MockClient)(nil)
	errMockError             = errors.New("mock Error")
)

// MockClient provides a mock implementation of GRPCClient and HTTPClient interfaces
type MockClient struct {
	ResultRoundTripper      http.RoundTripper
	ResultPerRPCCredentials credentials.PerRPCCredentials
	MustError               bool
}

// Start for the MockClient does nothing
func (m *MockClient) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown for the MockClient does nothing
func (m *MockClient) Shutdown(_ context.Context) error {
	return nil
}

// RoundTripper for the MockClient either returns error if the mock authenticator is forced to or
// returns the supplied resultRoundTripper.
func (m *MockClient) RoundTripper(_ http.RoundTripper) (http.RoundTripper, error) {
	if m.MustError {
		return nil, errMockError
	}
	return m.ResultRoundTripper, nil
}

// PerRPCCredentials for the MockClient either returns error if the mock authenticator is forced to or
// returns the supplied resultPerRPCCredentials.
func (m *MockClient) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if m.MustError {
		return nil, errMockError
	}
	return m.ResultPerRPCCredentials, nil
}
