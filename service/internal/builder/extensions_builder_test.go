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

package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/testcomponents"
)

func TestService_setupExtensions(t *testing.T) {
	exampleExtensionFactory := &testcomponents.ExampleExtensionFactory{FailCreation: true}
	exampleExtensionConfig := &testcomponents.ExampleExtensionCfg{
		ExtensionSettings: configmodels.ExtensionSettings{
			TypeVal: exampleExtensionFactory.Type(),
			NameVal: string(exampleExtensionFactory.Type()),
		},
	}

	badExtensionFactory := newBadExtensionFactory()
	badExtensionFactoryCfg := badExtensionFactory.CreateDefaultConfig()
	badExtensionFactoryCfg.SetName("bf")

	tests := []struct {
		name       string
		factories  component.Factories
		config     *configmodels.Config
		wantErrMsg string
	}{
		{
			name: "extension_not_configured",
			config: &configmodels.Config{
				Service: configmodels.Service{
					Extensions: []string{
						"myextension",
					},
				},
			},
			wantErrMsg: "extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					string(exampleExtensionFactory.Type()): exampleExtensionConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						string(exampleExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "extension factory for type \"exampleextension\" is not configured",
		},
		{
			name: "error_on_create_extension",
			factories: component.Factories{
				Extensions: map[configmodels.Type]component.ExtensionFactory{
					exampleExtensionFactory.Type(): exampleExtensionFactory,
				},
			},
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					string(exampleExtensionFactory.Type()): exampleExtensionConfig,
				},
				Service: configmodels.Service{
					Extensions: []string{
						string(exampleExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "failed to create extension \"exampleextension\": cannot create \"exampleextension\" extension type",
		},
		{
			name: "bad_factory",
			factories: component.Factories{
				Extensions: map[configmodels.Type]component.ExtensionFactory{
					badExtensionFactory.Type(): badExtensionFactory,
				},
			},
			config: &configmodels.Config{
				Extensions: map[string]configmodels.Extension{
					string(badExtensionFactory.Type()): badExtensionFactoryCfg,
				},
				Service: configmodels.Service{
					Extensions: []string{
						string(badExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "factory for \"bf\" produced a nil extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, err := BuildExtensions(zap.NewNop(), component.DefaultApplicationStartInfo(), tt.config, tt.factories.Extensions)

			assert.Error(t, err)
			assert.Equal(t, tt.wantErrMsg, err.Error())
			assert.Equal(t, 0, len(ext))
		})
	}
}
