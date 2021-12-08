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

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/internalinterface"
	"go.opentelemetry.io/collector/internal/testcomponents"
)

func TestValidateConfigFromFactories_Success(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	require.NoError(t, err)

	err = validateConfigFromFactories(factories)
	require.NoError(t, err)
}

func TestValidateConfigFromFactories_Failure(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	require.NoError(t, err)

	// Add a factory returning config not following pattern to force error.
	f := &badConfigExtensionFactory{}
	factories.Extensions[f.Type()] = f

	err = validateConfigFromFactories(factories)
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
