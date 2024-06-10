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
	"go.opentelemetry.io/collector/featuregate"
)

func TestNewCommandVersion(t *testing.T) {
	cmd := NewCommandMustProviderSettings(CollectorSettings{BuildInfo: component.BuildInfo{Version: "test_version"}})
	assert.Equal(t, "test_version", cmd.Version)
}

func TestNewCommandNoConfigURI(t *testing.T) {
	cmd := NewCommandMustProviderSettings(CollectorSettings{Factories: nopFactories})
	require.Error(t, cmd.Execute())
}

func TestAddFlagToSettings(t *testing.T) {
	set := CollectorSettings{
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:               []string{filepath.Join("testdata", "otelcol-invalid.yaml")},
				ProviderFactories:  []confmap.ProviderFactory{fileprovider.NewFactory()},
				ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
			},
		},
	}
	flgs := flags(featuregate.NewRegistry())
	err := flgs.Parse([]string{"--config=otelcol-nop.yaml"})
	require.NoError(t, err)

	err = updateSettingsUsingFlags(&set, flgs, false)
	require.NoError(t, err)
	require.Len(t, set.ConfigProviderSettings.ResolverSettings.URIs, 1)
}

func TestAddDefaultConfmapModules(t *testing.T) {
	set := CollectorSettings{
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{},
		},
	}
	flgs := flags(featuregate.NewRegistry())
	err := flgs.Parse([]string{"--config=otelcol-nop.yaml"})
	require.NoError(t, err)

	err = updateSettingsUsingFlags(&set, flgs, false)
	require.NoError(t, err)
	require.Len(t, set.ConfigProviderSettings.ResolverSettings.URIs, 1)
	require.Len(t, set.ConfigProviderSettings.ResolverSettings.ConverterFactories, 1)
	require.Len(t, set.ConfigProviderSettings.ResolverSettings.ProviderFactories, 5)
}

func TestInvalidCollectorSettings(t *testing.T) {
	set := CollectorSettings{
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
				URIs:               []string{"--config=otelcol-nop.yaml"},
			},
		},
	}

	cmd := NewCommandMustProviderSettings(set)
	require.Error(t, cmd.Execute())
}

func TestNewCommandInvalidComponent(t *testing.T) {
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:               []string{filepath.Join("testdata", "otelcol-invalid.yaml")},
			ProviderFactories:  []confmap.ProviderFactory{fileprovider.NewFactory()},
			ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
		},
	}

	cmd := NewCommandMustProviderSettings(CollectorSettings{Factories: nopFactories, ConfigProviderSettings: set})
	require.Error(t, cmd.Execute())
}

func TestNoProvidersReturnsError(t *testing.T) {
	set := CollectorSettings{
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filepath.Join("testdata", "otelcol-invalid.yaml")},
			},
		},
	}
	flgs := flags(featuregate.NewRegistry())
	err := flgs.Parse([]string{"--config=otelcol-nop.yaml"})
	require.NoError(t, err)

	err = updateSettingsUsingFlags(&set, flgs, true)
	require.ErrorContains(t, err, "at least one provider or converter must be provided")
}
