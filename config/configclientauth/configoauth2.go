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

package configclientauth

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
	grpcOAuth "google.golang.org/grpc/credentials/oauth"
)

var (
	errNoClientIDProvided     = errors.New("no ClientID provided in the OAuth2 exporter configuration")
	errNoTokenURLProvided     = errors.New("no TokenURL provided in OAuth Client Credentials configuration")
	errNoClientSecretProvided = errors.New("no ClientSecret provided in OAuth Client Credentials configuration")
)

// OAuth2ClientSettings stores the configuration for OAuth2 Client Credentials (2-legged OAuth2 flow) setup
type OAuth2ClientSettings struct {
	// ClientID is the application's ID.
	ClientID string `mapstructure:"client_id"`

	// ClientSecret is the application's secret.
	ClientSecret string `mapstructure:"client_secret"`

	// TokenURL is the resource server's token endpoint
	// URL. This is a constant specific to each server.
	TokenURL string `mapstructure:"token_url"`

	// Scope specifies optional requested permissions.
	Scopes []string `mapstructure:"scopes,omitempty"`
}

// buildOAuth2ClientCredentials maps OAuth2ClientSettings to a build oauth2.clientcredentials.Config
func buildOAuth2ClientCredentials(c *OAuth2ClientSettings) (clientcredentials.Config, error) {
	if c.ClientID == "" {
		return clientcredentials.Config{}, errNoClientIDProvided
	}
	if c.ClientSecret == "" {
		return clientcredentials.Config{}, errNoClientSecretProvided
	}
	if c.TokenURL == "" {
		return clientcredentials.Config{}, errNoTokenURLProvided
	}
	return clientcredentials.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		TokenURL:     c.TokenURL,
		Scopes:       c.Scopes,
	}, nil
}

// OAuth2RoundTripper returns oauth2.Transport, an http.RoundTripper that performs "client-credential" OAuth flow and
// also auto refreshes OAuth tokens as needed.
func OAuth2RoundTripper(c *OAuth2ClientSettings, base http.RoundTripper) (http.RoundTripper, error) {
	cc, err := buildOAuth2ClientCredentials(c)
	if err != nil {
		return nil, err
	}
	return &oauth2.Transport{
		Source: cc.TokenSource(context.Background()),
		Base:   base,
	}, nil
}

// OAuth2ClientCredentialsPerRPCCredentials returns gRPC PerRPCCredentials that supports "client-credential" OAuth flow. The underneath
// oauth2.clientcredentials.Config instance will manage tokens performing auto refresh as necessary.
func OAuth2ClientCredentialsPerRPCCredentials(c *OAuth2ClientSettings) (credentials.PerRPCCredentials, error) {
	cc, err := buildOAuth2ClientCredentials(c)
	if err != nil {
		return nil, err
	}
	return grpcOAuth.TokenSource{TokenSource: cc.TokenSource(context.Background())}, nil
}
