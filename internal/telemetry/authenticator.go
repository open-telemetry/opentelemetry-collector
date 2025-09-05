// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/internal/telemetry"

import (
	"net/http"
	"sync"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/collector/extension/extensionauth"
)

// AuthenticatorProvider is an interface that provides authentication
// for internal telemetry exports.
type AuthenticatorProvider interface {
	// GetHTTPRoundTripper returns an authenticated round tripper for HTTP requests.
	GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error)

	// GetGRPCCredentials returns gRPC per-RPC credentials for authentication.
	GetGRPCCredentials() (credentials.PerRPCCredentials, error)
}

// AuthenticatorManager manages authentication for internal telemetry.
// This allows for late binding of authenticators since telemetry providers
// are created before extensions are initialized.
type AuthenticatorManager struct {
	mu            sync.RWMutex
	authenticator AuthenticatorProvider
}

// NewAuthenticatorManager creates a new AuthenticatorManager.
func NewAuthenticatorManager() *AuthenticatorManager {
	return &AuthenticatorManager{}
}

// SetAuthenticator sets the authenticator provider.
func (am *AuthenticatorManager) SetAuthenticator(auth AuthenticatorProvider) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.authenticator = auth
}

// GetHTTPRoundTripper returns an authenticated round tripper if an authenticator is set.
func (am *AuthenticatorManager) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.authenticator == nil {
		return base, nil
	}

	return am.authenticator.GetHTTPRoundTripper(base)
}

// GetGRPCCredentials returns gRPC credentials if an authenticator is set.
func (am *AuthenticatorManager) GetGRPCCredentials() (credentials.PerRPCCredentials, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.authenticator == nil {
		return nil, nil
	}

	return am.authenticator.GetGRPCCredentials()
}

// GlobalAuthenticatorManager is the global instance used by internal telemetry.
// This allows telemetry providers to be authenticated even when they're created
// before extensions are initialized.
var GlobalAuthenticatorManager = NewAuthenticatorManager()

// DefaultAuthenticatorProvider creates an AuthenticatorProvider from extension components.
type DefaultAuthenticatorProvider struct {
	httpClient extensionauth.HTTPClient
	grpcClient extensionauth.GRPCClient
}

// NewDefaultAuthenticatorProvider creates a new DefaultAuthenticatorProvider.
func NewDefaultAuthenticatorProvider(httpClient extensionauth.HTTPClient, grpcClient extensionauth.GRPCClient) *DefaultAuthenticatorProvider {
	return &DefaultAuthenticatorProvider{
		httpClient: httpClient,
		grpcClient: grpcClient,
	}
}

// GetHTTPRoundTripper implements AuthenticatorProvider.
func (p *DefaultAuthenticatorProvider) GetHTTPRoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if p.httpClient == nil {
		return base, nil
	}
	return p.httpClient.RoundTripper(base)
}

// GetGRPCCredentials implements AuthenticatorProvider.
func (p *DefaultAuthenticatorProvider) GetGRPCCredentials() (credentials.PerRPCCredentials, error) {
	if p.grpcClient == nil {
		return nil, nil
	}
	return p.grpcClient.PerRPCCredentials()
}
