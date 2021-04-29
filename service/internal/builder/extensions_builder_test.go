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
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/extension/extensionhelper"
)

func TestService_setupExtensions(t *testing.T) {
	errExtensionFactory := extensionhelper.NewFactory(
		"err",
		func() config.Extension {
			return &config.ExporterSettings{
				TypeVal: "err",
			}
		},
		func(ctx context.Context, params component.ExtensionCreateParams, extension config.Extension) (component.Extension, error) {
			return nil, fmt.Errorf("cannot create \"err\" extension type")
		},
	)
	errExtensionConfig := errExtensionFactory.CreateDefaultConfig()
	errExtensionConfig.SetName(string(errExtensionFactory.Type()))

	badExtensionFactory := newBadExtensionFactory()
	badExtensionCfg := badExtensionFactory.CreateDefaultConfig()
	badExtensionCfg.SetName(string(badExtensionFactory.Type()))

	tests := []struct {
		name       string
		factories  component.Factories
		config     *config.Config
		wantErrMsg string
	}{
		{
			name: "extension_not_configured",
			config: &config.Config{
				Service: config.Service{
					Extensions: []string{
						"myextension",
					},
				},
			},
			wantErrMsg: "extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			config: &config.Config{
				Extensions: map[string]config.Extension{
					string(errExtensionFactory.Type()): errExtensionConfig,
				},
				Service: config.Service{
					Extensions: []string{
						string(errExtensionFactory.Type()),
					},
				},
			},
			wantErrMsg: "extension factory for type \"err\" is not configured",
		},
		{
			name: "error_on_create_extension",
			factories: component.Factories{
				Extensions: map[config.Type]component.ExtensionFactory{
					errExtensionFactory.Type(): errExtensionFactory,
				},
			},
			config: &config.Config{
				Extensions: map[string]config.Extension{
					string(errExtensionFactory.Type()): errExtensionConfig,
				},
				Service: config.Service{
					Extensions: []string{
						string(errExtensionFactory.Type()),
					},
				},
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
			config: &config.Config{
				Extensions: map[string]config.Extension{
					string(badExtensionFactory.Type()): badExtensionCfg,
				},
				Service: config.Service{
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
			ext, err := BuildExtensions(zap.NewNop(), component.DefaultBuildInfo(), tt.config, tt.factories.Extensions)

			assert.Error(t, err)
			assert.Equal(t, tt.wantErrMsg, err.Error())
			assert.Equal(t, 0, len(ext))
		})
	}
}
