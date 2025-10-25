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
			name:            "invalid syntax without validate",
			path:            invalidConfig1,
			errString:       "'timeout' time: invalid duration",
			errOnlyRedacted: true,
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
				"redacted": `timeout: 1s`,

				// Since the structure is empty before
				// interpretation, no default is expanded.
				"unredacted": `e: null`,
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
				"unredacted": `"timeout": "5s"`,
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
				"redacted": `opaque: '[REDACTED]'`,

				// Note: the default opaque value does not print,
				// the other value is set in defaultConfig so that
				// the whole component config is not defaulted.
				"unredacted": `other: lala`,
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
					if g.ID() == featureGateName {
						defer func() {
							_ = fg.Set(featureGateName, g.IsEnabled())
						}()
					}
				})
				if test.disableFF {
					require.NoError(t, fg.Set(featureGateName, false))
				} else {
					require.NoError(t, fg.Set(featureGateName, true))
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
