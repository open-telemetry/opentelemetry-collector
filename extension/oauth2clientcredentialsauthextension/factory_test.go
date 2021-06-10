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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
)

func TestCreateDefaultConfig(t *testing.T) {
	// prepare and test
	expected := &Config{
		ExtensionSettings: config.NewExtensionSettings(config.NewID(typeStr)),
	}

	// test
	cfg := createDefaultConfig()

	// verify
	assert.Equal(t, expected, cfg)
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateExtension(t *testing.T) {
	cfg := createDefaultConfig().(*Config)

	tests := []struct {
		name        string
		settings    *Config
		shouldError bool
		expectedErr error
	}{
		{
			name: "valid_settings",
			settings: &Config{
				ClientID:     "testclientid",
				ClientSecret: "testsecret",
				TokenURL:     "https://example.com/v1/token",
				Scopes:       []string{"resource.read"},
			},
			shouldError: false,
		},
		{
			name: "invalid_client_settings_should_error",
			settings: &Config{
				ClientID: "testclientid",
				TokenURL: "https://example.com/v1/token",
				Scopes:   []string{"resource.read"},
			},
			shouldError: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			cfg.ClientID = testcase.settings.ClientID
			cfg.ClientSecret = testcase.settings.ClientSecret
			cfg.TokenURL = testcase.settings.TokenURL
			cfg.Scopes = testcase.settings.Scopes
			ext, err := createExtension(context.Background(), component.ExtensionCreateSettings{Logger: zap.NewNop()}, cfg)
			if testcase.shouldError {
				assert.Error(t, err)
				assert.Nil(t, ext)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, ext)
			}
		})
	}
}

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.NotNil(t, f)
}
