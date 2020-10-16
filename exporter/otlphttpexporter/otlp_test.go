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

package otlphttpexporter

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/testutil"
)

// TODO: add tests for retryable/permanent errors logic.
// TODO: add tests for throttling logic.

func TestInvalidConfig(t *testing.T) {
	config := &Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "",
		},
	}
	f := NewFactory()
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	_, err := f.CreateTraceExporter(context.Background(), params, config)
	require.Error(t, err)
	_, err = f.CreateMetricsExporter(context.Background(), params, config)
	require.Error(t, err)
	_, err = f.CreateLogsExporter(context.Background(), params, config)
	require.Error(t, err)
}

func TestTraceNoBackend(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exp := startTraceExporter(t, "", fmt.Sprintf("http://%s/v1/trace", addr))
	td := testdata.GenerateTraceDataOneSpan()
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceInvalidUrl(t *testing.T) {
	exp := startTraceExporter(t, "http:/\\//this_is_an/*/invalid_url", "")
	td := testdata.GenerateTraceDataOneSpan()
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))

	exp = startTraceExporter(t, "", "http:/\\//this_is_an/*/invalid_url")
	td = testdata.GenerateTraceDataOneSpan()
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	sink := new(exportertest.SinkTraceExporter)
	sink.SetConsumeTraceError(errors.New("my_error"))
	startTraceReceiver(t, addr, sink)
	exp := startTraceExporter(t, "", fmt.Sprintf("http://%s/v1/trace", addr))

	td := testdata.GenerateTraceDataOneSpan()
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceRoundTrip(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	tests := []struct {
		name        string
		baseURL     string
		overrideURL string
	}{
		{
			name:        "wrongbase",
			baseURL:     "http://wronghostname",
			overrideURL: fmt.Sprintf("http://%s/v1/trace", addr),
		},
		// TODO: open this test case after fixing this bug:
		// https://github.com/open-telemetry/opentelemetry-collector/issues/1968
		// {
		//	name:        "onlybase",
		//	baseURL:     fmt.Sprintf("http://%s", addr),
		//	overrideURL: "",
		// },
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/trace", addr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sink := new(exportertest.SinkTraceExporter)
			startTraceReceiver(t, addr, sink)
			exp := startTraceExporter(t, test.baseURL, test.overrideURL)

			td := testdata.GenerateTraceDataOneSpan()
			assert.NoError(t, exp.ConsumeTraces(context.Background(), td))
			require.Eventually(t, func() bool {
				return sink.SpansCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allTraces := sink.AllTraces()
			require.Len(t, allTraces, 1)
			assert.EqualValues(t, td, allTraces[0])
		})
	}
}

func TestMetricsError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	sink := new(exportertest.SinkMetricsExporter)
	sink.SetConsumeMetricsError(errors.New("my_error"))
	startMetricsReceiver(t, addr, sink)
	exp := startMetricsExporter(t, "", fmt.Sprintf("http://%s/v1/metrics", addr))

	md := testdata.GenerateMetricsOneMetric()
	assert.Error(t, exp.ConsumeMetrics(context.Background(), md))
}

func TestMetricsRoundTrip(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	tests := []struct {
		name        string
		baseURL     string
		overrideURL string
	}{
		{
			name:        "wrongbase",
			baseURL:     "http://wronghostname",
			overrideURL: fmt.Sprintf("http://%s/v1/metrics", addr),
		},
		{
			name:        "onlybase",
			baseURL:     fmt.Sprintf("http://%s", addr),
			overrideURL: "",
		},
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/metrics", addr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sink := new(exportertest.SinkMetricsExporter)
			startMetricsReceiver(t, addr, sink)
			exp := startMetricsExporter(t, test.baseURL, test.overrideURL)

			md := testdata.GenerateMetricsOneMetric()
			assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))
			require.Eventually(t, func() bool {
				return sink.MetricsCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allMetrics := sink.AllMetrics()
			require.Len(t, allMetrics, 1)
			assert.EqualValues(t, md, allMetrics[0])
		})
	}
}

func TestLogsError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	sink := new(exportertest.SinkLogsExporter)
	sink.SetConsumeLogError(errors.New("my_error"))
	startLogsReceiver(t, addr, sink)
	exp := startLogsExporter(t, "", fmt.Sprintf("http://%s/v1/logs", addr))

	md := testdata.GenerateLogDataOneLog()
	assert.Error(t, exp.ConsumeLogs(context.Background(), md))
}

func TestLogsRoundTrip(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	tests := []struct {
		name        string
		baseURL     string
		overrideURL string
	}{
		{
			name:        "wrongbase",
			baseURL:     "http://wronghostname",
			overrideURL: fmt.Sprintf("http://%s/v1/logs", addr),
		},
		{
			name:        "onlybase",
			baseURL:     fmt.Sprintf("http://%s", addr),
			overrideURL: "",
		},
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/logs", addr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sink := new(exportertest.SinkLogsExporter)
			startLogsReceiver(t, addr, sink)
			exp := startLogsExporter(t, test.baseURL, test.overrideURL)

			md := testdata.GenerateLogDataOneLog()
			assert.NoError(t, exp.ConsumeLogs(context.Background(), md))
			require.Eventually(t, func() bool {
				return sink.LogRecordsCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allLogs := sink.AllLogs()
			require.Len(t, allLogs, 1)
			assert.EqualValues(t, md, allLogs[0])
		})
	}
}

func startTraceExporter(t *testing.T, baseURL string, overrideURL string) component.TraceExporter {
	factory := NewFactory()
	cfg := createExporterConfig(baseURL, factory.CreateDefaultConfig())
	cfg.TracesEndpoint = overrideURL
	exp, err := factory.CreateTraceExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func startMetricsExporter(t *testing.T, baseURL string, overrideURL string) component.MetricsExporter {
	factory := NewFactory()
	cfg := createExporterConfig(baseURL, factory.CreateDefaultConfig())
	cfg.MetricsEndpoint = overrideURL
	exp, err := factory.CreateMetricsExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func startLogsExporter(t *testing.T, baseURL string, overrideURL string) component.LogsExporter {
	factory := NewFactory()
	cfg := createExporterConfig(baseURL, factory.CreateDefaultConfig())
	cfg.LogsEndpoint = overrideURL
	exp, err := factory.CreateLogsExporter(context.Background(), component.ExporterCreateParams{Logger: zap.NewNop()}, cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func createExporterConfig(baseURL string, defaultCfg configmodels.Exporter) *Config {
	cfg := defaultCfg.(*Config)
	cfg.Endpoint = baseURL
	cfg.QueueSettings.Enabled = false
	cfg.RetrySettings.Enabled = false
	return cfg
}

func startTraceReceiver(t *testing.T, addr string, next consumer.TraceConsumer) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{Logger: zap.NewNop()}, cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func startMetricsReceiver(t *testing.T, addr string, next consumer.MetricsConsumer) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{Logger: zap.NewNop()}, cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func startLogsReceiver(t *testing.T, addr string, next consumer.LogsConsumer) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{Logger: zap.NewNop()}, cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func createReceiverConfig(addr string, defaultCfg configmodels.Exporter) *otlpreceiver.Config {
	cfg := defaultCfg.(*otlpreceiver.Config)
	cfg.HTTP.Endpoint = addr
	cfg.GRPC = nil
	return cfg
}

func startAndCleanup(t *testing.T, cmp component.Component) {
	require.NoError(t, cmp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, cmp.Shutdown(context.Background()))
	})
}
