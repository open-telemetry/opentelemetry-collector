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

package otelcol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/internalinterface"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/service"
)

func TestNewCommand(t *testing.T) {
	set := service.CollectorSettings{}
	cmd := NewCobraCommand(set)
	err := cmd.Execute()
	require.Error(t, err)
	assert.Equal(t, set.BuildInfo.Version, cmd.Version)
}

func TestNewCommandMapProviderIsNil(t *testing.T) {
	settings := service.CollectorSettings{}
	settings.ConfigProvider = nil
	cmd := NewCobraCommand(settings)
	err := cmd.Execute()
	require.Error(t, err)
}

func TestNewCommandInvalidFactories(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	require.NoError(t, err)
	f := &badConfigExtensionFactory{}
	factories.Extensions[f.Type()] = f
	settings := service.CollectorSettings{Factories: factories}
	cmd := NewCobraCommand(settings)
	err = cmd.Execute()
	require.Error(t, err)
}

// badConfigExtensionFactory was created to force error path from factory returning
// a config not satisfying the validation.
type badConfigExtensionFactory struct {
	internalinterface.BaseInternal
}

func (b badConfigExtensionFactory) Type() config.Type {
	return "bad_config"
}

func (b badConfigExtensionFactory) CreateDefaultConfig() config.Extension {
	return &struct {
		config.ExtensionSettings
		BadTagField int `mapstructure:"tag-with-dashes"`
	}{}
}

func (b badConfigExtensionFactory) CreateExtension(_ context.Context, _ component.ExtensionCreateSettings, _ config.Extension) (component.Extension, error) {
	return nil, nil
}
