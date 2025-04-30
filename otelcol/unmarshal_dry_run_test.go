// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/receiver"
)

var _ component.Config = (*Config)(nil)

type ValidateTestConfig struct {
	Number int    `mapstructure:"number"`
	String string `mapstructure:"string"`
}

var genericType component.Type = component.MustNewType("generic")

func NewFactories(_ *testing.T) func() (Factories, error) {
	return func() (Factories, error) {
		factories, err := nopFactories()
		if err != nil {
			return Factories{}, err
		}
		factories.Receivers[genericType] = receiver.NewFactory(
			genericType,
			func() component.Config {
				return &ValidateTestConfig{
					Number: 1,
					String: "default",
				}
			})

		return factories, nil
	}
}

var sampleYAMLConfig = `
receivers:
  generic:
    number: ${mock:number}
    string: ${mock:number}

exporters:
  nop:

service:
  pipelines:
    traces:
      receivers: [generic]
      exporters: [nop]
`

func TestDryRunWithExpandedValues(t *testing.T) {
	tests := []struct {
		name       string
		yamlConfig string
		mockMap    map[string]string
		expectErr  bool
	}{
		{
			name:       "string that looks like an integer",
			yamlConfig: sampleYAMLConfig,
			mockMap: map[string]string{
				"number": "123",
			},
			expectErr: true,
		},
		{
			name:       "string that looks like a bool",
			yamlConfig: sampleYAMLConfig,
			mockMap: map[string]string{
				"number": "true",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := NewCollector(CollectorSettings{
				Factories: NewFactories(t),
				ConfigProviderSettings: ConfigProviderSettings{
					ResolverSettings: confmap.ResolverSettings{
						URIs:          []string{"file:file"},
						DefaultScheme: "mock",
						ProviderFactories: []confmap.ProviderFactory{
							newFakeProvider("mock", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
								return confmap.NewRetrievedFromYAML([]byte(tt.mockMap[uri[len("mock:"):]]))
							}),
							newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
								return confmap.NewRetrievedFromYAML([]byte(tt.yamlConfig))
							}),
						},
					},
				},
				SkipSettingGRPCLogger: true,
			})
			require.NoError(t, err)

			err = collector.DryRun(context.Background())
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
