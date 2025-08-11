// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// otlpFactories creates factories that include OTLP components for testing sensitive data
func otlpFactories() (Factories, error) {
	var factories Factories
	var err error

	if factories.Receivers, err = MakeFactoryMap(otlpreceiver.NewFactory()); err != nil {
		return Factories{}, err
	}

	if factories.Exporters, err = MakeFactoryMap(otlpexporter.NewFactory()); err != nil {
		return Factories{}, err
	}

	if factories.Processors, err = MakeFactoryMap(processortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}

	if factories.Extensions, err = MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return Factories{}, err
	}

	if factories.Connectors, err = MakeFactoryMap(connectortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}

	return factories, nil
}

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
				URIs: []string{"yaml:processors::test/foo::timeout: 3s"},
				ProviderFactories: []confmap.ProviderFactory{
					yamlprovider.NewFactory(),
				},
				DefaultScheme: "yaml",
			},
		},
		{
			name: "valid URI - no provider set",
			set: confmap.ResolverSettings{
				URIs:          []string{"yaml:processors::test/foo::timeout: 3s"},
				DefaultScheme: "yaml",
			},
			errString: "at least one Provider must be supplied",
		},
		{
			name: "valid URI - invalid scheme name",
			set: confmap.ResolverSettings{
				URIs:          []string{"yaml:processors::test/foo::timeout: 3s"},
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
			URIs:          []string{"yaml:processors::test/foo::timeout: 3s"},
			DefaultScheme: "foo",
			ProviderFactories: []confmap.ProviderFactory{
				yamlprovider.NewFactory(),
			},
		},
	}}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()
	require.ErrorContains(t, err, "print-initial-config is currently experimental, use the otelcol.printInitialConfig feature gate to enable this command")
}

func TestPrintInitialConfigCommand(t *testing.T) {
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
			cmd := newPrintInitialConfigSubCommand(CollectorSettings{ConfigProviderSettings: set}, flags(featuregate.GlobalRegistry()))
			err := cmd.Execute()
			if test.errString != "" {
				require.ErrorContains(t, err, test.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPrintInitialConfigCommandFeaturegateDisabled(t *testing.T) {
	cmd := newPrintInitialConfigSubCommand(CollectorSettings{ConfigProviderSettings: ConfigProviderSettings{
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

func TestPrintInitialConfig(t *testing.T) {
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

			cmd := newPrintInitialConfigSubCommand(CollectorSettings{ConfigProviderSettings: set}, flags(featuregate.GlobalRegistry()))
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

func TestPrintTypedConfigCommand(t *testing.T) {
	// Test with minimal nop configuration
	factories, err := nopFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		Factories: func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{"yaml:service:\n  pipelines:\n    traces:\n      receivers: [nop]\n      exporters: [nop]"},
				ProviderFactories: []confmap.ProviderFactory{yamlprovider.NewFactory()},
				DefaultScheme:     "yaml",
			},
		},
	}

	cmd := newPrintTypedConfigSubCommand(set, flags(featuregate.GlobalRegistry()))
	require.NoError(t, cmd.Execute())
}

func TestPrintTypedConfigCommandJSON(t *testing.T) {
	// Test JSON format output with DefaultScheme override
	factories, err := nopFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		Factories: func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{"yaml:service:\n  pipelines:\n    traces:\n      receivers: [nop]\n      exporters: [nop]"},
				ProviderFactories: []confmap.ProviderFactory{yamlprovider.NewFactory()},
				DefaultScheme:     "yaml", // YAML as default scheme
			},
		},
	}

	// Create command with JSON format flag
	cmd := newPrintTypedConfigSubCommand(set, flags(featuregate.GlobalRegistry()))
	cmd.SetArgs([]string{"--format", "json"})
	require.NoError(t, cmd.Execute())
}

func TestPrintConfigCommandWithSensitiveData(t *testing.T) {
	// Test that print-config shows [REDACTED] for sensitive data
	factories, err := otlpFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		Factories: func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{"file:testdata/config_with_sensitive_data.yaml"},
				ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory()},
				DefaultScheme:     "file",
			},
		},
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newPrintTypedConfigSubCommand(set, flags(featuregate.GlobalRegistry()))
	err = cmd.Execute()

	// Restore stdout and get output
	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	// Should succeed with OTLP factories and show [REDACTED] for sensitive data
	require.NoError(t, err, "Command should execute successfully with OTLP factories")

	// Verify that sensitive data is redacted
	assert.Contains(t, output, "[REDACTED]")
	assert.NotContains(t, output, "Bearer secret-token")
	assert.NotContains(t, output, "super-secret-key")

	// Verify basic structure is still there
	assert.Contains(t, output, "receivers:")
	assert.Contains(t, output, "exporters:")
	assert.Contains(t, output, "service:")
}

func TestPrintInitialConfigWithSensitiveData(t *testing.T) {
	// Test that print-initial-config shows the raw configuration values (not redacted)
	require.NoError(t, featuregate.GlobalRegistry().Set(printCommandFeatureFlag.ID(), true))
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(printCommandFeatureFlag.ID(), false))
	}()

	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{"file:testdata/config_with_sensitive_data.yaml"},
			ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory()},
			DefaultScheme:     "file",
		},
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newPrintInitialConfigSubCommand(CollectorSettings{ConfigProviderSettings: set}, flags(featuregate.GlobalRegistry()))
	err := cmd.Execute()

	// Restore stdout and get output
	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	// Initial config should succeed since it doesn't validate against factories
	require.NoError(t, err, "Command should execute successfully")

	// Verify output contains the raw sensitive data, not [REDACTED]
	assert.Contains(t, output, "Bearer secret-token")
	assert.Contains(t, output, "super-secret-key")
	assert.NotContains(t, output, "[REDACTED]")

	// Verify basic structure is there
	assert.Contains(t, output, "receivers:")
	assert.Contains(t, output, "exporters:")
	assert.Contains(t, output, "service:")
}
