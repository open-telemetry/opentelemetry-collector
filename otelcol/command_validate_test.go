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

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/service/servicetest"
)

func TestNewValidateSubCommand(t *testing.T) {
	factories, err := servicetest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(
		ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:       []string{filepath.Join("testdata", "otelcol-nop.yaml")},
				Providers:  map[string]confmap.Provider{"file": fileprovider.New()},
				Converters: []confmap.Converter{expandconverter.New()},
			},
		})
	require.NoError(t, err)

	validateCmd := newValidateSubCommand(CollectorSettings{Factories: factories, ConfigProvider: cfgProvider})

	assert.Equal(t, "validate", validateCmd.Use)
	assert.Equal(t, "Validates the config without running the collector", validateCmd.Short)

	require.NoError(t, validateCmd.Execute())
}