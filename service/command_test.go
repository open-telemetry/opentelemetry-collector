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

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
)

func TestNewCommand(t *testing.T) {
	set := CollectorSettings{}
	cmd := NewCommand(set)
	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, set.BuildInfo.Version, cmd.Version)
}

func TestNewCommandMapProviderIsNil(t *testing.T) {
	settings := CollectorSettings{}
	settings.ConfigProvider = nil
	cmd := NewCommand(settings)
	err := cmd.Execute()
	require.Error(t, err)
}

func TestNewCommandInvalidFactories(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	factories.Extensions[badConfigExtensionFactory.Type()] = badConfigExtensionFactory
	settings := CollectorSettings{Factories: factories}
	cmd := NewCommand(settings)
	err = cmd.Execute()
	require.Error(t, err)
}

// badConfigExtensionFactory was created to force error path from factory returning
// a config not satisfying the validation.
var badConfigExtensionFactory = component.NewExtensionFactory(
	"bad_config",
	func() config.Extension {
		return &struct {
			config.ExtensionSettings
			BadTagField int `mapstructure:"tag-with-dashes"`
		}{}
	},
	func(context.Context, component.ExtensionCreateSettings, config.Extension) (component.Extension, error) {
		return nil, nil
	})
