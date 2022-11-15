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

package extensions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/service"
)

func TestBuildExtensions(t *testing.T) {
	nopExtensionFactory := componenttest.NewNopExtensionFactory()
	nopExtensionConfig := nopExtensionFactory.CreateDefaultConfig()
	errExtensionFactory := newCreateErrorExtensionFactory()
	errExtensionConfig := errExtensionFactory.CreateDefaultConfig()
	badExtensionFactory := newBadExtensionFactory()
	badExtensionCfg := badExtensionFactory.CreateDefaultConfig()

	tests := []struct {
		name              string
		factories         service.Factories
		extensionsConfigs map[component.ID]extension.Config
		serviceExtensions []component.ID
		wantErrMsg        string
	}{
		{
			name: "extension_not_configured",
			serviceExtensions: []component.ID{
				component.NewID("myextension"),
			},
			wantErrMsg: "extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			extensionsConfigs: map[component.ID]extension.Config{
				component.NewID("unknown"): nopExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.NewID("unknown"),
			},
			wantErrMsg: "extension factory for type \"unknown\" is not configured",
		},
		{
			name: "error_on_create_extension",
			factories: service.Factories{
				Extensions: map[component.Type]extension.Factory{
					errExtensionFactory.Type(): errExtensionFactory,
				},
			},
			extensionsConfigs: map[component.ID]extension.Config{
				component.NewID(errExtensionFactory.Type()): errExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.NewID(errExtensionFactory.Type()),
			},
			wantErrMsg: "failed to create extension \"err\": cannot create \"err\" extension type",
		},
		{
			name: "bad_factory",
			factories: service.Factories{
				Extensions: map[component.Type]extension.Factory{
					badExtensionFactory.Type(): badExtensionFactory,
				},
			},
			extensionsConfigs: map[component.ID]extension.Config{
				component.NewID(badExtensionFactory.Type()): badExtensionCfg,
			},
			serviceExtensions: []component.ID{
				component.NewID(badExtensionFactory.Type()),
			},
			wantErrMsg: "factory for \"bf\" produced a nil extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(context.Background(), Settings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				Configs:   tt.extensionsConfigs,
				Factories: tt.factories.Extensions,
			}, tt.serviceExtensions)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErrMsg)
		})
	}
}

func newBadExtensionFactory() extension.Factory {
	return extension.NewExtensionFactory(
		"bf",
		func() extension.Config {
			return &struct {
				config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
			}{
				ExtensionSettings: config.NewExtensionSettings(component.NewID("bf")),
			}
		},
		func(ctx context.Context, set extension.CreateSettings, extension extension.Config) (extension.Extension, error) {
			return nil, nil
		},
		component.StabilityLevelInDevelopment,
	)
}

func newCreateErrorExtensionFactory() extension.Factory {
	return extension.NewExtensionFactory(
		"err",
		func() extension.Config {
			return &struct {
				config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
			}{
				ExtensionSettings: config.NewExtensionSettings(component.NewID("err")),
			}
		},
		func(ctx context.Context, set extension.CreateSettings, extension extension.Config) (extension.Extension, error) {
			return nil, errors.New("cannot create \"err\" extension type")
		},
		component.StabilityLevelInDevelopment,
	)
}
