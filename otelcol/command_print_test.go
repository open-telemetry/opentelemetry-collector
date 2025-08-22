// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"bytes"
	"context"
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
)

func TestPrintCommand(t *testing.T) {
	const nonexistentConfig = "file:nope.yaml"

	invalidConfig := fmt.Sprint("file:", filepath.Join("testdata", "print_invalid.yaml"))
	validConfig := fmt.Sprint("file:", filepath.Join("testdata", "print.yaml"))
	defaultConfig := fmt.Sprint("file:", filepath.Join("testdata", "print_default.yaml"))

	tests := []struct {
		name      string
		ofmt      string
		path      string
		disableFF bool // disable the feature flag
		errString string
		outString map[string]string
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
			name:      "invalid field",
			path:      invalidConfig,
			errString: "'timeout' time: invalid duration",
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
		testModes := []string{"redacted", "unredacted"}
		for _, mode := range testModes {
			t.Run(fmt.Sprint(test.name, "_", mode), func(t *testing.T) {
				// Save current feature flag state and restore after test
				fg := featuregate.GlobalRegistry()

				fg.VisitAll(func(g *featuregate.Gate) {
					if g.ID() == featureGateName {
						defer fg.Set(featureGateName, g.IsEnabled())
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
						return struct {
							Opaque configopaque.String `mapstructure:"opaque"`
							Other  string              `mapstructure:"other,omitempty"`
						}{
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
						return struct {
							Timeout time.Duration `mapstructure:"timeout"`
						}{
							Timeout: time.Second,
						}
					},
					xexporter.WithLogs(func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
						return nil, nil
					}, component.StabilityLevelStable),
				)
				var stdout, stderr bytes.Buffer

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
						}, nil
					},
					ConfigProviderSettings: ConfigProviderSettings{
						ResolverSettings: set,
					},
				}, flags(featuregate.GlobalRegistry()), &stdout, &stderr)
				args := []string{
					"--mode", mode,
					"--format", test.ofmt,
				}
				if mode == "unredacted" {
					args = append(args, "--validate=true")
				} else {
					args = append(args, "--validate=false")
				}
				cmd.SetArgs(args)
				err := cmd.Execute()

				if test.errString != "" {
					require.ErrorContains(t, err, test.errString)
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
