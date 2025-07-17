// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/featuregate"
)

func TestPrintCommand(t *testing.T) {
	tests := []struct {
		name      string
		set       confmap.ResolverSettings
		errString string
	}{
		{
			name:      "no URIs",
			set:       confmap.ResolverSettings{},
			errString: "at least one config flag must be provided",
		},
		{
			name: "valid URI - file not found",
			set: confmap.ResolverSettings{
				URIs: []string{"file:blabla.yaml"},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
				DefaultScheme: "file",
			},
			errString: "cannot retrieve the configuration: unable to read the file",
		},
		{
			name: "valid URI",
			set: confmap.ResolverSettings{
				URIs: []string{"yaml:processors::batch/foo::timeout: 3s"},
				ProviderFactories: []confmap.ProviderFactory{
					yamlprovider.NewFactory(),
				},
				DefaultScheme: "yaml",
			},
		},
		{
			name: "valid URI - no provider set",
			set: confmap.ResolverSettings{
				URIs:          []string{"yaml:processors::batch/foo::timeout: 3s"},
				DefaultScheme: "yaml",
			},
			errString: "at least one Provider must be supplied",
		},
		{
			name: "valid URI - invalid scheme name",
			set: confmap.ResolverSettings{
				URIs:          []string{"yaml:processors::batch/foo::timeout: 3s"},
				DefaultScheme: "foo",
				ProviderFactories: []confmap.ProviderFactory{
					yamlprovider.NewFactory(),
				},
			},
			errString: "configuration: DefaultScheme not found in providers list",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(printCommandFeatureFlag.ID(), true))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(printCommandFeatureFlag.ID(), false))
			}()

			set := ConfigProviderSettings{
				ResolverSettings: test.set,
			}
			cmd := newConfigPrintSubCommand(CollectorSettings{ConfigProviderSettings: set}, flags(featuregate.GlobalRegistry()))
			err := cmd.Execute()
			if test.errString != "" {
				require.ErrorContains(t, err, test.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPrintCommandFeaturegateDisabled(t *testing.T) {
	cmd := newConfigPrintSubCommand(CollectorSettings{ConfigProviderSettings: ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:          []string{"yaml:processors::batch/foo::timeout: 3s"},
			DefaultScheme: "foo",
			ProviderFactories: []confmap.ProviderFactory{
				yamlprovider.NewFactory(),
			},
		},
	}}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()
	require.ErrorContains(t, err, "print-initial-config is currently experimental, use the otelcol.printInitialConfig feature gate to enable this command")
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name        string
		configs     []string
		finalConfig string
	}{
		{
			name: "two-configs",
			configs: []string{
				"file:testdata/configs/1-config-first.yaml",
				"file:testdata/configs/1-config-second.yaml",
			},
			finalConfig: "testdata/configs/1-config-output.yaml",
		},
		{
			name: "two-configs-yaml",
			configs: []string{
				"file:testdata/configs/1-config-first.yaml",
				"file:testdata/configs/1-config-second.yaml",
				"yaml:service::pipelines::logs::receivers: [foo,bar]",
				"yaml:service::pipelines::logs::exporters: [foo,bar]",
			},
			finalConfig: "testdata/configs/2-config-output.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(printCommandFeatureFlag.ID(), true))
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(printCommandFeatureFlag.ID(), false))
			}()
			set := ConfigProviderSettings{
				ResolverSettings: confmap.ResolverSettings{
					URIs: test.configs,
					ProviderFactories: []confmap.ProviderFactory{
						fileprovider.NewFactory(),
						yamlprovider.NewFactory(),
					},
					DefaultScheme: "file",
				},
			}
			tmpFile, err := os.CreateTemp(t.TempDir(), "*")
			require.NoError(t, err)
			t.Cleanup(func() { _ = tmpFile.Close() })

			// save the os.Stdout and temporarily set it to temp file
			oldStdout := os.Stdout
			os.Stdout = tmpFile

			cmd := newConfigPrintSubCommand(CollectorSettings{ConfigProviderSettings: set}, flags(featuregate.GlobalRegistry()))
			require.NoError(t, cmd.Execute())

			// restore os.Stdout
			os.Stdout = oldStdout

			expectedOutput, err := os.ReadFile(test.finalConfig)
			require.NoError(t, err)

			actualOutput, err := os.ReadFile(tmpFile.Name())
			require.NoError(t, err)

			actualConfig := make(map[string]any, 0)
			expectedConfig := make(map[string]any, 0)

			require.NoError(t, yaml.Unmarshal(bytes.TrimSpace(actualOutput), actualConfig))
			require.NoError(t, yaml.Unmarshal(bytes.TrimSpace(expectedOutput), expectedConfig))

			require.Equal(t, expectedConfig, actualConfig)
		})
	}
}
