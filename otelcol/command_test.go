// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

func TestNewCommandVersion(t *testing.T) {
	cmd := NewCommand(CollectorSettings{BuildInfo: component.BuildInfo{Version: "test_version"}})
	assert.Equal(t, "test_version", cmd.Version)
}

func TestNewCommandNoConfigURI(t *testing.T) {
	cmd := NewCommand(CollectorSettings{Factories: nopFactories})
	require.Error(t, cmd.Execute())
}

func TestNewCommandInvalidComponent(t *testing.T) {
	cfgProvider, err := NewConfigProvider(
		ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:       []string{filepath.Join("testdata", "otelcol-invalid.yaml")},
				Providers:  map[string]confmap.Provider{"file": fileprovider.New()},
				Converters: []confmap.Converter{expandconverter.New()},
			},
		})
	require.NoError(t, err)

	cmd := NewCommand(CollectorSettings{Factories: nopFactories, ConfigProvider: cfgProvider})
	require.Error(t, cmd.Execute())
}
