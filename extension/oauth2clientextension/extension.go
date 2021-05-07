// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oauth2clientextension

import (
	"context"
	"errors"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
	grpcOAuth "google.golang.org/grpc/credentials/oauth"
	"net/http"
)

var (
	errNoClientIDProvided     = errors.New("no ClientID provided in the OAuth2 exporter configuration")
	errNoTokenURLProvided     = errors.New("no TokenURL provided in OAuth Client Credentials configuration")
	errNoClientSecretProvided = errors.New("no ClientSecret provided in OAuth Client Credentials configuration")
)

// OAuth2Authenticator provides implementation for providing client authentication using OAuth2 client credentials
// workflow for both gRPC and HTTP clients.
type OAuth2Authenticator struct {
	clientCredentials *clientcredentials.Config
	logger            *zap.Logger
}

// OAuth2Authenticator implements both HTTPClientAuth and GRPCClientAuth
var (
	_ configauth.HTTPClientAuth = (*OAuth2Authenticator)(nil)
	_ configauth.GRPCClientAuth = (*OAuth2Authenticator)(nil)
)

func newOAuth2Extension(cfg *OAuth2ClientSettings, logger *zap.Logger) (*OAuth2Authenticator, error) {
	if cfg.ClientID == "" {
		return nil, errNoClientIDProvided
	}
	if cfg.ClientSecret == "" {
		return nil, errNoClientSecretProvided
	}
	if cfg.TokenURL == "" {
		return nil, errNoTokenURLProvided
	}
	return &OAuth2Authenticator{
		clientCredentials: &clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Scopes:       cfg.Scopes,
		},
		logger: logger,
	}, nil
}

func (O *OAuth2Authenticator) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (O *OAuth2Authenticator) Shutdown(ctx context.Context) error {
	return nil
}

// RoundTripper returns oauth2.Transport, an http.RoundTripper that performs "client-credential" OAuth flow and
// also auto refreshes OAuth tokens as needed.
func (O *OAuth2Authenticator) RoundTripper(base http.RoundTripper) http.RoundTripper {
	return &oauth2.Transport{
		Source: O.clientCredentials.TokenSource(context.Background()),
		Base:   base,
	}
}

// PerRPCCredential returns gRPC PerRPCCredentials that supports "client-credential" OAuth flow. The underneath
// oauth2.clientcredentials.Config instance will manage tokens performing auto refresh as necessary.
func (O *OAuth2Authenticator) PerRPCCredential() (credentials.PerRPCCredentials, error) {
	return grpcOAuth.TokenSource{
		TokenSource: O.clientCredentials.TokenSource(context.Background()),
	}, nil
}
