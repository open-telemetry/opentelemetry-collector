// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientcredentialsauthextension

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	expected := factory.CreateDefaultConfig().(*Config)
	expected.ClientSecret = "someclientsecret"
	expected.ClientID = "someclientid"
	expected.Scopes = []string{"api.metrics"}
	expected.TokenURL = "https://someserver.com/oauth2/default/v1/token"

	ext := cfg.Extensions[config.NewIDWithName(typeStr, "1")]
	assert.Equal(t,
		&Config{
			ExtensionSettings: config.NewExtensionSettings(config.NewIDWithName(typeStr, "1")),
			ClientSecret:      "someclientsecret",
			ClientID:          "someclientid",
			Scopes:            []string{"api.metrics"},
			TokenURL:          "https://someserver.com/oauth2/default/v1/token",
			Timeout:           time.Second,
		},
		ext)

	assert.Equal(t, 2, len(cfg.Service.Extensions))
	assert.Equal(t, config.NewIDWithName(typeStr, "1"), cfg.Service.Extensions[0])
}

func TestConfigTLSSettings(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	ext2 := cfg.Extensions[config.NewIDWithName(typeStr, "withtls")]

	cfg2 := ext2.(*Config)
	assert.Equal(t, cfg2.TLSSetting, configtls.TLSClientSetting{
		TLSSetting: configtls.TLSSetting{
			CAFile:   "cafile",
			CertFile: "certfile",
			KeyFile:  "keyfile",
		},
		Insecure:           true,
		InsecureSkipVerify: false,
		ServerName:         "",
	})
}

func TestLoadConfigError(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	tests := []struct {
		configFile  string
		expectedErr error
	}{
		{
			"config_missing_url",
			errNoTokenURLProvided,
		},
		{
			"config_missing_client_id",
			errNoClientIDProvided,
		},
		{
			"config_missing_client_secret",
			errNoClientSecretProvided,
		},
	}
	for _, tt := range tests {
		factory := NewFactory()
		factories.Extensions[typeStr] = factory
		configFile := fmt.Sprintf("%s.yaml", tt.configFile)
		_, ferr := configtest.LoadConfigAndValidate(path.Join(".", "testdata", configFile), factories)
		require.ErrorIs(t, ferr, tt.expectedErr)
	}
}
