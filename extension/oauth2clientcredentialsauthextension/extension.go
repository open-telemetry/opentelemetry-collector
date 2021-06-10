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

package oauth2clientcredentialsauthextension

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
	grpcOAuth "google.golang.org/grpc/credentials/oauth"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
)

// ClientCredentialsAuthenticator provides implementation for providing client authentication using OAuth2 client credentials
// workflow for both gRPC and HTTP clients.
type ClientCredentialsAuthenticator struct {
	clientCredentials *clientcredentials.Config
	logger            *zap.Logger
	client            *http.Client
}

// ClientCredentialsAuthenticator implements both HTTPClientAuth and GRPCClientAuth
var (
	_ configauth.HTTPClientAuthenticator = (*ClientCredentialsAuthenticator)(nil)
	_ configauth.GRPCClientAuthenticator = (*ClientCredentialsAuthenticator)(nil)
)

func newClientCredentialsExtension(cfg *Config, logger *zap.Logger) (*ClientCredentialsAuthenticator, error) {
	if cfg.ClientID == "" {
		return nil, errNoClientIDProvided
	}
	if cfg.ClientSecret == "" {
		return nil, errNoClientSecretProvided
	}
	if cfg.TokenURL == "" {
		return nil, errNoTokenURLProvided
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()

	tlsCfg, err := cfg.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, err
	}
	transport.TLSClientConfig = tlsCfg

	return &ClientCredentialsAuthenticator{
		clientCredentials: &clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Scopes:       cfg.Scopes,
		},
		logger: logger,
		client: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		},
	}, nil
}

// Start for ClientCredentialsAuthenticator extension does nothing
func (o *ClientCredentialsAuthenticator) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown for ClientCredentialsAuthenticator extension does nothing
func (o *ClientCredentialsAuthenticator) Shutdown(_ context.Context) error {
	return nil
}

// RoundTripper returns oauth2.Transport, an http.RoundTripper that performs "client-credential" OAuth flow and
// also auto refreshes OAuth tokens as needed.
func (o *ClientCredentialsAuthenticator) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, o.client)
	return &oauth2.Transport{
		Source: o.clientCredentials.TokenSource(ctx),
		Base:   base,
	}, nil
}

// PerRPCCredentials returns gRPC PerRPCCredentials that supports "client-credential" OAuth flow. The underneath
// oauth2.clientcredentials.Config instance will manage tokens performing auto refresh as necessary.
func (o *ClientCredentialsAuthenticator) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, o.client)
	return grpcOAuth.TokenSource{
		TokenSource: o.clientCredentials.TokenSource(ctx),
	}, nil
}
