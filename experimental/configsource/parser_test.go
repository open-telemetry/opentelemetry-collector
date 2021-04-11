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

package configsource

import (
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestConfigSourceParser(t *testing.T) {
	ctx := context.Background()

	testFactories := Factories{
		"tstcfgsrc": &mockCfgSrcFactory{},
	}
	tests := []struct {
		name      string
		file      string
		factories Factories
		expected  map[string]ConfigSettings
		wantErr   error
	}{
		{
			name:      "basic_config",
			file:      "basic_config",
			factories: testFactories,
			expected: map[string]ConfigSettings{
				"tstcfgsrc": &mockCfgSrcSettings{
					Settings: Settings{
						TypeVal: "tstcfgsrc",
						NameVal: "tstcfgsrc",
					},
					Endpoint: "some_endpoint",
					Token:    "some_token",
				},
				"tstcfgsrc/named": &mockCfgSrcSettings{
					Settings: Settings{
						TypeVal: "tstcfgsrc",
						NameVal: "tstcfgsrc/named",
					},
					Endpoint: "default_endpoint",
				},
			},
		},
		{
			name:      "bad_name",
			file:      "bad_name",
			factories: testFactories,
			wantErr:   &errInvalidTypeAndNameKey{},
		},
		{
			name: "missing_factory",
			file: "basic_config",
			factories: Factories{
				"not_in_basic_config": &mockCfgSrcFactory{},
			},
			wantErr: &errUnknownType{},
		},
		{
			name:      "unknown_field",
			file:      "unknown_field",
			factories: testFactories,
			wantErr:   &errUnmarshalError{},
		},
		{
			name:      "duplicated_name",
			file:      "duplicated_name",
			factories: testFactories,
			wantErr:   &errDuplicateName{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgFile := path.Join("testdata", tt.file+".yaml")
			v, err := config.NewParserFromFile(cfgFile)
			require.NoError(t, err)

			cfgSrcSettings, err := Load(ctx, v, tt.factories)
			require.IsType(t, tt.wantErr, err)
			assert.Equal(t, tt.expected, cfgSrcSettings)
		})
	}
}

type mockCfgSrcSettings struct {
	Settings
	Endpoint string `mapstructure:"endpoint"`
	Token    string `mapstructure:"token"`
}

var _ (ConfigSettings) = (*mockCfgSrcSettings)(nil)

type mockCfgSrcFactory struct {
	ErrOnCreateConfigSource error
}

var _ (Factory) = (*mockCfgSrcFactory)(nil)

func (m *mockCfgSrcFactory) Type() config.Type {
	return "tstcfgsrc"
}

func (m *mockCfgSrcFactory) CreateDefaultConfig() ConfigSettings {
	return &mockCfgSrcSettings{
		Settings: Settings{
			TypeVal: "tstcfgsrc",
		},
		Endpoint: "default_endpoint",
	}
}

func (m *mockCfgSrcFactory) CreateConfigSource(_ context.Context, _ CreateParams, cfg ConfigSettings) (ConfigSource, error) {
	if m.ErrOnCreateConfigSource != nil {
		return nil, m.ErrOnCreateConfigSource
	}
	return &testConfigSource{
		ValueMap: map[string]valueEntry{
			cfg.Name(): {
				Value: cfg,
			},
		},
	}, nil
}
