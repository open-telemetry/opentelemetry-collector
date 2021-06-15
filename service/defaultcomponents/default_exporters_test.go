// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package defaultcomponents

import (
	"context"
	"errors"
	"io/ioutil"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/exporter/fileexporter"
	"go.opentelemetry.io/collector/exporter/jaegerexporter"
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/exporter/prometheusexporter"
	"go.opentelemetry.io/collector/exporter/zipkinexporter"
	"go.opentelemetry.io/collector/testutil"
)

func TestDefaultExporters(t *testing.T) {
	factories, err := Components()
	assert.NoError(t, err)

	expFactories := factories.Exporters
	endpoint := testutil.GetAvailableLocalAddress(t)

	tests := []struct {
		exporter      config.Type
		getConfigFn   getExporterConfigFn
		skipLifecycle bool
	}{
		{
			exporter: "file",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["file"].CreateDefaultConfig().(*fileexporter.Config)
				f, err := ioutil.TempFile("", "otelcol_defaults_file_exporter_test*.tmp")
				require.NoError(t, err)
				assert.NoError(t, f.Close())
				cfg.Path = f.Name()
				return cfg
			},
		},
		{
			exporter: "jaeger",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["jaeger"].CreateDefaultConfig().(*jaegerexporter.Config)
				cfg.Endpoint = endpoint
				return cfg
			},
		},
		{
			exporter: "kafka",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["kafka"].CreateDefaultConfig().(*kafkaexporter.Config)
				cfg.Brokers = []string{"invalid:9092"}
				// this disables contacting the broker so we can successfully create the exporter
				cfg.Metadata.Full = false
				return cfg
			},
		},
		{
			exporter:      "logging",
			skipLifecycle: runtime.GOOS == "darwin", // TODO: investigate why this fails on darwin.
		},
		{
			exporter: "opencensus",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["opencensus"].CreateDefaultConfig().(*opencensusexporter.Config)
				cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
				}
				return cfg
			},
		},
		{
			exporter: "otlp",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["otlp"].CreateDefaultConfig().(*otlpexporter.Config)
				cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
				}
				return cfg
			},
		},
		{
			exporter: "otlphttp",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["otlphttp"].CreateDefaultConfig().(*otlphttpexporter.Config)
				cfg.Endpoint = "http://" + endpoint
				return cfg
			},
		},
		{
			exporter: "prometheus",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["prometheus"].CreateDefaultConfig().(*prometheusexporter.Config)
				cfg.Endpoint = endpoint
				return cfg
			},
		},
		{
			exporter: "prometheusremotewrite",
		},
		{
			exporter: "zipkin",
			getConfigFn: func() config.Exporter {
				cfg := expFactories["zipkin"].CreateDefaultConfig().(*zipkinexporter.Config)
				cfg.Endpoint = endpoint
				return cfg
			},
		},
	}

	assert.Equal(t, len(tests), len(expFactories))
	for _, tt := range tests {
		t.Run(string(tt.exporter), func(t *testing.T) {
			factory, ok := expFactories[tt.exporter]
			require.True(t, ok)
			assert.Equal(t, tt.exporter, factory.Type())
			assert.Equal(t, config.NewID(tt.exporter), factory.CreateDefaultConfig().ID())

			if tt.skipLifecycle {
				t.Log("Skipping lifecycle test", tt.exporter)
				return
			}

			verifyExporterLifecycle(t, factory, tt.getConfigFn)
		})
	}
}

// GetExporterConfigFn is used customize the configuration passed to the verification.
// This is used to change ports or provide values required but not provided by the
// default configuration.
type getExporterConfigFn func() config.Exporter

// verifyExporterLifecycle is used to test if an exporter type can handle the typical
// lifecycle of a component. The getConfigFn parameter only need to be specified if
// the test can't be done with the default configuration for the component.
func verifyExporterLifecycle(t *testing.T, factory component.ExporterFactory, getConfigFn getExporterConfigFn) {
	ctx := context.Background()
	host := newAssertNoErrorHost(t)
	expCreateSettings := componenttest.NewNopExporterCreateSettings()

	cfg := factory.CreateDefaultConfig()
	if getConfigFn != nil {
		cfg = getConfigFn()
	}

	createFns := []createExporterFn{
		wrapCreateLogsExp(factory),
		wrapCreateTracesExp(factory),
		wrapCreateMetricsExp(factory),
	}

	for i := 0; i < 2; i++ {
		var exps []component.Exporter
		for _, createFn := range createFns {
			exp, err := createFn(ctx, expCreateSettings, cfg)
			if errors.Is(err, componenterror.ErrDataTypeIsNotSupported) {
				continue
			}
			require.NoError(t, err)
			require.NoError(t, exp.Start(ctx, host))
			exps = append(exps, exp)
		}
		for _, exp := range exps {
			assert.NoError(t, exp.Shutdown(ctx))
		}
	}
}

type createExporterFn func(
	ctx context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.Exporter, error)

func wrapCreateLogsExp(factory component.ExporterFactory) createExporterFn {
	return func(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.Exporter, error) {
		return factory.CreateLogsExporter(ctx, set, cfg)
	}
}

func wrapCreateTracesExp(factory component.ExporterFactory) createExporterFn {
	return func(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.Exporter, error) {
		return factory.CreateTracesExporter(ctx, set, cfg)
	}
}

func wrapCreateMetricsExp(factory component.ExporterFactory) createExporterFn {
	return func(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.Exporter, error) {
		return factory.CreateMetricsExporter(ctx, set, cfg)
	}
}
