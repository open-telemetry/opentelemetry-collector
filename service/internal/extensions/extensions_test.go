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
		factories         component.Factories
		extensionsConfigs map[config.ComponentID]config.Extension
		serviceExtensions []config.ComponentID
		wantErrMsg        string
	}{
		{
			name: "extension_not_configured",
			serviceExtensions: []config.ComponentID{
				config.NewComponentID("myextension"),
			},
			wantErrMsg: "extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			extensionsConfigs: map[config.ComponentID]config.Extension{
				config.NewComponentID("unknown"): nopExtensionConfig,
			},
			serviceExtensions: []config.ComponentID{
				config.NewComponentID("unknown"),
			},
			wantErrMsg: "extension factory for type \"unknown\" is not configured",
		},
		{
			name: "error_on_create_extension",
			factories: component.Factories{
				Extensions: map[config.Type]component.ExtensionFactory{
					errExtensionFactory.Type(): errExtensionFactory,
				},
			},
			extensionsConfigs: map[config.ComponentID]config.Extension{
				config.NewComponentID(errExtensionFactory.Type()): errExtensionConfig,
			},
			serviceExtensions: []config.ComponentID{
				config.NewComponentID(errExtensionFactory.Type()),
			},
			wantErrMsg: "failed to create extension \"err\": cannot create \"err\" extension type",
		},
		{
			name: "bad_factory",
			factories: component.Factories{
				Extensions: map[config.Type]component.ExtensionFactory{
					badExtensionFactory.Type(): badExtensionFactory,
				},
			},
			extensionsConfigs: map[config.ComponentID]config.Extension{
				config.NewComponentID(badExtensionFactory.Type()): badExtensionCfg,
			},
			serviceExtensions: []config.ComponentID{
				config.NewComponentID(badExtensionFactory.Type()),
			},
			wantErrMsg: "factory for \"bf\" produced a nil extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Build(context.Background(), componenttest.NewNopTelemetrySettings(), component.NewDefaultBuildInfo(), tt.extensionsConfigs, tt.serviceExtensions, tt.factories.Extensions)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErrMsg)
		})
	}
}

func newBadExtensionFactory() component.ExtensionFactory {
	return component.NewExtensionFactory(
		"bf",
		func() config.Extension {
			return &struct {
				config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
			}{
				ExtensionSettings: config.NewExtensionSettings(config.NewComponentID("bf")),
			}
		},
		func(ctx context.Context, set component.ExtensionCreateSettings, extension config.Extension) (component.Extension, error) {
			return nil, nil
		},
	)
}

func newCreateErrorExtensionFactory() component.ExtensionFactory {
	return component.NewExtensionFactory(
		"err",
		func() config.Extension {
			return &struct {
				config.ExtensionSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
			}{
				ExtensionSettings: config.NewExtensionSettings(config.NewComponentID("err")),
			}
		},
		func(ctx context.Context, set component.ExtensionCreateSettings, extension config.Extension) (component.Extension, error) {
			return nil, errors.New("cannot create \"err\" extension type")
		},
	)
}
