// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
)

func sendTraces(ctx context.Context, t *testing.T, batcher processor.Traces, wg *sync.WaitGroup, td ptrace.Traces) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, batcher.ConsumeTraces(ctx, td))
	}()
}

func sendMetrics(ctx context.Context, t *testing.T, batcher processor.Metrics, wg *sync.WaitGroup, md pmetric.Metrics) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, batcher.ConsumeMetrics(ctx, md))
	}()
}

func sendLogs(ctx context.Context, t *testing.T, batcher processor.Logs, wg *sync.WaitGroup, ld plog.Logs) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, batcher.ConsumeLogs(ctx, ld))
	}()
}

func TestProcessorShutdown(t *testing.T) {
	factory := NewFactory()

	ctx := context.Background()
	processorCreationSet := processortest.NewNopSettings()

	for i := 0; i < 5; i++ {
		require.NotPanics(t, func() {
			tProc, err := factory.CreateTracesProcessor(ctx, processorCreationSet, factory.CreateDefaultConfig(), consumertest.NewNop())
			require.NoError(t, err)
			_ = tProc.Shutdown(ctx)
		})

		require.NotPanics(t, func() {
			mProc, err := factory.CreateMetricsProcessor(ctx, processorCreationSet, factory.CreateDefaultConfig(), consumertest.NewNop())
			require.NoError(t, err)
			_ = mProc.Shutdown(ctx)
		})

		require.NotPanics(t, func() {
			lProc, err := factory.CreateLogsProcessor(ctx, processorCreationSet, factory.CreateDefaultConfig(), consumertest.NewNop())
			require.NoError(t, err)
			_ = lProc.Shutdown(ctx)
		})
	}
}

func TestProcessorLifecycle(t *testing.T) {
	factory := NewFactory()

	ctx := context.Background()
	processorCreationSet := processortest.NewNopSettings()

	for i := 0; i < 5; i++ {
		tProc, err := factory.CreateTracesProcessor(ctx, processorCreationSet, factory.CreateDefaultConfig(), consumertest.NewNop())
		require.NoError(t, err)
		require.NoError(t, tProc.Start(ctx, componenttest.NewNopHost()))
		require.NoError(t, tProc.Shutdown(ctx))

		mProc, err := factory.CreateMetricsProcessor(ctx, processorCreationSet, factory.CreateDefaultConfig(), consumertest.NewNop())
		require.NoError(t, err)
		require.NoError(t, mProc.Start(ctx, componenttest.NewNopHost()))
		require.NoError(t, mProc.Shutdown(ctx))

		lProc, err := factory.CreateLogsProcessor(ctx, processorCreationSet, factory.CreateDefaultConfig(), consumertest.NewNop())
		require.NoError(t, err)
		require.NoError(t, lProc.Start(ctx, componenttest.NewNopHost()))
		require.NoError(t, lProc.Shutdown(ctx))
	}
}

func TestBatchProcessorUnbrokenParentContextSingle(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 100
	cfg.SendBatchMaxSize = 100
	cfg.Timeout = 3 * time.Second
	requestCount := 10
	spansPerRequest := 5249
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	)
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("otel")
	bg, rootSp := tracer.Start(context.Background(), "test_parent")

	createSet := exporter.Settings{
		ID:                component.MustNewID("test_exporter"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	createSet.TelemetrySettings.TracerProvider = tp

	opt := exporterhelper.WithQueue(exporterhelper.QueueSettings{
		Enabled: false,
	})
	next, err := exporterhelper.NewTracesExporter(bg, createSet, Config{}, func(context.Context, ptrace.Traces) error { return nil }, opt)
	require.NoError(t, err)

	processorSet := processortest.NewNopSettings()
	processorSet.MetricsLevel = configtelemetry.LevelDetailed
	processorSet.TracerProvider = tp
	bp, err := newBatchTracesProcessor(processorSet, next, cfg)
	require.NoError(t, err)
	require.NoError(t, bp.Start(context.Background(), componenttest.NewNopHost()))

	sentResourceSpans := ptrace.NewTraces().ResourceSpans()
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)
		spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		td.ResourceSpans().At(0).CopyTo(sentResourceSpans.AppendEmpty())
		sendTraces(bg, t, bp, &wg, td)
	}
	wg.Wait()
	rootSp.End()
	require.NoError(t, bp.Shutdown(context.Background()))

	// need to flush tracerprovider
	tp.ForceFlush(bg)
	td := exp.GetSpans()
	numBatches := float64(spansPerRequest*requestCount) / float64(cfg.SendBatchMaxSize)
	assert.Len(t, td, 2*int(math.Ceil(numBatches))+1)
	for _, span := range td {
		switch span.Name {
		case "batch_processor/export":
			// more test below
		case "exporter/test_exporter/traces":
			continue
		case "test_parent":
			continue
		default:
			t.Error("unexpected span name:", span.Name)
		}
		// confirm parent is rootSp
		assert.Equal(t, span.Parent, rootSp.SpanContext())
	}

	require.NoError(t, tp.Shutdown(context.Background()))
}

func TestBatchProcessorUnbrokenParentContextMultiple(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 100
	cfg.SendBatchMaxSize = 100
	cfg.Timeout = 3 * time.Second
	requestCount := 50
	// keep spansPerRequest small to ensure multiple contexts end up in the same batch.
	spansPerRequest := 5
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
	)
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("otel")
	bg := context.Background()

	createSet := exporter.Settings{
		ID:                component.MustNewID("test_exporter"),
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
	}
	createSet.TelemetrySettings.TracerProvider = tp
	opt := exporterhelper.WithQueue(exporterhelper.QueueSettings{
		Enabled: false,
	})
	next, err := exporterhelper.NewTracesExporter(bg, createSet, Config{}, func(context.Context, ptrace.Traces) error { return nil }, opt)
	require.NoError(t, err)

	processorSet := processortest.NewNopSettings()
	processorSet.MetricsLevel = configtelemetry.LevelDetailed
	processorSet.TracerProvider = tp
	bp, err := newBatchTracesProcessor(processorSet, next, cfg)
	require.NoError(t, err)
	require.NoError(t, bp.Start(bg, componenttest.NewNopHost()))

	var endLater []trace.Span
	mkCtx := func() context.Context {
		ctx, span := tracer.Start(bg, "test_context")
		endLater = append(endLater, span)
		return ctx
	}
	callCtxs := []context.Context{
		mkCtx(),
		mkCtx(),
		mkCtx(),
	}

	sentResourceSpans := ptrace.NewTraces().ResourceSpans()
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		num := requestNum % len(callCtxs)
		td := testdata.GenerateTraces(spansPerRequest)
		spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		td.ResourceSpans().At(0).CopyTo(sentResourceSpans.AppendEmpty())
		sendTraces(callCtxs[num], t, bp, &wg, td)
	}
	wg.Wait()

	require.NoError(t, bp.Shutdown(context.Background()))

	// Flush and reset the internal traces exporter.
	tp.ForceFlush(bg)
	td := exp.GetSpans()
	exp.Reset()

	// Expect 2 spans per batch, one exporter and one batch processor.
	numBatches := float64(spansPerRequest*requestCount) / float64(cfg.SendBatchMaxSize)
	assert.Len(t, td, 2*int(math.Ceil(numBatches)))

	var expectSpanCtxs []trace.SpanContext
	for _, span := range endLater {
		expectSpanCtxs = append(expectSpanCtxs, span.SpanContext())
	}
	for _, span := range td {
		switch span.Name {
		case "batch_processor/export":
			// more test below
		case "exporter/test_exporter/traces":
			continue
		default:
			t.Error("unexpected span name:", span.Name)
		}
		assert.Len(t, span.Links, len(callCtxs))

		var haveSpanCtxs []trace.SpanContext
		for _, link := range span.Links {
			haveSpanCtxs = append(haveSpanCtxs, link.SpanContext)
		}

		assert.ElementsMatch(t, expectSpanCtxs, haveSpanCtxs)
	}

	// End the parent spans
	for _, span := range endLater {
		span.End()
	}

	tp.ForceFlush(bg)
	td = exp.GetSpans()

	assert.Len(t, td, len(callCtxs))
	for _, span := range td {
		switch span.Name {
		case "test_context":
		default:
			t.Error("unexpected span name:", span.Name)
		}
		assert.NotEmpty(t, span.Links)
	}

	require.NoError(t, tp.Shutdown(context.Background()))
}

func TestBatchProcessorSpansDelivered(t *testing.T) {
	bg := context.Background()
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 128
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	requestCount := 1000
	spansPerRequest := 100
	sentResourceSpans := ptrace.NewTraces().ResourceSpans()
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)
		spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		td.ResourceSpans().At(0).CopyTo(sentResourceSpans.AppendEmpty())
		sendTraces(bg, t, batcher, &wg, td)
	}

	// Added to test logic that check for empty resources.
	td := ptrace.NewTraces()
	sendTraces(bg, t, batcher, &wg, td)

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
	receivedTraces := sink.AllTraces()
	spansReceivedByName := spansReceivedByName(receivedTraces)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		spans := sentResourceSpans.At(requestNum).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			require.EqualValues(t,
				spans.At(spanIndex),
				spansReceivedByName[getTestSpanName(requestNum, spanIndex)])
		}
	}
}

func TestBatchProcessorSpansDeliveredEnforceBatchSize(t *testing.T) {
	bg := context.Background()
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 128
	cfg.SendBatchMaxSize = 130
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	requestCount := 1000
	spansPerRequest := 150
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)
		spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		sendTraces(bg, t, batcher, &wg, td)
	}

	// Added to test logic that check for empty resources.
	td := ptrace.NewTraces()
	sendTraces(bg, t, batcher, &wg, td)

	// shutdown will flush any remaining spans
	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
	for i := 0; i < len(sink.AllTraces())-1; i++ {
		assert.Equal(t, int(cfg.SendBatchMaxSize), sink.AllTraces()[i].SpanCount())
	}
	// the last batch has the remaining size
	assert.Equal(t, (requestCount*spansPerRequest)%int(cfg.SendBatchMaxSize), sink.AllTraces()[len(sink.AllTraces())-1].SpanCount())
}

func TestBatchProcessorTracesSentBySize(t *testing.T) {
	tel := setupTestTelemetry()
	sizer := &ptrace.ProtoMarshaler{}
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	bg := context.Background()
	sendBatchSize := 20
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 500 * time.Millisecond
	creationSet := tel.NewSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	start := time.Now()
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	requestCount := 100
	spansPerRequest := 5

	sizeSum := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)

		sendTraces(bg, t, batcher, &wg, td)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	// We expect at no timeout periods because (items % sendBatchMaxSize) == 0.
	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := requestCount * spansPerRequest / sendBatchSize
	expectedBatchingFactor := sendBatchSize / spansPerRequest

	require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
	receivedTraces := sink.AllTraces()
	require.Len(t, receivedTraces, expectedBatchesNum)
	for _, td := range receivedTraces {
		sizeSum += sizer.TracesSize(td)
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).ScopeSpans().At(0).Spans().Len())
		}
	}

	tel.assertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_batch_batch_send_size_bytes",
			Description: "Number of bytes in batch that was sent",
			Unit:        "By",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
						Count:      uint64(expectedBatchesNum),
						Bounds: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
							100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
							1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
						BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, uint64(expectedBatchesNum), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(sizeSum),
						Min:          metricdata.NewExtrema(int64(sizeSum / expectedBatchesNum)),
						Max:          metricdata.NewExtrema(int64(sizeSum / expectedBatchesNum)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_send_size",
			Description: "Number of units in the batch",
			Unit:        "{units}",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes:   attribute.NewSet(attribute.String("processor", "batch")),
						Count:        uint64(expectedBatchesNum),
						Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
						BucketCounts: []uint64{0, uint64(expectedBatchesNum), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(sink.SpanCount()),
						Min:          metricdata.NewExtrema(int64(sendBatchSize)),
						Max:          metricdata.NewExtrema(int64(sendBatchSize)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_size_trigger_send",
			Description: "Number of times the batch was sent due to a size trigger",
			Unit:        "{times}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      int64(expectedBatchesNum),
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_metadata_cardinality",
			Description: "Number of distinct metadata value combinations being processed",
			Unit:        "{combinations}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: false,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      1,
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
	})
}

func TestBatchProcessorTracesSentByMaxSize(t *testing.T) {
	tel := setupTestTelemetry()
	sizer := &ptrace.ProtoMarshaler{}
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	bg := context.Background()
	sendBatchSize := 20
	sendBatchMaxSize := 37
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.SendBatchMaxSize = uint32(sendBatchMaxSize)
	cfg.Timeout = 500 * time.Millisecond
	creationSet := tel.NewSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	start := time.Now()
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	requestCount := 1
	spansPerRequest := 500
	totalSpans := requestCount * spansPerRequest

	sizeSum := 0
	sendTraces(bg, t, batcher, &wg, testdata.GenerateTraces(spansPerRequest))

	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	// We expect at least one timeout period because (items % sendBatchMaxSize) != 0.
	elapsed := time.Since(start)
	require.Less(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	// The max batch size is not a divisor of the total number of spans
	expectedBatchesNum := int(math.Ceil(float64(totalSpans) / float64(sendBatchMaxSize)))

	// The max batch size is a divisor of the total number of spans, which
	// ensures no timeout was needed (as tested above).
	require.Equal(t, totalSpans, sink.SpanCount())
	receivedTraces := sink.AllTraces()
	require.Len(t, receivedTraces, expectedBatchesNum)

	// we have to count the size after it was processed since splitTraces will cause some
	// repeated ResourceSpan data to be sent through the processor
	var min, max int
	for _, td := range receivedTraces {
		if min == 0 || sizer.TracesSize(td) < min {
			min = sizer.TracesSize(td)
		}
		if sizer.TracesSize(td) > max {
			max = sizer.TracesSize(td)
		}
		sizeSum += sizer.TracesSize(td)
	}

	tel.assertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_batch_batch_send_size_bytes",
			Description: "Number of bytes in batch that was sent",
			Unit:        "By",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
						Count:      uint64(expectedBatchesNum),
						Bounds: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
							100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
							1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
						BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, uint64(expectedBatchesNum - 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(sizeSum),
						Min:          metricdata.NewExtrema(int64(min)),
						Max:          metricdata.NewExtrema(int64(max)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_send_size",
			Description: "Number of units in the batch",
			Unit:        "{units}",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes:   attribute.NewSet(attribute.String("processor", "batch")),
						Count:        uint64(expectedBatchesNum),
						Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
						BucketCounts: []uint64{0, 1, uint64(expectedBatchesNum - 1), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(sink.SpanCount()),
						Min:          metricdata.NewExtrema(int64(uint32(spansPerRequest) % cfg.SendBatchMaxSize)),
						Max:          metricdata.NewExtrema(int64(cfg.SendBatchMaxSize)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_size_trigger_send",
			Description: "Number of times the batch was sent due to a size trigger",
			Unit:        "{times}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      int64(expectedBatchesNum - 1),
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_timeout_trigger_send",
			Description: "Number of times the batch was sent due to a timeout trigger",
			Unit:        "{times}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      1,
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_metadata_cardinality",
			Description: "Number of distinct metadata value combinations being processed",
			Unit:        "{combinations}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: false,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      1,
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
	})
}

func TestBatchProcessorSentByTimeout(t *testing.T) {
	bg := context.Background()
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	sendBatchSize := 100
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 100 * time.Millisecond

	requestCount := 5
	spansPerRequest := 10

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	start := time.Now()
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)
		sendTraces(bg, t, batcher, &wg, td)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	elapsed := time.Since(start)
	// We should not observe a wait for the timeout.
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
	receivedTraces := sink.AllTraces()
	require.Len(t, receivedTraces, expectedBatchesNum)
	for _, td := range receivedTraces {
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).ScopeSpans().At(0).Spans().Len())
		}
	}
}

func TestBatchProcessorTraceSendWhenClosing(t *testing.T) {
	bg := context.Background()
	cfg := Config{
		Timeout:       3 * time.Second,
		SendBatchSize: 1000,
	}
	sink := new(consumertest.TracesSink)

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	requestCount := 10
	spansPerRequest := 10
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)
		sendTraces(bg, t, batcher, &wg, td)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
	require.Len(t, sink.AllTraces(), 1)
}

func TestBatchMetricProcessor_ReceivingData(t *testing.T) {
	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       200 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	metricsPerRequest := 5
	sink := new(consumertest.MetricsSink)

	bg := context.Background()
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	sentResourceMetrics := pmetric.NewMetrics().ResourceMetrics()

	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetrics(metricsPerRequest)
		metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			metrics.At(metricIndex).SetName(getTestMetricName(requestNum, metricIndex))
		}
		md.ResourceMetrics().At(0).CopyTo(sentResourceMetrics.AppendEmpty())
		sendMetrics(bg, t, batcher, &wg, md)
	}

	// Added to test case with empty resources sent.
	md := pmetric.NewMetrics()
	sendMetrics(bg, t, batcher, &wg, md)

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*2*metricsPerRequest, sink.DataPointCount())
	receivedMds := sink.AllMetrics()
	metricsReceivedByName := metricsReceivedByName(receivedMds)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		metrics := sentResourceMetrics.At(requestNum).ScopeMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			require.EqualValues(t,
				metrics.At(metricIndex),
				metricsReceivedByName[getTestMetricName(requestNum, metricIndex)])
		}
	}
}

func TestBatchMetricProcessorBatchSize(t *testing.T) {
	tel := setupTestTelemetry()
	bg := context.Background()
	sizer := &pmetric.ProtoMarshaler{}

	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	metricsPerRequest := 5
	dataPointsPerMetric := 2 // Since the int counter uses two datapoints.
	dataPointsPerRequest := metricsPerRequest * dataPointsPerMetric
	sink := new(consumertest.MetricsSink)

	creationSet := tel.NewSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	start := time.Now()
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	size := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetrics(metricsPerRequest)
		size += sizer.MetricsSize(md)
		sendMetrics(bg, t, batcher, &wg, md)
	}
	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	// We expect no timeout periods because (items % sendBatchMaxSize) == 0.
	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := requestCount * dataPointsPerRequest / int(cfg.SendBatchSize)
	expectedBatchingFactor := int(cfg.SendBatchSize) / dataPointsPerRequest

	require.Equal(t, requestCount*2*metricsPerRequest, sink.DataPointCount())
	receivedMds := sink.AllMetrics()
	require.Len(t, receivedMds, expectedBatchesNum)
	for _, md := range receivedMds {
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).ScopeMetrics().At(0).Metrics().Len())
		}
	}

	tel.assertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_batch_batch_send_size_bytes",
			Description: "Number of bytes in batch that was sent",
			Unit:        "By",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
						Count:      uint64(expectedBatchesNum),
						Bounds: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
							100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
							1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
						BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, uint64(expectedBatchesNum), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(size),
						Min:          metricdata.NewExtrema(int64(size / expectedBatchesNum)),
						Max:          metricdata.NewExtrema(int64(size / expectedBatchesNum)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_send_size",
			Description: "Number of units in the batch",
			Unit:        "{units}",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes:   attribute.NewSet(attribute.String("processor", "batch")),
						Count:        uint64(expectedBatchesNum),
						Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
						BucketCounts: []uint64{0, 0, uint64(expectedBatchesNum), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(sink.DataPointCount()),
						Min:          metricdata.NewExtrema(int64(cfg.SendBatchSize)),
						Max:          metricdata.NewExtrema(int64(cfg.SendBatchSize)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_size_trigger_send",
			Description: "Number of times the batch was sent due to a size trigger",
			Unit:        "{times}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      int64(expectedBatchesNum),
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_metadata_cardinality",
			Description: "Number of distinct metadata value combinations being processed",
			Unit:        "{combinations}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: false,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      1,
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
	})
}

func TestBatchMetrics_UnevenBatchMaxSize(t *testing.T) {
	ctx := context.Background()
	sink := new(metricsSink)
	metricsCount := 50
	dataPointsPerMetric := 2
	sendBatchMaxSize := 99

	batchMetrics := newBatchMetrics(sink)
	md := testdata.GenerateMetrics(metricsCount)

	batchMetrics.add(md)
	require.Equal(t, dataPointsPerMetric*metricsCount, batchMetrics.dataPointCount)
	sent, req := batchMetrics.splitBatch(ctx, sendBatchMaxSize)
	sendErr := batchMetrics.export(ctx, req)
	require.NoError(t, sendErr)
	require.Equal(t, sendBatchMaxSize, sent)
	remainingDataPointCount := metricsCount*dataPointsPerMetric - sendBatchMaxSize
	require.Equal(t, remainingDataPointCount, batchMetrics.dataPointCount)
}

func TestBatchMetricsProcessor_Timeout(t *testing.T) {
	bg := context.Background()
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 101,
	}
	requestCount := 5
	metricsPerRequest := 10
	sink := new(consumertest.MetricsSink)

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	var wg sync.WaitGroup

	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetrics(metricsPerRequest)
		sendMetrics(bg, t, batcher, &wg, md)
	}

	wg.Wait()

	require.NoError(t, batcher.Shutdown(context.Background()))

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*2*metricsPerRequest, sink.DataPointCount())
	receivedMds := sink.AllMetrics()
	require.Len(t, receivedMds, expectedBatchesNum)
	for _, md := range receivedMds {
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).ScopeMetrics().At(0).Metrics().Len())
		}
	}
}

func TestBatchMetricProcessor_Shutdown(t *testing.T) {
	bg := context.Background()
	cfg := Config{
		Timeout:       3 * time.Second,
		SendBatchSize: 1000,
	}
	requestCount := 5
	metricsPerRequest := 10
	sink := new(consumertest.MetricsSink)

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetrics(metricsPerRequest)
		sendMetrics(bg, t, batcher, &wg, md)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*2*metricsPerRequest, sink.DataPointCount())
	require.Len(t, sink.AllMetrics(), 1)
}

func getTestSpanName(requestNum, index int) string {
	return fmt.Sprintf("test-span-%d-%d", requestNum, index)
}

func spansReceivedByName(tds []ptrace.Traces) map[string]ptrace.Span {
	spansReceivedByName := map[string]ptrace.Span{}
	for i := range tds {
		rss := tds[i].ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			ilss := rss.At(i).ScopeSpans()
			for j := 0; j < ilss.Len(); j++ {
				spans := ilss.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)
					spansReceivedByName[spans.At(k).Name()] = span
				}
			}
		}
	}
	return spansReceivedByName
}

func metricsReceivedByName(mds []pmetric.Metrics) map[string]pmetric.Metric {
	metricsReceivedByName := map[string]pmetric.Metric{}
	for _, md := range mds {
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			ilms := rms.At(i).ScopeMetrics()
			for j := 0; j < ilms.Len(); j++ {
				metrics := ilms.At(j).Metrics()
				for k := 0; k < metrics.Len(); k++ {
					metric := metrics.At(k)
					metricsReceivedByName[metric.Name()] = metric
				}
			}
		}
	}
	return metricsReceivedByName
}

func getTestMetricName(requestNum, index int) string {
	return fmt.Sprintf("test-metric-int-%d-%d", requestNum, index)
}

func BenchmarkTraceSizeBytes(b *testing.B) {
	sizer := &ptrace.ProtoMarshaler{}
	td := testdata.GenerateTraces(8192)
	for n := 0; n < b.N; n++ {
		fmt.Println(sizer.TracesSize(td))
	}
}

func BenchmarkTraceSizeSpanCount(b *testing.B) {
	td := testdata.GenerateTraces(8192)
	for n := 0; n < b.N; n++ {
		td.SpanCount()
	}
}

func BenchmarkBatchMetricProcessor(b *testing.B) {
	b.StopTimer()
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 2000,
	}
	runMetricsProcessorBenchmark(b, cfg)
}

func BenchmarkMultiBatchMetricProcessor(b *testing.B) {
	b.StopTimer()
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 2000,
		MetadataKeys:  []string{"test", "test2"},
	}
	runMetricsProcessorBenchmark(b, cfg)
}

func runMetricsProcessorBenchmark(b *testing.B, cfg Config) {
	ctx := context.Background()
	sink := new(metricsSink)
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	metricsPerRequest := 1000
	batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg)
	require.NoError(b, err)
	require.NoError(b, batcher.Start(ctx, componenttest.NewNopHost()))

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			require.NoError(b, batcher.ConsumeMetrics(ctx, testdata.GenerateMetrics(metricsPerRequest)))
		}
	})
	b.StopTimer()
	require.NoError(b, batcher.Shutdown(ctx))
	require.Equal(b, b.N*metricsPerRequest, sink.metricsCount)
}

type metricsSink struct {
	mu           sync.Mutex
	metricsCount int
}

func (sme *metricsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{
		MutatesData: false,
	}
}

func (sme *metricsSink) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	sme.metricsCount += md.MetricCount()
	return nil
}

func TestBatchLogProcessor_ReceivingData(t *testing.T) {
	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       200 * time.Millisecond,
		SendBatchSize: 50,
	}
	bg := context.Background()

	requestCount := 100
	logsPerRequest := 5
	sink := new(consumertest.LogsSink)

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	sentResourceLogs := plog.NewLogs().ResourceLogs()

	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogs(logsPerRequest)
		logs := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
		for logIndex := 0; logIndex < logsPerRequest; logIndex++ {
			logs.At(logIndex).SetSeverityText(getTestLogSeverityText(requestNum, logIndex))
		}
		ld.ResourceLogs().At(0).CopyTo(sentResourceLogs.AppendEmpty())
		sendLogs(bg, t, batcher, &wg, ld)
	}

	// Added to test case with empty resources sent.
	ld := plog.NewLogs()
	sendLogs(bg, t, batcher, &wg, ld)

	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordCount())
	receivedMds := sink.AllLogs()
	logsReceivedBySeverityText := logsReceivedBySeverityText(receivedMds)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		logs := sentResourceLogs.At(requestNum).ScopeLogs().At(0).LogRecords()
		for logIndex := 0; logIndex < logsPerRequest; logIndex++ {
			require.EqualValues(t,
				logs.At(logIndex),
				logsReceivedBySeverityText[getTestLogSeverityText(requestNum, logIndex)])
		}
	}
}

func TestBatchLogProcessor_BatchSize(t *testing.T) {
	tel := setupTestTelemetry()
	sizer := &plog.ProtoMarshaler{}
	bg := context.Background()

	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	logsPerRequest := 5
	sink := new(consumertest.LogsSink)

	creationSet := tel.NewSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	start := time.Now()
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	size := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogs(logsPerRequest)
		size += sizer.LogsSize(ld)
		sendLogs(bg, t, batcher, &wg, ld)
	}
	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	elapsed := time.Since(start)
	// We expect no timeout periods because (items % sendBatchMaxSize) == 0.
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := requestCount * logsPerRequest / int(cfg.SendBatchSize)
	expectedBatchingFactor := int(cfg.SendBatchSize) / logsPerRequest

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordCount())
	receivedMds := sink.AllLogs()
	require.Len(t, receivedMds, expectedBatchesNum)
	for _, ld := range receivedMds {
		require.Equal(t, expectedBatchingFactor, ld.ResourceLogs().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, logsPerRequest, ld.ResourceLogs().At(i).ScopeLogs().At(0).LogRecords().Len())
		}
	}

	tel.assertMetrics(t, []metricdata.Metrics{
		{
			Name:        "otelcol_processor_batch_batch_send_size_bytes",
			Description: "Number of bytes in batch that was sent",
			Unit:        "By",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
						Count:      uint64(expectedBatchesNum),
						Bounds: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
							100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
							1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
						BucketCounts: []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, uint64(expectedBatchesNum), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(size),
						Min:          metricdata.NewExtrema(int64(size / expectedBatchesNum)),
						Max:          metricdata.NewExtrema(int64(size / expectedBatchesNum)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_send_size",
			Description: "Number of units in the batch",
			Unit:        "{units}",
			Data: metricdata.Histogram[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[int64]{
					{
						Attributes:   attribute.NewSet(attribute.String("processor", "batch")),
						Count:        uint64(expectedBatchesNum),
						Bounds:       []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
						BucketCounts: []uint64{0, 0, uint64(expectedBatchesNum), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
						Sum:          int64(sink.LogRecordCount()),
						Min:          metricdata.NewExtrema(int64(cfg.SendBatchSize)),
						Max:          metricdata.NewExtrema(int64(cfg.SendBatchSize)),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_batch_size_trigger_send",
			Description: "Number of times the batch was sent due to a size trigger",
			Unit:        "{times}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      int64(expectedBatchesNum),
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
		{
			Name:        "otelcol_processor_batch_metadata_cardinality",
			Description: "Number of distinct metadata value combinations being processed",
			Unit:        "{combinations}",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: false,
				DataPoints: []metricdata.DataPoint[int64]{
					{
						Value:      1,
						Attributes: attribute.NewSet(attribute.String("processor", "batch")),
					},
				},
			},
		},
	})
}

func TestBatchLogsProcessor_Timeout(t *testing.T) {
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 100,
	}
	bg := context.Background()
	requestCount := 5
	logsPerRequest := 10
	sink := new(consumertest.LogsSink)

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)

	var wg sync.WaitGroup
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogs(logsPerRequest)
		sendLogs(bg, t, batcher, &wg, ld)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordCount())
	receivedMds := sink.AllLogs()
	require.Len(t, receivedMds, expectedBatchesNum)
	for _, ld := range receivedMds {
		require.Equal(t, expectedBatchingFactor, ld.ResourceLogs().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, logsPerRequest, ld.ResourceLogs().At(i).ScopeLogs().At(0).LogRecords().Len())
		}
	}
}

func TestBatchLogProcessor_Shutdown(t *testing.T) {
	cfg := Config{
		Timeout:       3 * time.Second,
		SendBatchSize: 1000,
	}
	bg := context.Background()
	requestCount := 5
	logsPerRequest := 10
	sink := new(consumertest.LogsSink)

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogs(logsPerRequest)
		sendLogs(bg, t, batcher, &wg, ld)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordCount())
	require.Len(t, sink.AllLogs(), 1)
}

func getTestLogSeverityText(requestNum, index int) string {
	return fmt.Sprintf("test-log-int-%d-%d", requestNum, index)
}

func logsReceivedBySeverityText(lds []plog.Logs) map[string]plog.LogRecord {
	logsReceivedBySeverityText := map[string]plog.LogRecord{}
	for i := range lds {
		ld := lds[i]
		rms := ld.ResourceLogs()
		for i := 0; i < rms.Len(); i++ {
			ilms := rms.At(i).ScopeLogs()
			for j := 0; j < ilms.Len(); j++ {
				logs := ilms.At(j).LogRecords()
				for k := 0; k < logs.Len(); k++ {
					log := logs.At(k)
					logsReceivedBySeverityText[log.SeverityText()] = log
				}
			}
		}
	}
	return logsReceivedBySeverityText
}

func TestShutdown(t *testing.T) {
	factory := NewFactory()
	verifyTracesDoesNotProduceAfterShutdown(t, factory, factory.CreateDefaultConfig())
}
func verifyTracesDoesNotProduceAfterShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	// Create a proc and output its produce to a sink.
	nextSink := new(consumertest.TracesSink)
	bg := context.Background()
	proc, err := factory.CreateTracesProcessor(bg, processortest.NewNopSettings(), cfg, nextSink)
	if err != nil {
		if errors.Is(err, pipeline.ErrSignalNotSupported) {
			return
		}
		require.NoError(t, err)
	}
	require.NoError(t, proc.Start(bg, componenttest.NewNopHost()))

	// Send some traces to the proc.
	const generatedCount = 10
	var wg sync.WaitGroup
	for i := 0; i < generatedCount; i++ {
		sendTraces(bg, t, proc, &wg, testdata.GenerateTraces(1))
	}

	// Now shutdown the proc.
	wg.Wait()
	require.NoError(t, proc.Shutdown(bg))

	// The Shutdown() is done. It means the proc must have sent everything we
	// gave it to the next sink.
	assert.EqualValues(t, generatedCount, nextSink.SpanCount())
}

type metadataTracesSink struct {
	*consumertest.TracesSink

	lock               sync.Mutex
	spanCountByToken12 map[string]int
}

func formatTwo(first, second []string) string {
	return fmt.Sprintf("%s;%s", first, second)
}

func (mts *metadataTracesSink) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	info := client.FromContext(ctx)
	token1 := info.Metadata.Get("token1")
	token2 := info.Metadata.Get("token2")
	mts.lock.Lock()
	defer mts.lock.Unlock()

	mts.spanCountByToken12[formatTwo(
		token1,
		token2,
	)] += td.SpanCount()
	return mts.TracesSink.ConsumeTraces(ctx, td)
}

func TestBatchProcessorSpansBatchedByMetadata(t *testing.T) {
	sink := &metadataTracesSink{
		TracesSink:         &consumertest.TracesSink{},
		spanCountByToken12: map[string]int{},
	}
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 1000
	cfg.Timeout = 10 * time.Minute
	cfg.MetadataKeys = []string{"token1", "token2"}
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	bg := context.Background()
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	callCtxs := []context.Context{
		client.NewContext(bg, client.Info{
			Metadata: client.NewMetadata(map[string][]string{
				"token1": {"single"},
				"token3": {"n/a"},
			}),
		}),
		client.NewContext(bg, client.Info{
			Metadata: client.NewMetadata(map[string][]string{
				"token1": {"single"},
				"token2": {"one", "two"},
				"token4": {"n/a"},
			}),
		}),
		client.NewContext(bg, client.Info{
			Metadata: client.NewMetadata(map[string][]string{
				"token1": nil,
				"token2": {"single"},
			}),
		}),
		client.NewContext(bg, client.Info{
			Metadata: client.NewMetadata(map[string][]string{
				"token1": {"one", "two", "three"},
				"token2": {"single"},
				"token3": {"n/a"},
				"token4": {"n/a", "d/c"},
			}),
		}),
	}
	expectByContext := make([]int, len(callCtxs))

	requestCount := 1000
	spansPerRequest := 33
	sentResourceSpans := ptrace.NewTraces().ResourceSpans()
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraces(spansPerRequest)
		spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		td.ResourceSpans().At(0).CopyTo(sentResourceSpans.AppendEmpty())
		// use round-robin to assign context.
		num := requestNum % len(callCtxs)
		expectByContext[num] += spansPerRequest
		sendTraces(callCtxs[num], t, batcher, &wg, td)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))

	// The following tests are the same as TestBatchProcessorSpansDelivered().
	require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
	receivedTraces := sink.AllTraces()
	spansReceivedByName := spansReceivedByName(receivedTraces)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		spans := sentResourceSpans.At(requestNum).ScopeSpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			require.EqualValues(t,
				spans.At(spanIndex),
				spansReceivedByName[getTestSpanName(requestNum, spanIndex)])
		}
	}

	// This test ensures each context had the expected number of spans.
	require.Equal(t, len(callCtxs), len(sink.spanCountByToken12))
	for idx, ctx := range callCtxs {
		md := client.FromContext(ctx).Metadata
		exp := formatTwo(md.Get("token1"), md.Get("token2"))
		require.Equal(t, expectByContext[idx], sink.spanCountByToken12[exp])
	}
}

func TestBatchProcessorDuplicateMetadataKeys(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.MetadataKeys = []string{"myTOKEN", "mytoken"}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
	require.Contains(t, err.Error(), "mytoken")
}

func TestBatchProcessorMetadataCardinalityLimit(t *testing.T) {
	const cardLimit = 10

	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.MetadataKeys = []string{"token"}
	cfg.MetadataCardinalityLimit = cardLimit
	cfg.Timeout = 1 * time.Second
	creationSet := processortest.NewNopSettings()
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	bg := context.Background()
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < cardLimit; requestNum++ {
		td := testdata.GenerateTraces(1)
		ctx := client.NewContext(bg, client.Info{
			Metadata: client.NewMetadata(map[string][]string{
				"token": {fmt.Sprint(requestNum)},
			}),
		})

		sendTraces(ctx, t, batcher, &wg, td)
	}

	wg.Wait()
	td := testdata.GenerateTraces(2)
	ctx := client.NewContext(bg, client.Info{
		Metadata: client.NewMetadata(map[string][]string{
			"token": {"limit_exceeded"},
		}),
	})

	wg.Add(1)
	go func() {
		err := batcher.ConsumeTraces(ctx, td)
		assert.ErrorIs(t, err, errTooManyBatchers)
		wg.Done()
	}()

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))
}

func TestBatchZeroConfig(t *testing.T) {
	// This is a no-op configuration. No need for a timer, no
	// minimum, no mxaimum, just a pass through.
	cfg := Config{}
	bg := context.Background()

	require.NoError(t, cfg.Validate())

	const requestCount = 5
	const logsPerRequest = 10
	sink := new(consumertest.LogsSink)
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	expect := 0
	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		cnt := logsPerRequest + requestNum
		expect += cnt
		ld := testdata.GenerateLogs(cnt)
		sendLogs(bg, t, batcher, &wg, ld)
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	// Expect them to be the original sizes.  Since they can
	// arrive out of order, we use an ElementsMatch test below.
	receivedMds := sink.AllLogs()
	require.Len(t, receivedMds, requestCount)
	var receiveSizes []int
	var expectSizes []int
	for i, ld := range receivedMds {
		require.Equal(t, 1, ld.ResourceLogs().Len())
		expectSizes = append(expectSizes, logsPerRequest+i)
		receiveSizes = append(receiveSizes, ld.LogRecordCount())
	}
	assert.ElementsMatch(t, expectSizes, receiveSizes)
}

func TestBatchSplitOnly(t *testing.T) {
	const maxBatch = 10
	const requestCount = 5
	const logsPerRequest = 100

	cfg := Config{
		SendBatchMaxSize: maxBatch,
	}
	bg := context.Background()

	require.NoError(t, cfg.Validate())

	sink := new(consumertest.LogsSink)
	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchLogsProcessor(creationSet, sink, &cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(bg, componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogs(logsPerRequest)
		sendLogs(bg, t, batcher, &wg, ld)
	}

	// Wait for all batches.
	wg.Wait()
	require.NoError(t, batcher.Shutdown(bg))

	// Expect them to be the limited by maxBatch.
	receivedMds := sink.AllLogs()
	require.Len(t, receivedMds, requestCount*logsPerRequest/maxBatch)
	for _, ld := range receivedMds {
		require.Equal(t, maxBatch, ld.LogRecordCount())
	}
}

func TestBatchProcessorEmptyBatch(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	sendBatchSize := 100
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 100 * time.Millisecond

	requestCount := 5

	creationSet := processortest.NewNopSettings()
	creationSet.MetricsLevel = configtelemetry.LevelDetailed
	batcher, err := newBatchTracesProcessor(creationSet, sink, cfg)
	require.NoError(t, err)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := ptrace.NewTraces()
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
		}()
	}

	wg.Wait()
	require.NoError(t, batcher.Shutdown(context.Background()))
}
