// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/otelcol/internal/metadata"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/collector/service/telemetry"
)

type printExporterConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

type printReceiverConfig struct {
	Opaque configopaque.String `mapstructure:"opaque"`
	Other  string              `mapstructure:"other,omitempty"`
}

func (c *printExporterConfig) Validate() error {
	if c.Timeout < 0 {
		return errors.New("timeout cannot be negative")
	}
	return nil
}

func TestPrintCommand(t *testing.T) {
	const nonexistentConfig = "file:nope.yaml"

	validConfig := fmt.Sprint("file:", filepath.Join("testdata", "print.yaml"))
	invalidConfig1 := fmt.Sprint("file:", filepath.Join("testdata", "print_invalid.yaml"))
	invalidConfig2 := fmt.Sprint("file:", filepath.Join("testdata", "print_negative.yaml"))
	defaultConfig := fmt.Sprint("file:", filepath.Join("testdata", "print_default.yaml"))

	tests := []struct {
		name            string
		ofmt            string
		path            string
		errString       string
		outString       map[string]string
		disableFF       bool // disable the feature flag
		validate        bool // add validation (even redacted)
		errOnlyRedacted bool // error applies only in redacted mode
	}{
		{
			name:      "file not found",
			path:      nonexistentConfig,
			errString: "cannot retrieve the configuration: unable to read the file",
		},
		{
			name: "valid yaml",
			path: validConfig,
		},
		{
			name:      "invalid syntax without validate",
			path:      invalidConfig1,
			errString: "'timeout' time: invalid duration",
		},
		{
			name:      "validation fail",
			path:      invalidConfig2,
			validate:  true,
			errString: "timeout cannot be negative",
		},
		{
			name:      "no feature flag",
			path:      validConfig,
			disableFF: true,
			errString: "use the otelcol.printInitialConfig feature gate",
		},
		{
			name: "field is set yaml",
			path: validConfig,
			outString: map[string]string{
				"redacted":   `timeout: 5s`,
				"unredacted": `timeout: 5s`,
			},
		},
		{
			name: "default field value",
			path: defaultConfig,
			outString: map[string]string{
				"redacted":   `timeout: 1s`,
				"unredacted": `timeout: 1s`,
			},
		},
		{
			name: "field is set json",
			ofmt: "json",
			path: validConfig,
			outString: map[string]string{
				// Note: JSON does not format as a time.Duration
				"redacted": `"timeout": 5000000000`,

				// Note: the original input is "5s"
				"unredacted": `"timeout": 5000000000`,
			},
		},
		{
			name: "opaque field",
			path: validConfig,
			outString: map[string]string{
				"redacted":   `opaque: '[REDACTED]'`,
				"unredacted": `opaque: OOO`,
			},
		},
		{
			name: "opaque default",
			path: defaultConfig,
			outString: map[string]string{
				"redacted":   `opaque: '[REDACTED]'`,
				"unredacted": `opaque: '[REDACTED]'`,
			},
		},
	}
	for _, test := range tests {
		testModes := []string{"redacted", "unredacted", "unrecognized"}
		for _, mode := range testModes {
			t.Run(fmt.Sprint(test.name, "_", mode), func(t *testing.T) {
				// Save current feature flag state and restore after test
				fg := featuregate.GlobalRegistry()

				fg.VisitAll(func(g *featuregate.Gate) {
					if g.ID() == metadata.OtelcolPrintInitialConfigFeatureGate.ID() {
						defer func() {
							_ = fg.Set(metadata.OtelcolPrintInitialConfigFeatureGate.ID(), g.IsEnabled())
						}()
					}
				})
				if test.disableFF {
					require.NoError(t, fg.Set(metadata.OtelcolPrintInitialConfigFeatureGate.ID(), false))
				} else {
					require.NoError(t, fg.Set(metadata.OtelcolPrintInitialConfigFeatureGate.ID(), true))
				}

				testR := component.MustNewType("r")
				testE := component.MustNewType("e")
				testReceiver := xreceiver.NewFactory(
					testR,
					func() component.Config {
						return printReceiverConfig{
							Opaque: "1234",
							Other:  "",
						}
					},
					xreceiver.WithLogs(func(context.Context, receiver.Settings, component.Config, consumer.Logs) (receiver.Logs, error) {
						return nil, nil
					}, component.StabilityLevelStable),
				)
				testExporter := xexporter.NewFactory(
					testE,
					func() component.Config {
						return printExporterConfig{
							Timeout: time.Second,
						}
					},
					xexporter.WithLogs(func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
						return nil, nil
					}, component.StabilityLevelStable),
				)
				var stdout bytes.Buffer

				set := confmap.ResolverSettings{}
				set.ProviderFactories = []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				}
				set.DefaultScheme = "file"
				set.URIs = []string{test.path}

				cmd := newConfigPrintSubCommand(CollectorSettings{
					Factories: func() (Factories, error) {
						return Factories{
							Receivers: map[component.Type]receiver.Factory{
								testR: testReceiver,
							},
							Exporters: map[component.Type]exporter.Factory{
								testE: testExporter,
							},
							Telemetry: telemetry.NewFactory(func() component.Config {
								return fakeTelemetryConfig{}
							}),
						}, nil
					},
					ConfigProviderSettings: ConfigProviderSettings{
						ResolverSettings: set,
					},
				}, flags(featuregate.GlobalRegistry()))
				cmd.SetOut(&stdout)
				args := []string{
					"--mode", mode,
					"--format", test.ofmt,
				}
				if test.validate {
					args = append(args, "--validate=true")
				} else {
					args = append(args, "--validate=false")
				}
				cmd.SetArgs(args)
				err := cmd.Execute()

				expectErr := test.errString != ""
				expectErrMsg := test.errString

				switch mode {
				case "redacted":
				case "unredacted":
					if test.errOnlyRedacted {
						expectErr = false
						expectErrMsg = ""
					}
				default:
					expectErr = true
					if test.disableFF {
						expectErrMsg = "feature gate"
					} else {
						expectErrMsg = "unrecognized"
					}
				}

				if expectErr {
					require.Error(t, err)
					require.ErrorContains(t, err, expectErrMsg)
				} else {
					require.NoError(t, err)
				}
				if test.outString[mode] != "" {
					require.Contains(t, stdout.String(), test.outString[mode])
				}
			})
		}
	}
}

func TestRestoreSecrets(t *testing.T) {
	tests := []struct {
		name     string
		base     any
		raw      any
		expected any
	}{
		{
			name:     "replace in map",
			base:     map[string]any{"key1": "[REDACTED]", "key2": "normal"},
			raw:      map[string]any{"key1": "secret1", "key2": "normal"},
			expected: map[string]any{"key1": "secret1", "key2": "normal"},
		},
		{
			name:     "replace in slice",
			base:     []any{"[REDACTED]", "normal"},
			raw:      []any{"secret_in_slice", "normal"},
			expected: []any{"secret_in_slice", "normal"},
		},
		{
			name: "nested structures",
			base: map[string]any{
				"nested_slice": []any{"[REDACTED]"},
				"nested_map":   map[string]any{"deep_key": "[REDACTED]"},
			},
			raw: map[string]any{
				"nested_slice": []any{"deep_secret1"},
				"nested_map":   map[string]any{"deep_key": "deep_secret2"},
			},
			expected: map[string]any{
				"nested_slice": []any{"deep_secret1"},
				"nested_map":   map[string]any{"deep_key": "deep_secret2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreSecrets(tt.base, tt.raw)
			require.Equal(t, tt.expected, tt.base)
		})
	}
}
