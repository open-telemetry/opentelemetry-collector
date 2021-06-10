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
	"errors"
	"time"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtls"
)

var (
	errNoClientIDProvided     = errors.New("no ClientID provided in the OAuth2 exporter configuration")
	errNoTokenURLProvided     = errors.New("no TokenURL provided in OAuth Client Credentials configuration")
	errNoClientSecretProvided = errors.New("no ClientSecret provided in OAuth Client Credentials configuration")
)

// Config stores the configuration for OAuth2 Client Credentials (2-legged OAuth2 flow) setup
type Config struct {
	config.ExtensionSettings `mapstructure:",squash"`

	// ClientID is the application's ID.
	ClientID string `mapstructure:"client_id"`

	// ClientSecret is the application's secret.
	ClientSecret string `mapstructure:"client_secret"`

	// TokenURL is the resource server's token endpoint
	// URL. This is a constant specific to each server.
	TokenURL string `mapstructure:"token_url"`

	// Scope specifies optional requested permissions.
	Scopes []string `mapstructure:"scopes,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:"tls,omitempty"`

	// Timeout parameter configures `http.Client.Timeout`.
	Timeout time.Duration `mapstructure:"timeout,omitempty"`
}

var _ config.Extension = (*Config)(nil)

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	if cfg.ClientID == "" {
		return errNoClientIDProvided
	}
	if cfg.ClientSecret == "" {
		return errNoClientSecretProvided
	}
	if cfg.TokenURL == "" {
		return errNoTokenURLProvided
	}
	return nil
}
