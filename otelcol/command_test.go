// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/featuregate"
)

func TestNewCommandVersion(t *testing.T) {
	cmd := NewCommand(CollectorSettings{BuildInfo: component.BuildInfo{Version: "test_version"}})
	assert.Equal(t, "test_version", cmd.Version)
}

func TestNewCommandNoConfigURI(t *testing.T) {
	cmd := NewCommand(CollectorSettings{Factories: nopFactories})
	require.Error(t, cmd.Execute())
}

// This test emulates usage of Collector in Jaeger all-in-one, which
// allows running the binary with no explicit configuration.
func TestNewCommandProgrammaticallyPassedConfig(t *testing.T) {
	cmd := NewCommand(CollectorSettings{Factories: nopFactories})
	otelRunE := cmd.RunE
	cmd.RunE = func(c *cobra.Command, args []string) error {
		configFlag := c.Flag("config")
		cfg := `
service:
  extensions: [invalid_component_name]
receivers:
  invalid_component_name:
`
		require.NoError(t, configFlag.Value.Set("yaml:"+cfg))
		return otelRunE(cmd, args)
	}
	// verify that cmd.Execute was run with the implicitly provided config.
	require.ErrorContains(t, cmd.Execute(), "invalid_component_name")
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

	err = updateSettingsUsingFlags(&set, flgs)
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

	err = updateSettingsUsingFlags(&set, flgs)
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

	cmd := NewCommand(set)
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

	cmd := NewCommand(CollectorSettings{Factories: nopFactories, ConfigProviderSettings: set})
	require.Error(t, cmd.Execute())
}
