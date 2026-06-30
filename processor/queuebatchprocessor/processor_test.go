// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
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

func generateProfiles(numSamples int) pprofile.Profiles {
	pd := pprofile.NewProfiles()
	p := pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	for range numSamples {
		p.Samples().AppendEmpty()
	}
	return pd
}

func attrStr(set attribute.Set, key string) string {
	v, _ := set.Value(attribute.Key(key))
	return v.AsString()
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg, ok := createDefaultConfig().(*Config)
	require.True(t, ok)
	// The caller receives success once the request enters the queue;
	// wait_for_result is disabled by default (see README.md).
	require.False(t, cfg.WaitForResult)
	require.True(t, cfg.BlockOnOverflow)
	require.Equal(t, 1, cfg.NumConsumers)
	// queue_size defaults to 10 (not exporterhelper's 1000): the logical
	// equivalent of the legacy batchprocessor's num_cpus queueing.
	require.Equal(t, int64(10), cfg.QueueSize)
	// Batching must be enabled by default: it is the purpose of this component.
	require.True(t, cfg.Batch.HasValue(), "batching should be enabled by default")
	require.Positive(t, cfg.Batch.Get().MinSize)
	require.NoError(t, componenttest.CheckConfigStruct(cfg))
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

// assertHistogramPositive asserts a histogram recorded a single batch (count 1)
// with a positive sum. Used for the bytes histogram, whose exact encoded size
// is not asserted because it depends on the pdata wire encoding.
func assertHistogramPositive(t *testing.T, tt *componenttest.Telemetry, name, id, signal string) {
	t.Helper()
	m, err := tt.GetMetric(name)
	require.NoError(t, err)
	hist, ok := m.Data.(metricdata.Histogram[int64])
	require.True(t, ok, "expected histogram for %s", name)
	require.Len(t, hist.DataPoints, 1)
	dp := hist.DataPoints[0]
	require.Equal(t, uint64(1), dp.Count)
	require.Positive(t, dp.Sum)
	require.Equal(t, id, attrStr(dp.Attributes, processorKey))
	require.Equal(t, signal, attrStr(dp.Attributes, signalKey))
}

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

func TestProfilesMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	id, signal := set.ID.String(), xpipeline.SignalProfiles.String()
	sink := new(consumertest.ProfilesSink)
	p, err := newProfilesProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeProfiles(context.Background(), generateProfiles(6)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 6, sink.SampleCount())
	assertCounter(t, tt, "otelcol_processor_incoming_items", id, signal, 6)
	assertCounter(t, tt, "otelcol_processor_outgoing_items", id, signal, 6)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_items", id, signal, 6, true)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_bytes", id, signal, 0, false)
	assertNoExporterMetrics(t, tt)
}

// errDownstream is a sentinel returned by the downstream consumer to verify
// error propagation.
var errDownstream = errors.New("downstream failure")

// TestBatchingAccumulatesAcrossRequests verifies the core batching behavior:
// several separate inputs that individually stay below min_size are merged into
// a single batch, delivered downstream once the processor shuts down.
func TestBatchingAccumulatesAcrossRequests(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set := processortest.NewNopSettings(metadata.Type)
	set.TelemetrySettings = tt.NewTelemetrySettings()
	cfg := createDefaultConfig().(*Config)
	// A large min_size and flush_timeout keep the inputs accumulating rather
	// than flushing on their own. The batch sizer (items) differs from the
	// queue sizer (requests), so min_size is not bounded by queue_size.
	cfg.Batch.Get().MinSize = 1000
	cfg.Batch.Get().FlushTimeout = time.Minute

	id, signal := set.ID.String(), pipeline.SignalTraces.String()
	sink := new(consumertest.TracesSink)
	p, err := newTracesProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	for range 3 {
		require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(2)))
	}
	// Shutdown flushes the accumulated batch.
	require.NoError(t, p.Shutdown(context.Background()))

	// The three two-span inputs are merged and delivered as one batch of six.
	require.Equal(t, 6, sink.SpanCount())
	require.Len(t, sink.AllTraces(), 1)
	assertCounter(t, tt, "otelcol_processor_incoming_items", id, signal, 6)
	assertCounter(t, tt, "otelcol_processor_outgoing_items", id, signal, 6)
	assertHistogram(t, tt, "otelcol_processor_queuebatch_items", id, signal, 6, true)
	assertHistogramPositive(t, tt, "otelcol_processor_queuebatch_bytes", id, signal)
	assertNoExporterMetrics(t, tt)
}

// TestWaitForResultPropagatesError verifies that, with wait_for_result enabled,
// an error from the downstream consumer is propagated back to the caller.
func TestWaitForResultPropagatesError(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	cfg.WaitForResult = true

	p, err := newTracesProcessor(context.Background(), set, cfg, consumertest.NewErr(errDownstream))
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, p.Shutdown(context.Background())) })

	require.ErrorIs(t, p.ConsumeTraces(context.Background(), generateTraces(1)), errDownstream)
}

// TestDefaultDoesNotWaitForResult verifies the default (wait_for_result=false):
// the caller receives success as soon as the request enters the queue, even
// when the downstream consumer subsequently fails.
func TestDefaultDoesNotWaitForResult(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	require.False(t, cfg.WaitForResult, "wait_for_result must be disabled by default")

	p, err := newTracesProcessor(context.Background(), set, cfg, consumertest.NewErr(errDownstream))
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, p.Shutdown(context.Background())) })

	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(1)))
}

// errMeterFail is returned by errMeter to force telemetry construction to fail.
var errMeterFail = errors.New("meter creation failed")

type errMeterProvider struct{ noopmetric.MeterProvider }

func (errMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return errMeter{} }

type errMeter struct{ noopmetric.Meter }

func (errMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errMeterFail
}

// TestNewProcessorTelemetryError covers the error path in every signal's
// constructor when the telemetry builder cannot be created.
func TestNewProcessorTelemetryError(t *testing.T) {
	set := processortest.NewNopSettings(metadata.Type)
	set.MeterProvider = errMeterProvider{}
	cfg := createDefaultConfig().(*Config)
	ctx := context.Background()

	_, errT := newTracesProcessor(ctx, set, cfg, consumertest.NewNop())
	_, errM := newMetricsProcessor(ctx, set, cfg, consumertest.NewNop())
	_, errL := newLogsProcessor(ctx, set, cfg, consumertest.NewNop())
	_, errP := newProfilesProcessor(ctx, set, cfg, consumertest.NewNop())
	for _, err := range []error{errT, errM, errL, errP} {
		require.ErrorIs(t, err, errMeterFail)
	}
}

// TestNewProcessorExporterError covers the error path in every signal's
// constructor when the exporterhelper cannot be created. The telemetry builder
// only needs the meter, so a nil logger fails exporter construction while the
// telemetry succeeds.
func TestNewProcessorExporterError(t *testing.T) {
	set := processortest.NewNopSettings(metadata.Type)
	set.Logger = nil
	cfg := createDefaultConfig().(*Config)
	ctx := context.Background()

	_, errT := newTracesProcessor(ctx, set, cfg, consumertest.NewNop())
	_, errM := newMetricsProcessor(ctx, set, cfg, consumertest.NewNop())
	_, errL := newLogsProcessor(ctx, set, cfg, consumertest.NewNop())
	_, errP := newProfilesProcessor(ctx, set, cfg, consumertest.NewNop())
	for _, err := range []error{errT, errM, errL, errP} {
		require.Error(t, err)
	}
}

// TestInstrumentEnabledFallback covers the default-enabled path for an
// instrument that does not implement Enabled(context.Context).
func TestInstrumentEnabledFallback(t *testing.T) {
	require.True(t, instrumentEnabled(context.Background(), struct{}{}))
}
