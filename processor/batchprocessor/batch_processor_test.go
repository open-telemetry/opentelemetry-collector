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

package batchprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/dataold/testdataold"
)

func TestBatchProcessorSpansDelivered(t *testing.T) {
	sink := &exportertest.SinkTraceExporter{}
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 128
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	requestCount := 1000
	spansPerRequest := 100
	traceDataSlice := make([]pdata.Traces, 0, requestCount)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		traceDataSlice = append(traceDataSlice, td.Clone())
		assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
	}

	// Added to test logic that check for empty resources.
	td := testdata.GenerateTraceDataEmpty()
	assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*spansPerRequest, sink.SpansCount())
	receivedTraces := sink.AllTraces()
	spansReceivedByName := spansReceivedByName(receivedTraces)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		spans := traceDataSlice[requestNum].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			require.EqualValues(t,
				spans.At(spanIndex),
				spansReceivedByName[getTestSpanName(requestNum, spanIndex)])
		}
	}
}

func TestBatchProcessorSpansDeliveredEnforceBatchSize(t *testing.T) {
	sink := &exportertest.SinkTraceExporter{}
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 128
	cfg.SendBatchMaxSize = 128
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, cfg, telemetry.Basic)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	requestCount := 1000
	spansPerRequest := 150
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
	}

	// Added to test logic that check for empty resources.
	td := testdata.GenerateTraceDataEmpty()
	batcher.ConsumeTraces(context.Background(), td)

	// wait for all spans to be reported
	for {
		if sink.SpansCount() == requestCount*spansPerRequest {
			break
		}
		<-time.After(cfg.Timeout)
	}

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*spansPerRequest, sink.SpansCount())
	for i := 0; i < len(sink.AllTraces())-1; i++ {
		assert.Equal(t, cfg.SendBatchSize, uint32(sink.AllTraces()[i].SpanCount()))
	}
	// the last batch has the remaining size
	assert.Equal(t, (requestCount*spansPerRequest)%int(cfg.SendBatchSize), sink.AllTraces()[len(sink.AllTraces())-1].SpanCount())
}

func TestBatchProcessorSentBySize(t *testing.T) {
	views := MetricViews(telemetry.Detailed)
	view.Register(views...)
	defer view.Unregister(views...)

	sink := &exportertest.SinkTraceExporter{}
	cfg := createDefaultConfig().(*Config)
	sendBatchSize := 20
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 500 * time.Millisecond
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	requestCount := 100
	spansPerRequest := 5

	start := time.Now()
	sizeSum := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		sizeSum += td.Size()
		assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
	}

	require.NoError(t, batcher.Shutdown(context.Background()))

	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := requestCount * spansPerRequest / sendBatchSize
	expectedBatchingFactor := sendBatchSize / spansPerRequest

	require.Equal(t, requestCount*spansPerRequest, sink.SpansCount())
	receivedTraces := sink.AllTraces()
	require.EqualValues(t, expectedBatchesNum, len(receivedTraces))
	for _, td := range receivedTraces {
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).InstrumentationLibrarySpans().At(0).Spans().Len())
		}
	}

	viewData, err := view.RetrieveData(statBatchSendSize.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sink.SpansCount(), int(distData.Sum()))
	assert.Equal(t, sendBatchSize, int(distData.Min))
	assert.Equal(t, sendBatchSize, int(distData.Max))

	viewData, err = view.RetrieveData(statBatchSendSizeBytes.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sizeSum, int(distData.Sum()))
}

func TestBatchProcessorSentByTimeout(t *testing.T) {
	sink := &exportertest.SinkTraceExporter{}
	cfg := createDefaultConfig().(*Config)
	sendBatchSize := 100
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 100 * time.Millisecond
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	requestCount := 5
	spansPerRequest := 10
	start := time.Now()

	batcher := newBatchTracesProcessor(creationParams, sink, cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
	}

	// Wait for at least one batch to be sent.
	for {
		if sink.SpansCount() != 0 {
			break
		}
		<-time.After(cfg.Timeout)
	}

	elapsed := time.Since(start)
	require.LessOrEqual(t, cfg.Timeout.Nanoseconds(), elapsed.Nanoseconds())

	// This should not change the results in the sink, verified by the expectedBatchesNum
	require.NoError(t, batcher.Shutdown(context.Background()))

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*spansPerRequest, sink.SpansCount())
	receivedTraces := sink.AllTraces()
	require.EqualValues(t, expectedBatchesNum, len(receivedTraces))
	for _, td := range receivedTraces {
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).InstrumentationLibrarySpans().At(0).Spans().Len())
		}
	}
}

func TestBatchProcessorTraceSendWhenClosing(t *testing.T) {
	cfg := Config{
		Timeout:       3 * time.Second,
		SendBatchSize: 1000,
	}
	sink := &exportertest.SinkTraceExporter{}

	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, &cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	requestCount := 10
	spansPerRequest := 10
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
	}

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*spansPerRequest, sink.SpansCount())
	require.Equal(t, 1, len(sink.AllTraces()))
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
	sink := &exportertest.SinkMetricsExporter{}

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	metricDataSlice := make([]dataold.MetricData, 0, requestCount)

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdataold.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			metrics.At(metricIndex).MetricDescriptor().SetName(getTestMetricName(requestNum, metricIndex))
		}
		metricDataSlice = append(metricDataSlice, md.Clone())
		pd := pdatautil.MetricsFromOldInternalMetrics(md)
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), pd))
	}

	// Added to test case with empty resources sent.
	md := testdataold.GenerateMetricDataEmpty()
	assert.NoError(t, batcher.ConsumeMetrics(context.Background(), pdatautil.MetricsFromOldInternalMetrics(md)))

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*metricsPerRequest, sink.MetricsCount())
	receivedMds := sink.AllMetrics()
	metricsReceivedByName := metricsReceivedByName(receivedMds)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		metrics := metricDataSlice[requestNum].ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			require.EqualValues(t,
				metrics.At(metricIndex),
				metricsReceivedByName[getTestMetricName(requestNum, metricIndex)])
		}
	}
}

func TestBatchMetricProcessor_BatchSize(t *testing.T) {
	views := MetricViews(telemetry.Detailed)
	view.Register(views...)
	defer view.Unregister(views...)

	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	metricsPerRequest := 5
	sink := &exportertest.SinkMetricsExporter{}

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	start := time.Now()
	size := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdataold.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		size += md.Size()
		pd := pdatautil.MetricsFromOldInternalMetrics(md)
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), pd))
	}
	require.NoError(t, batcher.Shutdown(context.Background()))

	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := requestCount * metricsPerRequest / int(cfg.SendBatchSize)
	expectedBatchingFactor := int(cfg.SendBatchSize) / metricsPerRequest

	require.Equal(t, requestCount*metricsPerRequest, sink.MetricsCount())
	receivedMds := sink.AllMetrics()
	require.Equal(t, expectedBatchesNum, len(receivedMds))
	for _, md := range receivedMds {
		im := pdatautil.MetricsToOldInternalMetrics(md)
		require.Equal(t, expectedBatchingFactor, im.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, im.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
		}
	}

	viewData, err := view.RetrieveData(statBatchSendSize.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sink.MetricsCount(), int(distData.Sum()))
	assert.Equal(t, cfg.SendBatchSize, uint32(distData.Min))
	assert.Equal(t, cfg.SendBatchSize, uint32(distData.Max))

	viewData, err = view.RetrieveData(statBatchSendSizeBytes.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, size, int(distData.Sum()))
}

func TestBatchMetricsProcessor_Timeout(t *testing.T) {
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 100,
	}
	requestCount := 5
	metricsPerRequest := 10
	sink := &exportertest.SinkMetricsExporter{}

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	start := time.Now()
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdataold.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		pd := pdatautil.MetricsFromOldInternalMetrics(md)
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), pd))
	}

	// Wait for at least one batch to be sent.
	for {
		if sink.MetricsCount() != 0 {
			break
		}
		<-time.After(cfg.Timeout)
	}

	elapsed := time.Since(start)
	require.LessOrEqual(t, cfg.Timeout.Nanoseconds(), elapsed.Nanoseconds())

	// This should not change the results in the sink, verified by the expectedBatchesNum
	require.NoError(t, batcher.Shutdown(context.Background()))

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*metricsPerRequest, sink.MetricsCount())
	receivedMds := sink.AllMetrics()
	require.Equal(t, expectedBatchesNum, len(receivedMds))
	for _, md := range receivedMds {
		im := pdatautil.MetricsToOldInternalMetrics(md)
		require.Equal(t, expectedBatchingFactor, im.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, im.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
		}
	}
}

func TestBatchMetricProcessor_Shutdown(t *testing.T) {
	cfg := Config{
		Timeout:       3 * time.Second,
		SendBatchSize: 1000,
	}
	requestCount := 5
	metricsPerRequest := 10
	sink := &exportertest.SinkMetricsExporter{}

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, telemetry.Detailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdataold.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		pd := pdatautil.MetricsFromOldInternalMetrics(md)
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), pd))
	}

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*metricsPerRequest, sink.MetricsCount())
	require.Equal(t, 1, len(sink.AllMetrics()))
}

func getTestSpanName(requestNum, index int) string {
	return fmt.Sprintf("test-span-%d-%d", requestNum, index)
}

func spansReceivedByName(tds []pdata.Traces) map[string]pdata.Span {
	spansReceivedByName := map[string]pdata.Span{}
	for i := range tds {
		rss := tds[i].ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			rs := rss.At(i)
			if rs.IsNil() {
				continue
			}

			ilss := rs.InstrumentationLibrarySpans()
			for j := 0; j < ilss.Len(); j++ {
				ils := ilss.At(j)
				if ils.IsNil() {
					continue
				}

				spans := ils.Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)
					spansReceivedByName[spans.At(k).Name()] = span
				}
			}
		}
	}
	return spansReceivedByName
}

func metricsReceivedByName(mds []pdata.Metrics) map[string]dataold.Metric {
	metricsReceivedByName := map[string]dataold.Metric{}
	for i := range mds {
		im := pdatautil.MetricsToOldInternalMetrics(mds[i])
		rms := im.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			rm := rms.At(i)
			if rm.IsNil() {
				continue
			}
			ilms := rm.InstrumentationLibraryMetrics()
			for j := 0; j < ilms.Len(); j++ {
				ilm := ilms.At(j)
				if ilm.IsNil() {
					continue
				}
				metrics := ilm.Metrics()
				for k := 0; k < metrics.Len(); k++ {
					metric := metrics.At(k)
					metricsReceivedByName[metric.MetricDescriptor().Name()] = metric
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
	td := testdata.GenerateTraceDataManySpansSameResource(8192)
	for n := 0; n < b.N; n++ {
		fmt.Println(td.Size())
	}
}

func BenchmarkTraceSizeSpanCount(b *testing.B) {
	td := testdata.GenerateTraceDataManySpansSameResource(8192)
	for n := 0; n < b.N; n++ {
		td.SpanCount()
	}
}
