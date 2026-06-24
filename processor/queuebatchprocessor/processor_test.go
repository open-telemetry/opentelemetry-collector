// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
)

// testSettings returns processor settings wired to tt's (real) meter and a
// config that flushes on every request so assertions are deterministic.
func testSettings(tt *componenttest.Telemetry) (processor.Settings, *Config) {
	set := processortest.NewNopSettings(metadata.Type)
	set.TelemetrySettings = tt.NewTelemetrySettings()
	cfg := createDefaultConfig().(*Config)
	cfg.Batch.Get().MinSize = 1
	return set, cfg
}

func generateTraces(numSpans int) ptrace.Traces {
	td := ptrace.NewTraces()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	for range numSpans {
		ss.Spans().AppendEmpty().SetName("span")
	}
	return td
}

func generateMetrics(numPoints int) pmetric.Metrics {
	md := pmetric.NewMetrics()
	ms := md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	for range numPoints {
		m := ms.Metrics().AppendEmpty()
		m.SetName("gauge")
		m.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(1)
	}
	return md
}

func generateLogs(numRecords int) plog.Logs {
	ld := plog.NewLogs()
	ls := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	for range numRecords {
		ls.LogRecords().AppendEmpty().Body().SetStr("log")
	}
	return ld
}

func attrStr(set attribute.Set, key string) string {
	v, _ := set.Value(attribute.Key(key))
	return v.AsString()
}

// assertCounter asserts a monotonic int counter has a single data point with
// the expected value and the processor/signal attributes.
func assertCounter(t *testing.T, tt *componenttest.Telemetry, name, id, signal string, value int64) {
	t.Helper()
	m, err := tt.GetMetric(name)
	require.NoError(t, err)
	sum, ok := m.Data.(metricdata.Sum[int64])
	require.True(t, ok, "expected sum for %s", name)
	require.Len(t, sum.DataPoints, 1)
	dp := sum.DataPoints[0]
	require.Equal(t, value, dp.Value)
	require.Equal(t, id, attrStr(dp.Attributes, processorKey))
	require.Equal(t, signal, attrStr(dp.Attributes, signalKey))
}

// assertHistogram asserts a histogram has a single batch (count 1) and,
// optionally, the expected sum.
func assertHistogram(t *testing.T, tt *componenttest.Telemetry, name, id, signal string, sum int64, checkSum bool) {
	t.Helper()
	m, err := tt.GetMetric(name)
	require.NoError(t, err)
	hist, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok, "expected histogram for %s", name)
	require.Len(t, hist.DataPoints, 1)
	dp := hist.DataPoints[0]
	require.Equal(t, uint64(1), dp.Count)
	if checkSum {
		require.Equal(t, sum, dp.Sum)
	}
	require.Equal(t, id, attrStr(dp.Attributes, processorKey))
	require.Equal(t, signal, attrStr(dp.Attributes, signalKey))
}

// assertNoExporterMetrics asserts the exporterhelper's own telemetry is
// disabled: no otelcol_exporter_* series are present.
func assertNoExporterMetrics(t *testing.T, tt *componenttest.Telemetry) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, tt.Reader.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			require.False(t, strings.HasPrefix(m.Name, "otelcol_exporter_"),
				"unexpected exporter metric leaked: %s", m.Name)
		}
	}
}

func TestTracesMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	id, signal := set.ID.String(), pipeline.SignalTraces.String()
	sink := new(consumertest.TracesSink)
	p, err := newTracesProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(5)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 5, sink.SpanCount())
	assertCounter(t, tt, "otelcol_processor_incoming_items", id, signal, 5)
	assertCounter(t, tt, "otelcol_processor_outgoing_items", id, signal, 5)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_items", id, signal, 5, true)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_bytes", id, signal, 0, false)
	assertNoExporterMetrics(t, tt)
}

func TestMetricsMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	id, signal := set.ID.String(), pipeline.SignalMetrics.String()
	sink := new(consumertest.MetricsSink)
	p, err := newMetricsProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeMetrics(context.Background(), generateMetrics(3)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 3, sink.DataPointCount())
	assertCounter(t, tt, "otelcol_processor_incoming_items", id, signal, 3)
	assertCounter(t, tt, "otelcol_processor_outgoing_items", id, signal, 3)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_items", id, signal, 3, true)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_bytes", id, signal, 0, false)
	assertNoExporterMetrics(t, tt)
}

func TestLogsMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	id, signal := set.ID.String(), pipeline.SignalLogs.String()
	sink := new(consumertest.LogsSink)
	p, err := newLogsProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeLogs(context.Background(), generateLogs(4)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 4, sink.LogRecordCount())
	assertCounter(t, tt, "otelcol_processor_incoming_items", id, signal, 4)
	assertCounter(t, tt, "otelcol_processor_outgoing_items", id, signal, 4)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_items", id, signal, 4, true)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_bytes", id, signal, 0, false)
	assertNoExporterMetrics(t, tt)
}
