// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
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
	cmd := NewCommand(CollectorSettings{Factories: nopFactories, ConfigProviderSettings: ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			ProviderFactories: []confmap.ProviderFactory{confmap.NewProviderFactory(newFailureProvider)},
			DefaultScheme:     "file",
		},
	}})
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
	filePath := filepath.Join("testdata", "otelcol-invalid.yaml")
	fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, filePath))
	})
	set := CollectorSettings{
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{filePath},
				ProviderFactories: []confmap.ProviderFactory{fileProvider},
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

func TestInvalidCollectorSettings(t *testing.T) {
	set := CollectorSettings{
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{"--config=otelcol-nop.yaml"},
			},
		},
	}

	cmd := NewCommand(set)
	require.Error(t, cmd.Execute())
}

func TestNewCommandInvalidComponent(t *testing.T) {
	filePath := filepath.Join("testdata", "otelcol-invalid.yaml")
	fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, filePath))
	})
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{filePath},
			ProviderFactories: []confmap.ProviderFactory{fileProvider},
		},
	}

	cmd := NewCommand(CollectorSettings{Factories: nopFactories, ConfigProviderSettings: set})
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

	err = updateSettingsUsingFlags(&set, flgs)
	require.ErrorContains(t, err, "at least one Provider must be supplied")
}

func Test_UseUnifiedEnvVarExpansionRules(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "default scheme set",
			input:    "file",
			expected: "file",
		},
		{
			name:     "default scheme not set",
			input:    "",
			expected: "env",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
				return &confmap.Retrieved{}, nil
			})
			set := CollectorSettings{
				ConfigProviderSettings: ConfigProviderSettings{
					ResolverSettings: confmap.ResolverSettings{
						ProviderFactories: []confmap.ProviderFactory{fileProvider},
						DefaultScheme:     tt.input,
					},
				},
			}
			flgs := flags(featuregate.NewRegistry())
			err := flgs.Parse([]string{"--config=otelcol-nop.yaml"})
			require.NoError(t, err)

			err = updateSettingsUsingFlags(&set, flgs)
			require.NoError(t, err)
			require.Equal(t, tt.expected, set.ConfigProviderSettings.ResolverSettings.DefaultScheme)
		})
	}
}
