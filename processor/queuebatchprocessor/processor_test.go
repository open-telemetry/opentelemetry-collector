// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatchprocessor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadata"
	"go.opentelemetry.io/collector/processor/queuebatchprocessor/internal/metadatatest"
)

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

func TestCreateDefaultConfig(t *testing.T) {
	cfg, ok := createDefaultConfig().(*Config)
	require.True(t, ok)
	require.False(t, cfg.WaitForResult)
	require.True(t, cfg.BlockOnOverflow)
	require.Equal(t, 1, cfg.NumConsumers)
	require.Equal(t, int64(10), cfg.QueueSize)
	require.True(t, cfg.Batch.HasValue(), "batching should be enabled by default")
	require.Positive(t, cfg.Batch.Get().MinSize)
	require.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestTraces(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	sink := new(consumertest.TracesSink)
	p, err := newTracesProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(5)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 5, sink.SpanCount())
}

func processorAttrs(set processor.Settings, signal pipeline.Signal) attribute.Set {
	return attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, signal.String()),
	)
}

func processorQueueAttrs(set processor.Settings, signal pipeline.Signal, sizer exporterhelper.RequestSizerType) attribute.Set {
	return attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, signal.String()),
		attribute.String(sizerKey, sizer.String()),
	)
}

// Every instrument in metadata.yaml is reported and no exporter instrument leaks through.
func TestTracesProcessorMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	cfg.WaitForResult = true
	p, err := newTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeTraces(context.Background(), generateTraces(5)))

	attrs := processorAttrs(set, pipeline.SignalTraces)
	queueAttrs := processorQueueAttrs(set, pipeline.SignalTraces, cfg.Sizer)

	metadatatest.AssertEqualProcessorQueuebatchQueueCapacity(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 10, Attributes: queueAttrs}},
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualProcessorQueuebatchQueueSize(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 0, Attributes: queueAttrs}},
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualProcessorQueuebatchInFlightRequests(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 0, Attributes: attrs}},
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
	metadatatest.AssertEqualProcessorQueuebatchSentItems(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 5, Attributes: attrs}},
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	enqueueMetric, err := tt.GetMetric("otelcol_processor_queuebatch_enqueue_size")
	require.NoError(t, err)
	enqueue := enqueueMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, uint64(1), enqueue.DataPoints[0].Count)
	require.Equal(t, int64(5), enqueue.DataPoints[0].Sum)
	require.Equal(t, attrs, enqueue.DataPoints[0].Attributes)

	enqueueBytesMetric, err := tt.GetMetric("otelcol_processor_queuebatch_enqueue_size_bytes")
	require.NoError(t, err)
	enqueueBytes := enqueueBytesMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, uint64(1), enqueueBytes.DataPoints[0].Count)
	require.Positive(t, enqueueBytes.DataPoints[0].Sum)
	require.Equal(t, attrs, enqueueBytes.DataPoints[0].Attributes)

	batchMetric, err := tt.GetMetric("otelcol_processor_queuebatch_batch_send_size")
	require.NoError(t, err)
	batch := batchMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, uint64(1), batch.DataPoints[0].Count)
	require.Equal(t, int64(5), batch.DataPoints[0].Sum)
	require.Equal(t, attrs, batch.DataPoints[0].Attributes)

	bytesMetric, err := tt.GetMetric("otelcol_processor_queuebatch_batch_send_size_bytes")
	require.NoError(t, err)
	bytes := bytesMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, uint64(1), bytes.DataPoints[0].Count)
	require.Positive(t, bytes.DataPoints[0].Sum)
	require.Equal(t, attrs, bytes.DataPoints[0].Attributes)

	require.NoError(t, p.Shutdown(context.Background()))

	for _, name := range []string{
		"otelcol_exporter_sent_spans",
		"otelcol_exporter_send_failed_spans",
		"otelcol_exporter_enqueue_failed_spans",
		"otelcol_exporter_enqueue_size",
		"otelcol_exporter_enqueue_size_bytes",
		"otelcol_exporter_queue_batch_send_size",
		"otelcol_exporter_queue_batch_send_size_bytes",
		"otelcol_exporter_queue_size",
		"otelcol_exporter_queue_capacity",
		"otelcol_exporter_in_flight_requests",
	} {
		_, err = tt.GetMetric(name)
		require.Error(t, err, "exporter instrument %q must not be reported", name)
	}
}

// The failure path routes exporterhelper's error attributes onto processor instruments.
func TestTracesProcessorSendFailureMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	cfg.WaitForResult = true
	consumeErr := errors.New("consume failed")
	p, err := newTracesProcessor(context.Background(), set, cfg, consumertest.NewErr(consumeErr))
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.ErrorIs(t, p.ConsumeTraces(context.Background(), generateTraces(5)), consumeErr)
	require.NoError(t, p.Shutdown(context.Background()))

	failed, err := tt.GetMetric("otelcol_processor_queuebatch_send_failed_items")
	require.NoError(t, err)
	sum := failed.Data.(metricdata.Sum[int64])
	require.Equal(t, int64(5), sum.DataPoints[0].Value)
	require.Equal(t, attribute.NewSet(
		attribute.String(processorKey, set.ID.String()),
		attribute.String(dataTypeKey, pipeline.SignalTraces.String()),
		attribute.String("error.type", "_OTHER"),
		attribute.Bool("error.permanent", false),
	), sum.DataPoints[0].Attributes)
}

// The drop path is only reachable when the queue rejects a request.
func TestTracesProcessorEnqueueFailureMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	cfg.BlockOnOverflow = false
	cfg.QueueSize = 1
	cfg.Batch.Get().MinSize = 1000
	p, err := newTracesProcessor(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	// Not started, so nothing drains the queue and it overflows.
	var lastErr error
	for range 100 {
		if lastErr = p.ConsumeTraces(context.Background(), generateTraces(5)); lastErr != nil {
			break
		}
	}
	require.Error(t, lastErr)
	require.NoError(t, p.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorQueuebatchEnqueueFailedItems(t, tt,
		[]metricdata.DataPoint[int64]{{Value: 5, Attributes: processorAttrs(set, pipeline.SignalTraces)}},
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

	enqueueMetric, err := tt.GetMetric("otelcol_processor_queuebatch_enqueue_size")
	require.NoError(t, err)
	enqueue := enqueueMetric.Data.(metricdata.Histogram[int64])
	require.Equal(t, uint64(2), enqueue.DataPoints[0].Count)
	require.Equal(t, int64(10), enqueue.DataPoints[0].Sum)
}

func TestObsMetricsQueueSizerAttribute(t *testing.T) {
	for _, sizer := range []exporterhelper.RequestSizerType{
		exporterhelper.RequestSizerTypeRequests,
		exporterhelper.RequestSizerTypeItems,
		exporterhelper.RequestSizerTypeBytes,
	} {
		t.Run(sizer.String(), func(t *testing.T) {
			tt := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			set := processortest.NewNopSettings(metadata.Type)
			om, err := newObsMetrics(tt.NewTelemetrySettings(), set.ID, pipeline.SignalTraces, sizer)
			require.NoError(t, err)
			t.Cleanup(om.Shutdown)

			require.NoError(t, om.RegisterQueueSize(func() int64 { return 7 }))
			require.NoError(t, om.RegisterQueueCapacity(func() int64 { return 9 }))
			attrs := processorQueueAttrs(set, pipeline.SignalTraces, sizer)

			metadatatest.AssertEqualProcessorQueuebatchQueueSize(t, tt,
				[]metricdata.DataPoint[int64]{{Value: 7, Attributes: attrs}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
			metadatatest.AssertEqualProcessorQueuebatchQueueCapacity(t, tt,
				[]metricdata.DataPoint[int64]{{Value: 9, Attributes: attrs}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		})
	}
}

// Each signal reports through the same instruments, distinguished by the signal attribute.
func TestObsMetricsSignalAttribute(t *testing.T) {
	for _, signal := range []pipeline.Signal{
		pipeline.SignalTraces,
		pipeline.SignalMetrics,
		pipeline.SignalLogs,
		xpipeline.SignalProfiles,
	} {
		t.Run(signal.String(), func(t *testing.T) {
			tt := componenttest.NewTelemetry()
			t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

			set := processortest.NewNopSettings(metadata.Type)
			om, err := newObsMetrics(tt.NewTelemetrySettings(), set.ID, signal, exporterhelper.RequestSizerTypeRequests)
			require.NoError(t, err)
			t.Cleanup(om.Shutdown)

			attrs := processorAttrs(set, signal)
			om.RecordSent(context.Background(), 7)
			om.RecordEnqueueFailure(context.Background(), 3)
			om.RecordInFlight(context.Background(), 1)

			metadatatest.AssertEqualProcessorQueuebatchSentItems(t, tt,
				[]metricdata.DataPoint[int64]{{Value: 7, Attributes: attrs}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
			metadatatest.AssertEqualProcessorQueuebatchEnqueueFailedItems(t, tt,
				[]metricdata.DataPoint[int64]{{Value: 3, Attributes: attrs}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
			metadatatest.AssertEqualProcessorQueuebatchInFlightRequests(t, tt,
				[]metricdata.DataPoint[int64]{{Value: 1, Attributes: attrs}},
				metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())
		})
	}
}

// A failed construction releases the telemetry, unregistering the asynchronous callbacks.
func TestObsMetricsShutdownOnConstructionFailure(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, _ := testSettings(tt)
	_, err := newSignalProcessor(set, pipeline.SignalTraces, exporterhelper.RequestSizerTypeRequests,
		func(om exporterhelper.ObsMetrics) (processor.Traces, error) {
			require.NoError(t, om.RegisterQueueSize(func() int64 { return 7 }))
			require.NoError(t, om.RegisterQueueCapacity(func() int64 { return 9 }))
			return nil, errors.New("construction failed")
		})
	require.Error(t, err)

	_, err = tt.GetMetric("otelcol_processor_queuebatch_queue_size")
	require.Error(t, err, "queue size callback must be unregistered")
	_, err = tt.GetMetric("otelcol_processor_queuebatch_queue_capacity")
	require.Error(t, err, "queue capacity callback must be unregistered")
}

func TestMetrics(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	sink := new(consumertest.MetricsSink)
	p, err := newMetricsProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeMetrics(context.Background(), generateMetrics(3)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 3, sink.DataPointCount())
}

func TestLogs(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	sink := new(consumertest.LogsSink)
	p, err := newLogsProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeLogs(context.Background(), generateLogs(4)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 4, sink.LogRecordCount())
}

func TestProfiles(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set, cfg := testSettings(tt)
	sink := new(consumertest.ProfilesSink)
	p, err := newProfilesProcessor(context.Background(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, p.ConsumeProfiles(context.Background(), generateProfiles(6)))
	require.NoError(t, p.Shutdown(context.Background()))

	require.Equal(t, 6, sink.SampleCount())
}

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

// TestDefaultDoesNotWaitForResult verifies the default wait_for_result=false:
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

// capabilityCases enumerates the MutatesData capability the processor must exhibit.
var capabilityCases = []struct {
	name         string
	batchEnabled bool
	storage      bool
	nextMutates  bool
	want         bool
}{
	{"batch_enabled_next_readonly", true, false, false, true},
	{"batch_enabled_next_mutating", true, false, true, true},
	{"storage_next_readonly", false, true, false, false},
	{"storage_next_mutating", false, true, true, false},
	{"memory_next_readonly", false, false, false, false},
	{"memory_next_mutating", false, false, true, true},
}

func capabilityConfig(batchEnabled, storage bool) *Config {
	cfg := createDefaultConfig().(*Config)
	if !batchEnabled {
		cfg.Batch = configoptional.None[exporterhelper.BatchConfig]()
	}
	if storage {
		id := component.MustNewID("file_storage")
		cfg.StorageID = &id
	}
	return cfg
}

func mutatingTraces(t *testing.T) consumer.Traces {
	c, err := consumer.NewTraces(
		func(context.Context, ptrace.Traces) error { return nil },
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
	require.NoError(t, err)
	return c
}

func mutatingMetrics(t *testing.T) consumer.Metrics {
	c, err := consumer.NewMetrics(
		func(context.Context, pmetric.Metrics) error { return nil },
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
	require.NoError(t, err)
	return c
}

func mutatingLogs(t *testing.T) consumer.Logs {
	c, err := consumer.NewLogs(
		func(context.Context, plog.Logs) error { return nil },
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
	require.NoError(t, err)
	return c
}

func mutatingProfiles(t *testing.T) xconsumer.Profiles {
	c, err := xconsumer.NewProfiles(
		func(context.Context, pprofile.Profiles) error { return nil },
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
	require.NoError(t, err)
	return c
}

func TestTracesCapabilities(t *testing.T) {
	set := processortest.NewNopSettings(metadata.Type)
	for _, tc := range capabilityCases {
		t.Run(tc.name, func(t *testing.T) {
			var next consumer.Traces = consumertest.NewNop()
			if tc.nextMutates {
				next = mutatingTraces(t)
			}
			p, err := newTracesProcessor(context.Background(), set, capabilityConfig(tc.batchEnabled, tc.storage), next)
			require.NoError(t, err)
			require.Equal(t, tc.want, p.Capabilities().MutatesData)
		})
	}
}

func TestMetricsCapabilities(t *testing.T) {
	set := processortest.NewNopSettings(metadata.Type)
	for _, tc := range capabilityCases {
		t.Run(tc.name, func(t *testing.T) {
			var next consumer.Metrics = consumertest.NewNop()
			if tc.nextMutates {
				next = mutatingMetrics(t)
			}
			p, err := newMetricsProcessor(context.Background(), set, capabilityConfig(tc.batchEnabled, tc.storage), next)
			require.NoError(t, err)
			require.Equal(t, tc.want, p.Capabilities().MutatesData)
		})
	}
}

func TestLogsCapabilities(t *testing.T) {
	set := processortest.NewNopSettings(metadata.Type)
	for _, tc := range capabilityCases {
		t.Run(tc.name, func(t *testing.T) {
			var next consumer.Logs = consumertest.NewNop()
			if tc.nextMutates {
				next = mutatingLogs(t)
			}
			p, err := newLogsProcessor(context.Background(), set, capabilityConfig(tc.batchEnabled, tc.storage), next)
			require.NoError(t, err)
			require.Equal(t, tc.want, p.Capabilities().MutatesData)
		})
	}
}

func TestProfilesCapabilities(t *testing.T) {
	set := processortest.NewNopSettings(metadata.Type)
	for _, tc := range capabilityCases {
		t.Run(tc.name, func(t *testing.T) {
			var next xconsumer.Profiles = consumertest.NewNop()
			if tc.nextMutates {
				next = mutatingProfiles(t)
			}
			p, err := newProfilesProcessor(context.Background(), set, capabilityConfig(tc.batchEnabled, tc.storage), next)
			require.NoError(t, err)
			require.Equal(t, tc.want, p.Capabilities().MutatesData)
		})
	}
}
