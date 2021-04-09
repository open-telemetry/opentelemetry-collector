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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
)

func TestConfigSourceBuild(t *testing.T) {
	ctx := context.Background()
	params := CreateParams{
		Logger:               zap.NewNop(),
		ApplicationStartInfo: component.ApplicationStartInfo{},
	}

	testFactories := Factories{
		"tstcfgsrc": &mockCfgSrcFactory{},
	}

	tests := []struct {
		name               string
		configSettings     map[string]ConfigSettings
		factories          Factories
		expectedCfgSources map[string]ConfigSource
		wantErr            error
	}{
		{
			name: "unknown_config_source",
			configSettings: map[string]ConfigSettings{
				"tstcfgsrc": &mockCfgSrcSettings{
					Settings: Settings{
						TypeVal: "unknown_config_source",
						NameVal: "tstcfgsrc",
					},
				},
			},
			factories: testFactories,
			wantErr:   &errUnknownType{},
		},
		{
			name: "creation_error",
			configSettings: map[string]ConfigSettings{
				"tstcfgsrc": &mockCfgSrcSettings{
					Settings: Settings{
						TypeVal: "tstcfgsrc",
						NameVal: "tstcfgsrc",
					},
				},
			},
			factories: Factories{
				"tstcfgsrc": &mockCfgSrcFactory{
					ErrOnCreateConfigSource: errors.New("forced test error"),
				},
			},
			wantErr: &errConfigSourceCreation{},
		},
		{
			name: "factory_return_nil",
			configSettings: map[string]ConfigSettings{
				"tstcfgsrc": &mockCfgSrcSettings{
					Settings: Settings{
						TypeVal: "tstcfgsrc",
						NameVal: "tstcfgsrc",
					},
				},
			},
			factories: Factories{
				"tstcfgsrc": &mockNilCfgSrcFactory{},
			},
			wantErr: &errFactoryCreatedNil{},
		},
		{
			name: "base_case",
			configSettings: map[string]ConfigSettings{
				"tstcfgsrc/named": &mockCfgSrcSettings{
					Settings: Settings{
						TypeVal: "tstcfgsrc",
						NameVal: "tstcfgsrc/named",
					},
					Endpoint: "some_endpoint",
					Token:    "some_token",
				},
			},
			factories: testFactories,
			expectedCfgSources: map[string]ConfigSource{
				"tstcfgsrc/named": &testConfigSource{
					ValueMap: map[string]valueEntry{
						"tstcfgsrc/named": {
							Value: &mockCfgSrcSettings{
								Settings: Settings{
									TypeVal: "tstcfgsrc",
									NameVal: "tstcfgsrc/named",
								},
								Endpoint: "some_endpoint",
								Token:    "some_token",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builtCfgSources, err := Build(ctx, tt.configSettings, params, tt.factories)
			require.IsType(t, tt.wantErr, err)
			require.Equal(t, tt.expectedCfgSources, builtCfgSources)
		})
	}
}

type mockNilCfgSrcFactory struct{}

func (m *mockNilCfgSrcFactory) Type() config.Type {
	return "tstcfgsrc"
}

var _ (Factory) = (*mockNilCfgSrcFactory)(nil)

func (m *mockNilCfgSrcFactory) CreateDefaultConfig() ConfigSettings {
	return &mockCfgSrcSettings{
		Settings: Settings{
			TypeVal: "tstcfgsrc",
		},
		Endpoint: "default_endpoint",
	}
}

func (m *mockNilCfgSrcFactory) CreateConfigSource(context.Context, CreateParams, ConfigSettings) (ConfigSource, error) {
	return nil, nil
}
