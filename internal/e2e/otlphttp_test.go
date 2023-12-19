// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestInvalidConfig(t *testing.T) {
	config := &otlphttpexporter.Config{
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: "",
		},
	}
	f := otlphttpexporter.NewFactory()
	set := exportertest.NewNopCreateSettings()
	_, err := f.CreateTracesExporter(context.Background(), set, config)
	require.Error(t, err)
	_, err = f.CreateMetricsExporter(context.Background(), set, config)
	require.Error(t, err)
	_, err = f.CreateLogsExporter(context.Background(), set, config)
	require.Error(t, err)
}

func TestTraceNoBackend(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exp := startTracesExporter(t, "", fmt.Sprintf("http://%s/v1/traces", addr))
	td := testdata.GenerateTraces(1)
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceInvalidUrl(t *testing.T) {
	exp := startTracesExporter(t, "http:/\\//this_is_an/*/invalid_url", "")
	td := testdata.GenerateTraces(1)
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))

	exp = startTracesExporter(t, "", "http:/\\//this_is_an/*/invalid_url")
	td = testdata.GenerateTraces(1)
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	startTracesReceiver(t, addr, consumertest.NewErr(errors.New("my_error")))
	exp := startTracesExporter(t, "", fmt.Sprintf("http://%s/v1/traces", addr))

	td := testdata.GenerateTraces(1)
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
			overrideURL: fmt.Sprintf("http://%s/v1/traces", addr),
		},
		{
			name:        "onlybase",
			baseURL:     fmt.Sprintf("http://%s", addr),
			overrideURL: "",
		},
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/traces", addr),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sink := new(consumertest.TracesSink)
			startTracesReceiver(t, addr, sink)
			exp := startTracesExporter(t, test.baseURL, test.overrideURL)

			td := testdata.GenerateTraces(1)
			assert.NoError(t, exp.ConsumeTraces(context.Background(), td))
			require.Eventually(t, func() bool {
				return sink.SpanCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allTraces := sink.AllTraces()
			require.Len(t, allTraces, 1)
			assert.EqualValues(t, td, allTraces[0])
		})
	}
}

func TestMetricsError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	startMetricsReceiver(t, addr, consumertest.NewErr(errors.New("my_error")))
	exp := startMetricsExporter(t, "", fmt.Sprintf("http://%s/v1/metrics", addr))

	md := testdata.GenerateMetrics(1)
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
			sink := new(consumertest.MetricsSink)
			startMetricsReceiver(t, addr, sink)
			exp := startMetricsExporter(t, test.baseURL, test.overrideURL)

			md := testdata.GenerateMetrics(1)
			assert.NoError(t, exp.ConsumeMetrics(context.Background(), md))
			require.Eventually(t, func() bool {
				return sink.DataPointCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allMetrics := sink.AllMetrics()
			require.Len(t, allMetrics, 1)
			assert.EqualValues(t, md, allMetrics[0])
		})
	}
}

func TestLogsError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	startLogsReceiver(t, addr, consumertest.NewErr(errors.New("my_error")))
	exp := startLogsExporter(t, "", fmt.Sprintf("http://%s/v1/logs", addr))

	md := testdata.GenerateLogs(1)
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
			sink := new(consumertest.LogsSink)
			startLogsReceiver(t, addr, sink)
			exp := startLogsExporter(t, test.baseURL, test.overrideURL)

			md := testdata.GenerateLogs(1)
			assert.NoError(t, exp.ConsumeLogs(context.Background(), md))
			require.Eventually(t, func() bool {
				return sink.LogRecordCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allLogs := sink.AllLogs()
			require.Len(t, allLogs, 1)
			assert.EqualValues(t, md, allLogs[0])
		})
	}
}

func TestIssue_4221(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { assert.NoError(t, r.Body.Close()) }()
		compressedData, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
		require.NoError(t, err)
		data, err := io.ReadAll(gzipReader)
		require.NoError(t, err)
		base64Data := base64.StdEncoding.EncodeToString(data)
		// Verify same base64 encoded string is received.
		assert.Equal(t, "CscBCkkKIAoMc2VydmljZS5uYW1lEhAKDnVvcC5zdGFnZS1ldS0xCiUKGW91dHN5c3RlbXMubW9kdWxlLnZlcnNpb24SCAoGOTAzMzg2EnoKEQoMdW9wX2NhbmFyaWVzEgExEmUKEEMDhT8Ib0+Mhs8Zi2VR34QSCOVRPDJ5XEG5IgA5QE41aASRrxZBQE41aASRrxZKEAoKc3Bhbl9pbmRleBICGANKHwoNY29kZS5mdW5jdGlvbhIOCgxteUZ1bmN0aW9uMzZ6AA==", base64Data)
		unbase64Data, err := base64.StdEncoding.DecodeString(base64Data)
		require.NoError(t, err)
		tr := ptraceotlp.NewExportRequest()
		require.NoError(t, tr.UnmarshalProto(unbase64Data))
		span := tr.Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
		traceID := span.TraceID()
		assert.Equal(t, "4303853f086f4f8c86cf198b6551df84", hex.EncodeToString(traceID[:]))
		spanID := span.SpanID()
		assert.Equal(t, "e5513c32795c41b9", hex.EncodeToString(spanID[:]))
	}))

	exp := startTracesExporter(t, "", svr.URL)

	md := ptrace.NewTraces()
	rms := md.ResourceSpans().AppendEmpty()
	rms.Resource().Attributes().PutStr("service.name", "uop.stage-eu-1")
	rms.Resource().Attributes().PutStr("outsystems.module.version", "903386")
	ils := rms.ScopeSpans().AppendEmpty()
	ils.Scope().SetName("uop_canaries")
	ils.Scope().SetVersion("1")
	span := ils.Spans().AppendEmpty()

	var traceIDBytes [16]byte
	traceIDBytesSlice, err := hex.DecodeString("4303853f086f4f8c86cf198b6551df84")
	require.NoError(t, err)
	copy(traceIDBytes[:], traceIDBytesSlice)
	span.SetTraceID(traceIDBytes)
	traceID := span.TraceID()
	assert.Equal(t, "4303853f086f4f8c86cf198b6551df84", hex.EncodeToString(traceID[:]))

	var spanIDBytes [8]byte
	spanIDBytesSlice, err := hex.DecodeString("e5513c32795c41b9")
	require.NoError(t, err)
	copy(spanIDBytes[:], spanIDBytesSlice)
	span.SetSpanID(spanIDBytes)
	spanID := span.SpanID()
	assert.Equal(t, "e5513c32795c41b9", hex.EncodeToString(spanID[:]))

	span.SetEndTimestamp(1634684637873000000)
	span.Attributes().PutInt("span_index", 3)
	span.Attributes().PutStr("code.function", "myFunction36")
	span.SetStartTimestamp(1634684637873000000)

	assert.NoError(t, exp.ConsumeTraces(context.Background(), md))
}

func startTracesExporter(t *testing.T, baseURL string, overrideURL string) exporter.Traces {
	factory := otlphttpexporter.NewFactory()
	cfg := createExporterConfig(baseURL, factory.CreateDefaultConfig())
	cfg.TracesEndpoint = overrideURL
	exp, err := factory.CreateTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func startMetricsExporter(t *testing.T, baseURL string, overrideURL string) exporter.Metrics {
	factory := otlphttpexporter.NewFactory()
	cfg := createExporterConfig(baseURL, factory.CreateDefaultConfig())
	cfg.MetricsEndpoint = overrideURL
	exp, err := factory.CreateMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func startLogsExporter(t *testing.T, baseURL string, overrideURL string) exporter.Logs {
	factory := otlphttpexporter.NewFactory()
	cfg := createExporterConfig(baseURL, factory.CreateDefaultConfig())
	cfg.LogsEndpoint = overrideURL
	exp, err := factory.CreateLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func createExporterConfig(baseURL string, defaultCfg component.Config) *otlphttpexporter.Config {
	cfg := defaultCfg.(*otlphttpexporter.Config)
	cfg.Endpoint = baseURL
	cfg.QueueConfig.Enabled = false
	cfg.RetryConfig.Enabled = false
	return cfg
}

func startTracesReceiver(t *testing.T, addr string, next consumer.Traces) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateTracesReceiver(context.Background(), receivertest.NewNopCreateSettings(), cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func startMetricsReceiver(t *testing.T, addr string, next consumer.Metrics) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateMetricsReceiver(context.Background(), receivertest.NewNopCreateSettings(), cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func startLogsReceiver(t *testing.T, addr string, next consumer.Logs) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateLogsReceiver(context.Background(), receivertest.NewNopCreateSettings(), cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func createReceiverConfig(addr string, defaultCfg component.Config) *otlpreceiver.Config {
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
