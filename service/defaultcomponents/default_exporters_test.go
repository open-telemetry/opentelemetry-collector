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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configmodels"
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
		exporter      configmodels.Type
		getConfigFn   getExporterConfigFn
		skipLifecycle bool
	}{
		{
			exporter: "file",
			getConfigFn: func() configmodels.Exporter {
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
			getConfigFn: func() configmodels.Exporter {
				cfg := expFactories["jaeger"].CreateDefaultConfig().(*jaegerexporter.Config)
				cfg.Endpoint = endpoint
				return cfg
			},
		},
		{
			exporter: "kafka",
			getConfigFn: func() configmodels.Exporter {
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
			getConfigFn: func() configmodels.Exporter {
				cfg := expFactories["opencensus"].CreateDefaultConfig().(*opencensusexporter.Config)
				cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
				}
				return cfg
			},
		},
		{
			exporter: "otlp",
			getConfigFn: func() configmodels.Exporter {
				cfg := expFactories["otlp"].CreateDefaultConfig().(*otlpexporter.Config)
				cfg.GRPCClientSettings = configgrpc.GRPCClientSettings{
					Endpoint: endpoint,
				}
				return cfg
			},
		},
		{
			exporter: "otlphttp",
			getConfigFn: func() configmodels.Exporter {
				cfg := expFactories["otlphttp"].CreateDefaultConfig().(*otlphttpexporter.Config)
				cfg.Endpoint = "http://" + endpoint
				return cfg
			},
		},
		{
			exporter: "prometheus",
			getConfigFn: func() configmodels.Exporter {
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
			getConfigFn: func() configmodels.Exporter {
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
			assert.Equal(t, tt.exporter, factory.CreateDefaultConfig().Type())

			if tt.skipLifecycle {
				verifyExporterLifecycle(t, factory, tt.getConfigFn)
			}
		})
	}
}

// GetExporterConfigFn is used customize the configuration passed to the verification.
// This is used to change ports or provide values required but not provided by the
// default configuration.
type getExporterConfigFn func() configmodels.Exporter

// verifyExporterLifecycle is used to test if an exporter type can handle the typical
// lifecycle of a component. The getConfigFn parameter only need to be specified if
// the test can't be done with the default configuration for the component.
func verifyExporterLifecycle(t *testing.T, factory component.ExporterFactory, getConfigFn getExporterConfigFn) {
	ctx := context.Background()
	host := newAssertNoErrorHost(t)
	expCreateParams := component.ExporterCreateParams{
		Logger:               zap.NewNop(),
		ApplicationStartInfo: component.DefaultApplicationStartInfo(),
	}

	if getConfigFn == nil {
		getConfigFn = factory.CreateDefaultConfig
	}

	createFns := []createExporterFn{
		wrapCreateLogsExp(factory),
		wrapCreateTracesExp(factory),
		wrapCreateMetricsExp(factory),
	}

	for _, createFn := range createFns {
		firstExp, err := createFn(ctx, expCreateParams, getConfigFn())
		if errors.Is(err, configerror.ErrDataTypeIsNotSupported) {
			continue
		}
		require.NoError(t, err)
		require.NoError(t, firstExp.Start(ctx, host))
		require.NoError(t, firstExp.Shutdown(ctx))

		secondExp, err := createFn(ctx, expCreateParams, getConfigFn())
		require.NoError(t, err)
		require.NoError(t, secondExp.Start(ctx, host))
		require.NoError(t, secondExp.Shutdown(ctx))
	}
}

type createExporterFn func(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.Exporter, error)

func wrapCreateLogsExp(factory component.ExporterFactory) createExporterFn {
	return func(ctx context.Context, params component.ExporterCreateParams, cfg configmodels.Exporter) (component.Exporter, error) {
		return factory.CreateLogsExporter(ctx, params, cfg)
	}
}

func wrapCreateTracesExp(factory component.ExporterFactory) createExporterFn {
	return func(ctx context.Context, params component.ExporterCreateParams, cfg configmodels.Exporter) (component.Exporter, error) {
		return factory.CreateTracesExporter(ctx, params, cfg)
	}
}

func wrapCreateMetricsExp(factory component.ExporterFactory) createExporterFn {
	return func(ctx context.Context, params component.ExporterCreateParams, cfg configmodels.Exporter) (component.Exporter, error) {
		return factory.CreateMetricsExporter(ctx, params, cfg)
	}
}
