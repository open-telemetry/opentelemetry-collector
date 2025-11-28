// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gootlpcollectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	gootlpcommon "go.opentelemetry.io/proto/otlp/common/v1"
	gootlpresource "go.opentelemetry.io/proto/otlp/resource/v1"
	gootlptrace "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestInvalidConfig(t *testing.T) {
	config := &otlphttpexporter.Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "",
		},
	}
	f := otlphttpexporter.NewFactory()
	set := exportertest.NewNopSettings(f.Type())
	_, err := f.CreateTraces(context.Background(), set, config)
	require.Error(t, err)
	_, err = f.CreateMetrics(context.Background(), set, config)
	require.Error(t, err)
	_, err = f.CreateLogs(context.Background(), set, config)
	require.Error(t, err)
}

func TestTraceNoBackend(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)
	exp := startTraces(t, "", fmt.Sprintf("http://%s/v1/traces", addr))
	td := testdata.GenerateTraces(1)
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceInvalidUrl(t *testing.T) {
	exp := startTraces(t, "http:/\\//this_is_an/*/invalid_url", "")
	td := testdata.GenerateTraces(1)
	require.Error(t, exp.ConsumeTraces(context.Background(), td))

	exp = startTraces(t, "", "http:/\\//this_is_an/*/invalid_url")
	td = testdata.GenerateTraces(1)
	assert.Error(t, exp.ConsumeTraces(context.Background(), td))
}

func TestTraceError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	startTracesReceiver(t, addr, consumertest.NewErr(errors.New("my_error")))
	exp := startTraces(t, "", fmt.Sprintf("http://%s/v1/traces", addr))

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
			baseURL:     "http://" + addr,
			overrideURL: "",
		},
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/traces", addr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.TracesSink)
			startTracesReceiver(t, addr, sink)
			exp := startTraces(t, tt.baseURL, tt.overrideURL)

			td := testdata.GenerateTraces(1)
			require.NoError(t, exp.ConsumeTraces(context.Background(), td))
			require.Eventually(t, func() bool {
				return sink.SpanCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allTraces := sink.AllTraces()
			require.Len(t, allTraces, 1)
			assert.Equal(t, td, allTraces[0])
		})
	}
}

func TestMetricsError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	startMetricsReceiver(t, addr, consumertest.NewErr(errors.New("my_error")))
	exp := startMetrics(t, "", fmt.Sprintf("http://%s/v1/metrics", addr))

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
			baseURL:     "http://" + addr,
			overrideURL: "",
		},
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/metrics", addr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.MetricsSink)
			startMetricsReceiver(t, addr, sink)
			exp := startMetrics(t, tt.baseURL, tt.overrideURL)

			md := testdata.GenerateMetrics(1)
			require.NoError(t, exp.ConsumeMetrics(context.Background(), md))
			require.Eventually(t, func() bool {
				return sink.DataPointCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allMetrics := sink.AllMetrics()
			require.Len(t, allMetrics, 1)
			assert.Equal(t, md, allMetrics[0])
		})
	}
}

func TestLogsError(t *testing.T) {
	addr := testutil.GetAvailableLocalAddress(t)

	startLogsReceiver(t, addr, consumertest.NewErr(errors.New("my_error")))
	exp := startLogs(t, "", fmt.Sprintf("http://%s/v1/logs", addr))

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
			baseURL:     "http://" + addr,
			overrideURL: "",
		},
		{
			name:        "override",
			baseURL:     "",
			overrideURL: fmt.Sprintf("http://%s/v1/logs", addr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.LogsSink)
			startLogsReceiver(t, addr, sink)
			exp := startLogs(t, tt.baseURL, tt.overrideURL)

			md := testdata.GenerateLogs(1)
			require.NoError(t, exp.ConsumeLogs(context.Background(), md))
			require.Eventually(t, func() bool {
				return sink.LogRecordCount() > 0
			}, 1*time.Second, 10*time.Millisecond)
			allLogs := sink.AllLogs()
			require.Len(t, allLogs, 1)
			assert.Equal(t, md, allLogs[0])
		})
	}
}

func TestIssue_4221(t *testing.T) {
	traceIDBytesSlice, err := hex.DecodeString("4303853f086f4f8c86cf198b6551df84")
	require.NoError(t, err)
	spanIDBytesSlice, err := hex.DecodeString("e5513c32795c41b9")
	require.NoError(t, err)

	svr := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		defer func() { assert.NoError(t, r.Body.Close()) }()
		compressedData, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
		assert.NoError(t, err)
		data, err := io.ReadAll(gzipReader)
		assert.NoError(t, err)
		// Verify same base64 encoded string is received.
		req := &gootlpcollectortrace.ExportTraceServiceRequest{}
		assert.NoError(t, proto.Unmarshal(data, req))

		assert.Empty(t, cmp.Diff(&gootlpcollectortrace.ExportTraceServiceRequest{
			ResourceSpans: []*gootlptrace.ResourceSpans{
				{
					Resource: &gootlpresource.Resource{
						Attributes: []*gootlpcommon.KeyValue{
							{
								Key: "service.name",
								Value: &gootlpcommon.AnyValue{
									Value: &gootlpcommon.AnyValue_StringValue{
										StringValue: "uop.stage-eu-1",
									},
								},
							},
							{
								Key: "outsystems.module.version",
								Value: &gootlpcommon.AnyValue{
									Value: &gootlpcommon.AnyValue_StringValue{
										StringValue: "903386",
									},
								},
							},
						},
					},
					ScopeSpans: []*gootlptrace.ScopeSpans{
						{
							Scope: &gootlpcommon.InstrumentationScope{
								Name:    "uop_canaries",
								Version: "1",
							},
							Spans: []*gootlptrace.Span{
								{
									TraceId:           traceIDBytesSlice,
									SpanId:            spanIDBytesSlice,
									StartTimeUnixNano: 1634684637873000000,
									EndTimeUnixNano:   1634684637873000000,
									Attributes: []*gootlpcommon.KeyValue{
										{
											Key: "span_index",
											Value: &gootlpcommon.AnyValue{
												Value: &gootlpcommon.AnyValue_IntValue{
													IntValue: 3,
												},
											},
										},
										{
											Key: "code.function",
											Value: &gootlpcommon.AnyValue{
												Value: &gootlpcommon.AnyValue_StringValue{
													StringValue: "myFunction36",
												},
											},
										},
									},
									Status: &gootlptrace.Status{},
								},
							},
						},
					},
				},
			},
		}, req, protocmp.Transform()))

		assert.NoError(t, err)
		tr := ptraceotlp.NewExportRequest()
		assert.NoError(t, tr.UnmarshalProto(data))
		span := tr.Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0)
		traceID := span.TraceID()
		assert.Equal(t, "4303853f086f4f8c86cf198b6551df84", hex.EncodeToString(traceID[:]))
		spanID := span.SpanID()
		assert.Equal(t, "e5513c32795c41b9", hex.EncodeToString(spanID[:]))
	}))
	defer func() { svr.Close() }()

	exp := startTraces(t, "", svr.URL)

	md := ptrace.NewTraces()
	rms := md.ResourceSpans().AppendEmpty()
	rms.Resource().Attributes().PutStr("service.name", "uop.stage-eu-1")
	rms.Resource().Attributes().PutStr("outsystems.module.version", "903386")
	ils := rms.ScopeSpans().AppendEmpty()
	ils.Scope().SetName("uop_canaries")
	ils.Scope().SetVersion("1")
	span := ils.Spans().AppendEmpty()

	var traceIDBytes [16]byte
	copy(traceIDBytes[:], traceIDBytesSlice)
	span.SetTraceID(traceIDBytes)
	traceID := span.TraceID()
	assert.Equal(t, "4303853f086f4f8c86cf198b6551df84", hex.EncodeToString(traceID[:]))

	var spanIDBytes [8]byte
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

func startTraces(t *testing.T, baseURL, overrideURL string) exporter.Traces {
	factory := otlphttpexporter.NewFactory()
	cfg := createConfig(baseURL, factory.CreateDefaultConfig())
	cfg.TracesEndpoint = overrideURL
	exp, err := factory.CreateTraces(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func startMetrics(t *testing.T, baseURL, overrideURL string) exporter.Metrics {
	factory := otlphttpexporter.NewFactory()
	cfg := createConfig(baseURL, factory.CreateDefaultConfig())
	cfg.MetricsEndpoint = overrideURL
	exp, err := factory.CreateMetrics(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func startLogs(t *testing.T, baseURL, overrideURL string) exporter.Logs {
	factory := otlphttpexporter.NewFactory()
	cfg := createConfig(baseURL, factory.CreateDefaultConfig())
	cfg.LogsEndpoint = overrideURL
	exp, err := factory.CreateLogs(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	startAndCleanup(t, exp)
	return exp
}

func createConfig(baseURL string, defaultCfg component.Config) *otlphttpexporter.Config {
	cfg := defaultCfg.(*otlphttpexporter.Config)
	cfg.ClientConfig.Endpoint = baseURL
	cfg.QueueConfig = configoptional.None[exporterhelper.QueueBatchConfig]()
	cfg.RetryConfig.Enabled = false
	return cfg
}

func startTracesReceiver(t *testing.T, addr string, next consumer.Traces) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateTraces(context.Background(), receivertest.NewNopSettings(factory.Type()), cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func startMetricsReceiver(t *testing.T, addr string, next consumer.Metrics) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateMetrics(context.Background(), receivertest.NewNopSettings(factory.Type()), cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func startLogsReceiver(t *testing.T, addr string, next consumer.Logs) {
	factory := otlpreceiver.NewFactory()
	cfg := createReceiverConfig(addr, factory.CreateDefaultConfig())
	recv, err := factory.CreateLogs(context.Background(), receivertest.NewNopSettings(factory.Type()), cfg, next)
	require.NoError(t, err)
	startAndCleanup(t, recv)
}

func createReceiverConfig(addr string, defaultCfg component.Config) *otlpreceiver.Config {
	cfg := defaultCfg.(*otlpreceiver.Config)
	cfg.HTTP.GetOrInsertDefault().ServerConfig.Endpoint = addr
	return cfg
}

func startAndCleanup(t *testing.T, cmp component.Component) {
	require.NoError(t, cmp.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, cmp.Shutdown(context.Background()))
	})
}
