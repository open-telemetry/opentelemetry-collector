// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
)

func TestAuthenticatorManager(t *testing.T) {
	manager := NewAuthenticatorManager()

	// Test no authenticator set
	rt, err := manager.GetHTTPRoundTripper(http.DefaultTransport)
	require.NoError(t, err)
	assert.Equal(t, http.DefaultTransport, rt)

	creds, err := manager.GetGRPCCredentials()
	require.NoError(t, err)
	assert.Nil(t, creds)

	// Test with authenticator set
	mockAuth := &mockAuthenticatorProvider{}
	manager.SetAuthenticator(mockAuth)

	rt, err = manager.GetHTTPRoundTripper(http.DefaultTransport)
	require.NoError(t, err)
	assert.Equal(t, mockAuth.roundTripper, rt)

	creds, err = manager.GetGRPCCredentials()
	require.NoError(t, err)
	assert.Equal(t, mockAuth.creds, creds)
}

func TestDefaultAuthenticatorProvider(t *testing.T) {
	mockHTTPClient := &mockHTTPClient{}
	mockGRPCClient := &mockGRPCClient{}

	provider := NewDefaultAuthenticatorProvider(mockHTTPClient, mockGRPCClient)

	// Test HTTP round tripper
	rt, err := provider.GetHTTPRoundTripper(http.DefaultTransport)
	require.NoError(t, err)
	assert.Equal(t, mockHTTPClient.roundTripper, rt)

	// Test gRPC credentials
	creds, err := provider.GetGRPCCredentials()
	require.NoError(t, err)
	assert.Equal(t, mockGRPCClient.creds, creds)
}

func TestDefaultAuthenticatorProviderWithNilClients(t *testing.T) {
	provider := NewDefaultAuthenticatorProvider(nil, nil)

	// Test HTTP round tripper
	rt, err := provider.GetHTTPRoundTripper(http.DefaultTransport)
	require.NoError(t, err)
	assert.Equal(t, http.DefaultTransport, rt)

	// Test gRPC credentials
	creds, err := provider.GetGRPCCredentials()
	require.NoError(t, err)
	assert.Nil(t, creds)
}

// Mock implementations for testing

type mockAuthenticatorProvider struct {
	roundTripper http.RoundTripper
	creds        credentials.PerRPCCredentials
}

func (m *mockAuthenticatorProvider) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if m.roundTripper == nil {
		m.roundTripper = &mockRoundTripper{}
	}
	return m.roundTripper, nil
}

func (m *mockAuthenticatorProvider) GetGRPCCredentials() (credentials.PerRPCCredentials, error) {
	if m.creds == nil {
		m.creds = &mockCredentials{}
	}
	return m.creds, nil
}

type mockHTTPClient struct {
	roundTripper http.RoundTripper
}

func (m *mockHTTPClient) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if m.roundTripper == nil {
		m.roundTripper = &mockRoundTripper{}
	}
	return m.roundTripper, nil
}

type mockGRPCClient struct {
	creds credentials.PerRPCCredentials
}

func (m *mockGRPCClient) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if m.creds == nil {
		m.creds = &mockCredentials{}
	}
	return m.creds, nil
}

type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}

type mockCredentials struct{}

func (m *mockCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer mock-token"}, nil
}

func (m *mockCredentials) RequireTransportSecurity() bool {
	return false
}
