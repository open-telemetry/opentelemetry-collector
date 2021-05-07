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

import "go.opentelemetry.io/collector/config"

// OAuth2ClientSettings stores the configuration for OAuth2 Client Credentials (2-legged OAuth2 flow) setup
type OAuth2ClientSettings struct {
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
}
