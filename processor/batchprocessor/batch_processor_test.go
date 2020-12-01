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
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestBatchProcessorSpansDelivered(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 128
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, cfg, configtelemetry.LevelDetailed)
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
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	cfg.SendBatchSize = 128
	cfg.SendBatchMaxSize = 128
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, cfg, configtelemetry.LevelBasic)
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
	views := MetricViews()
	require.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	sendBatchSize := 20
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 500 * time.Millisecond
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, cfg, configtelemetry.LevelDetailed)
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

	viewData, err := view.RetrieveData("processor/batch/" + statBatchSendSize.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sink.SpansCount(), int(distData.Sum()))
	assert.Equal(t, sendBatchSize, int(distData.Min))
	assert.Equal(t, sendBatchSize, int(distData.Max))

	viewData, err = view.RetrieveData("processor/batch/" + statBatchSendSizeBytes.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sizeSum, int(distData.Sum()))
}

func TestBatchProcessorSentByTimeout(t *testing.T) {
	sink := new(consumertest.TracesSink)
	cfg := createDefaultConfig().(*Config)
	sendBatchSize := 100
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 100 * time.Millisecond
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	requestCount := 5
	spansPerRequest := 10
	start := time.Now()

	batcher := newBatchTracesProcessor(creationParams, sink, cfg, configtelemetry.LevelDetailed)
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
	sink := new(consumertest.TracesSink)

	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sink, &cfg, configtelemetry.LevelDetailed)
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
	sink := new(consumertest.MetricsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	metricDataSlice := make([]pdata.Metrics, 0, requestCount)

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricsManyMetricsSameResource(metricsPerRequest)
		metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			metrics.At(metricIndex).SetName(getTestMetricName(requestNum, metricIndex))
		}
		metricDataSlice = append(metricDataSlice, md.Clone())
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))
	}

	// Added to test case with empty resources sent.
	md := testdata.GenerateMetricsEmpty()
	assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))

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
	views := MetricViews()
	require.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	metricsPerRequest := 5
	sink := new(consumertest.MetricsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	start := time.Now()
	size := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricsManyMetricsSameResource(metricsPerRequest)
		size += md.Size()
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))
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
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
		}
	}

	viewData, err := view.RetrieveData("processor/batch/" + statBatchSendSize.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sink.MetricsCount(), int(distData.Sum()))
	assert.Equal(t, cfg.SendBatchSize, uint32(distData.Min))
	assert.Equal(t, cfg.SendBatchSize, uint32(distData.Max))

	viewData, err = view.RetrieveData("processor/batch/" + statBatchSendSizeBytes.Name())
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
	sink := new(consumertest.MetricsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	start := time.Now()
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricsManyMetricsSameResource(metricsPerRequest)
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))
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
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
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
	sink := new(consumertest.MetricsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricsManyMetricsSameResource(metricsPerRequest)
		assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))
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
			ilss := rss.At(i).InstrumentationLibrarySpans()
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

func metricsReceivedByName(mds []pdata.Metrics) map[string]pdata.Metric {
	metricsReceivedByName := map[string]pdata.Metric{}
	for _, md := range mds {
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			ilms := rms.At(i).InstrumentationLibraryMetrics()
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

func TestBatchLogProcessor_ReceivingData(t *testing.T) {
	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       200 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	logsPerRequest := 5
	sink := new(consumertest.LogsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchLogsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	logDataSlice := make([]pdata.Logs, 0, requestCount)

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogDataManyLogsSameResource(logsPerRequest)
		logs := ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
		for logIndex := 0; logIndex < logsPerRequest; logIndex++ {
			logs.At(logIndex).SetName(getTestLogName(requestNum, logIndex))
		}
		logDataSlice = append(logDataSlice, ld.Clone())
		assert.NoError(t, batcher.ConsumeLogs(context.Background(), ld))
	}

	// Added to test case with empty resources sent.
	ld := testdata.GenerateLogDataEmpty()
	assert.NoError(t, batcher.ConsumeLogs(context.Background(), ld))

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordsCount())
	receivedMds := sink.AllLogs()
	logsReceivedByName := logsReceivedByName(receivedMds)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		logs := logDataSlice[requestNum].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
		for logIndex := 0; logIndex < logsPerRequest; logIndex++ {
			require.EqualValues(t,
				logs.At(logIndex),
				logsReceivedByName[getTestLogName(requestNum, logIndex)])
		}
	}
}

func TestBatchLogProcessor_BatchSize(t *testing.T) {
	views := MetricViews()
	require.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	logsPerRequest := 5
	sink := new(consumertest.LogsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchLogsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	start := time.Now()
	size := 0
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogDataManyLogsSameResource(logsPerRequest)
		size += ld.SizeBytes()
		assert.NoError(t, batcher.ConsumeLogs(context.Background(), ld))
	}
	require.NoError(t, batcher.Shutdown(context.Background()))

	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	expectedBatchesNum := requestCount * logsPerRequest / int(cfg.SendBatchSize)
	expectedBatchingFactor := int(cfg.SendBatchSize) / logsPerRequest

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordsCount())
	receivedMds := sink.AllLogs()
	require.Equal(t, expectedBatchesNum, len(receivedMds))
	for _, ld := range receivedMds {
		require.Equal(t, expectedBatchingFactor, ld.ResourceLogs().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, logsPerRequest, ld.ResourceLogs().At(i).InstrumentationLibraryLogs().At(0).Logs().Len())
		}
	}

	viewData, err := view.RetrieveData("processor/batch/" + statBatchSendSize.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, sink.LogRecordsCount(), int(distData.Sum()))
	assert.Equal(t, cfg.SendBatchSize, uint32(distData.Min))
	assert.Equal(t, cfg.SendBatchSize, uint32(distData.Max))

	viewData, err = view.RetrieveData("processor/batch/" + statBatchSendSizeBytes.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.DistributionData)
	assert.Equal(t, int64(expectedBatchesNum), distData.Count)
	assert.Equal(t, size, int(distData.Sum()))
}

func TestBatchLogsProcessor_Timeout(t *testing.T) {
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 100,
	}
	requestCount := 5
	logsPerRequest := 10
	sink := new(consumertest.LogsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchLogsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	start := time.Now()
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogDataManyLogsSameResource(logsPerRequest)
		assert.NoError(t, batcher.ConsumeLogs(context.Background(), ld))
	}

	// Wait for at least one batch to be sent.
	for {
		if sink.LogRecordsCount() != 0 {
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

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordsCount())
	receivedMds := sink.AllLogs()
	require.Equal(t, expectedBatchesNum, len(receivedMds))
	for _, ld := range receivedMds {
		require.Equal(t, expectedBatchingFactor, ld.ResourceLogs().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, logsPerRequest, ld.ResourceLogs().At(i).InstrumentationLibraryLogs().At(0).Logs().Len())
		}
	}
}

func TestBatchLogProcessor_Shutdown(t *testing.T) {
	cfg := Config{
		Timeout:       3 * time.Second,
		SendBatchSize: 1000,
	}
	requestCount := 5
	logsPerRequest := 10
	sink := new(consumertest.LogsSink)

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchLogsProcessor(createParams, sink, &cfg, configtelemetry.LevelDetailed)
	require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		ld := testdata.GenerateLogDataManyLogsSameResource(logsPerRequest)
		assert.NoError(t, batcher.ConsumeLogs(context.Background(), ld))
	}

	require.NoError(t, batcher.Shutdown(context.Background()))

	require.Equal(t, requestCount*logsPerRequest, sink.LogRecordsCount())
	require.Equal(t, 1, len(sink.AllLogs()))
}

func getTestLogName(requestNum, index int) string {
	return fmt.Sprintf("test-log-int-%d-%d", requestNum, index)
}

func logsReceivedByName(lds []pdata.Logs) map[string]pdata.LogRecord {
	logsReceivedByName := map[string]pdata.LogRecord{}
	for i := range lds {
		ld := lds[i]
		rms := ld.ResourceLogs()
		for i := 0; i < rms.Len(); i++ {
			ilms := rms.At(i).InstrumentationLibraryLogs()
			for j := 0; j < ilms.Len(); j++ {
				logs := ilms.At(j).Logs()
				for k := 0; k < logs.Len(); k++ {
					log := logs.At(k)
					logsReceivedByName[log.Name()] = log
				}
			}
		}
	}
	return logsReceivedByName
}
