// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestBatchProcessorSpansDelivered(t *testing.T) {
	sender := newTestSender()
	cfg := generateDefaultConfig()
	cfg.SendBatchSize = 128
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sender, cfg)
	requestCount := 1000
	spansPerRequest := 100
	waitForCn := sender.waitFor(requestCount*spansPerRequest, 5*time.Second)
	traceDataSlice := make([]pdata.Traces, 0, requestCount)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		spans := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
		}
		traceDataSlice = append(traceDataSlice, td.Clone())
		go batcher.ConsumeTraces(context.Background(), td)
	}

	// Added to test logic that check for empty resources.
	td := testdata.GenerateTraceDataEmpty()
	go batcher.ConsumeTraces(context.Background(), td)

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}

	require.Equal(t, requestCount*spansPerRequest, sender.spansReceived)

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		spans := traceDataSlice[requestNum].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans()
		for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
			require.EqualValues(t,
				spans.At(spanIndex),
				sender.spansReceivedByName[getTestSpanName(requestNum, spanIndex)])
		}
	}
}

func TestBatchProcessorSentBySize(t *testing.T) {
	sender := newTestSender()
	cfg := generateDefaultConfig()
	sendBatchSize := 20
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 500 * time.Millisecond
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sender, cfg)
	requestCount := 100
	spansPerRequest := 5
	waitForCn := sender.waitFor(requestCount*spansPerRequest, time.Second)

	start := time.Now()
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		go batcher.ConsumeTraces(context.Background(), td)
	}

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}

	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	sender.mtx.RLock()

	expectedBatchesNum := requestCount * spansPerRequest / sendBatchSize
	expectedBatchingFactor := sendBatchSize / spansPerRequest

	require.Equal(t, requestCount*spansPerRequest, sender.spansReceived)

	require.EqualValues(t, expectedBatchesNum, len(sender.traceDataReceived))
	for _, td := range sender.traceDataReceived {
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).InstrumentationLibrarySpans().At(0).Spans().Len())
		}
	}

	sender.mtx.RUnlock()
}

func TestBatchProcessorSentByTimeout(t *testing.T) {
	sender := newTestSender()
	cfg := generateDefaultConfig()
	sendBatchSize := 100
	cfg.SendBatchSize = uint32(sendBatchSize)
	cfg.Timeout = 100 * time.Millisecond
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}

	requestCount := 5
	spansPerRequest := 10
	waitForCn := sender.waitFor(requestCount*spansPerRequest, time.Second)

	start := time.Now()

	batcher := newBatchTracesProcessor(creationParams, sender, cfg)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		go batcher.ConsumeTraces(context.Background(), td)
	}

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}

	elapsed := time.Since(start)
	require.LessOrEqual(t, cfg.Timeout.Nanoseconds(), elapsed.Nanoseconds())

	sender.mtx.RLock()

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, sender.spansReceived, requestCount*spansPerRequest)

	require.EqualValues(t, len(sender.traceDataReceived), expectedBatchesNum)
	for _, td := range sender.traceDataReceived {
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).InstrumentationLibrarySpans().At(0).Spans().Len())
		}
	}

	sender.mtx.RUnlock()
}

func TestBatchProcessorTraceSendWhenClosing(t *testing.T) {
	cfg := Config{
		Timeout:       1 * time.Second,
		SendBatchSize: 1000,
	}
	sender := newTestSender()

	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchTracesProcessor(creationParams, sender, &cfg)

	requestCount := 10
	spansPerRequest := 10
	waitForCn := sender.waitFor(requestCount*spansPerRequest, 5*time.Second)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		td := testdata.GenerateTraceDataManySpansSameResource(spansPerRequest)
		batcher.ConsumeTraces(context.Background(), td)
	}

	batcher.Shutdown(context.Background())

	err := <-waitForCn
	if err != nil {
		t.Errorf("failed to wait for sender %s", err)
	}
	sender.mtx.RLock()

	require.Equal(t, requestCount*spansPerRequest, sender.spansReceived)
	require.Equal(t, len(sender.traceDataReceived), 1)

	expectedBatchingFactor := 10

	for _, td := range sender.traceDataReceived {
		rss := td.ResourceSpans()
		require.Equal(t, expectedBatchingFactor, rss.Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, spansPerRequest, rss.At(i).InstrumentationLibrarySpans().At(0).Spans().Len())
		}
	}

	sender.mtx.RUnlock()
}

func getTestSpanName(requestNum, index int) string {
	return fmt.Sprintf("test-span-%d-%d", requestNum, index)
}

type testSender struct {
	reqChan             chan pdata.Traces
	spansReceived       int
	traceDataReceived   []pdata.Traces
	spansReceivedByName map[string]pdata.Span
	mtx                 sync.RWMutex
}

func newTestSender() *testSender {
	return &testSender{
		reqChan:             make(chan pdata.Traces, 100),
		traceDataReceived:   make([]pdata.Traces, 0),
		spansReceivedByName: make(map[string]pdata.Span),
	}
}

func (ts *testSender) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	ts.reqChan <- td
	return nil
}

func (ts *testSender) waitFor(spans int, timeout time.Duration) chan error {
	errorCn := make(chan error)
	go func() {
		for {
			select {
			case td := <-ts.reqChan:
				ts.mtx.Lock()
				ts.traceDataReceived = append(ts.traceDataReceived, td)
				ts.spansReceived = ts.spansReceived + td.SpanCount()

				rss := td.ResourceSpans()
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
							ts.spansReceivedByName[spans.At(k).Name()] = span
						}
					}
				}
				ts.mtx.Unlock()
				if ts.spansReceived == spans {
					errorCn <- nil
				}
			case <-time.After(timeout):
				errorCn <- fmt.Errorf("timed out waiting for spans")
			}
		}
	}()
	return errorCn
}

type testMetricsSender struct {
	reqChan               chan pdata.Metrics
	metricsReceived       int
	metricsDataReceiver   []data.MetricData
	metricsReceivedByName map[string]pdata.Metric
	mtx                   sync.RWMutex
}

func newTestMetricsSender() *testMetricsSender {
	return &testMetricsSender{
		reqChan:               make(chan pdata.Metrics, 100),
		metricsDataReceiver:   make([]data.MetricData, 0),
		metricsReceivedByName: make(map[string]pdata.Metric),
	}
}

func (tms *testMetricsSender) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	tms.reqChan <- md
	return nil
}

func (tms *testMetricsSender) waitFor(metrics int, timeout time.Duration) chan error {
	errorCn := make(chan error)
	go func() {
		for {
			select {
			case md := <-tms.reqChan:
				tms.mtx.Lock()
				im := pdatautil.MetricsToInternalMetrics(md)
				tms.metricsDataReceiver = append(tms.metricsDataReceiver, im)
				tms.metricsReceived = tms.metricsReceived + im.MetricCount()

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
							tms.metricsReceivedByName[metric.MetricDescriptor().Name()] = metric
						}
					}
				}
				tms.mtx.Unlock()
				if tms.metricsReceived == metrics {
					errorCn <- nil
				}
			case <-time.After(timeout):
				errorCn <- fmt.Errorf("timed out waiting for metrics")
			}
		}
	}()
	return errorCn
}

func getTestMetricName(requestNum, index int) string {
	return fmt.Sprintf("test-metric-int-%d-%d", requestNum, index)
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
	// Instantiate upstream component to receive data after the batch processor.
	tms := newTestMetricsSender()

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, tms, &cfg)
	waitForCn := tms.waitFor(requestCount*metricsPerRequest, 5*time.Second)
	metricDataSlice := make([]data.MetricData, 0, requestCount)

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		metrics := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			metrics.At(metricIndex).MetricDescriptor().SetName(getTestMetricName(requestNum, metricIndex))
		}
		metricDataSlice = append(metricDataSlice, md.Clone())
		pd := pdatautil.MetricsFromInternalMetrics(md)
		go batcher.ConsumeMetrics(context.Background(), pd)
	}

	// Added to test case with empty resources sent.
	md := testdata.GenerateMetricDataEmpty()
	go batcher.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(md))

	err := <-waitForCn
	if err != nil {
		t.Errorf("faild to wait for sender %s", err)
	}

	tms.mtx.RLock()

	require.Equal(t, requestCount*metricsPerRequest, tms.metricsReceived)
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		metrics := metricDataSlice[requestNum].ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
		for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
			require.EqualValues(t,
				metrics.At(metricIndex),
				tms.metricsReceivedByName[getTestMetricName(requestNum, metricIndex)])
		}
	}
	tms.mtx.RUnlock()
}

func TestBatchMetricProcessor_BatchSize(t *testing.T) {
	// Instantiate the batch processor with low config values to test data
	// gets sent through the processor.
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 50,
	}

	requestCount := 100
	metricsPerRequest := 5

	// Instantiate upstream component to receive data after the batch processor.
	tms := newTestMetricsSender()

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, tms, &cfg)
	waitForCn := tms.waitFor(requestCount*metricsPerRequest, time.Second)
	start := time.Now()
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		pd := pdatautil.MetricsFromInternalMetrics(md)
		go batcher.ConsumeMetrics(context.Background(), pd)
	}
	err := <-waitForCn
	if err != nil {
		t.Errorf("faild to wait for sender %s", err)
	}

	elapsed := time.Since(start)
	require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

	tms.mtx.RLock()

	expectedBatchesNum := requestCount * metricsPerRequest / int(cfg.SendBatchSize)
	expectedBatchingFactor := int(cfg.SendBatchSize) / metricsPerRequest

	require.Equal(t, requestCount*metricsPerRequest, tms.metricsReceived)
	require.Equal(t, expectedBatchesNum, len(tms.metricsDataReceiver))
	for _, md := range tms.metricsDataReceiver {
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
		}
	}
	tms.mtx.RUnlock()
}

func TestBatchMetricsProcessor_Timeout(t *testing.T) {
	cfg := Config{
		Timeout:       100 * time.Millisecond,
		SendBatchSize: 100,
	}
	requestCount := 5
	metricsPerRequest := 10
	// Instantiate upstream component to receive data after the batch processor.
	tms := newTestMetricsSender()

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, tms, &cfg)
	waitForCn := tms.waitFor(requestCount*metricsPerRequest, time.Second)
	start := time.Now()
	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		pd := pdatautil.MetricsFromInternalMetrics(md)
		go batcher.ConsumeMetrics(context.Background(), pd)
	}
	err := <-waitForCn
	if err != nil {
		t.Errorf("faild to wait for sender %s", err)
	}

	elapsed := time.Since(start)
	require.LessOrEqual(t, cfg.Timeout.Nanoseconds(), elapsed.Nanoseconds())

	tms.mtx.RLock()

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*metricsPerRequest, tms.metricsReceived)
	require.Equal(t, expectedBatchesNum, len(tms.metricsDataReceiver))
	for _, md := range tms.metricsDataReceiver {
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
		}
	}
	tms.mtx.RUnlock()
}

func TestBatchMetricProcessor_Shutdown(t *testing.T) {
	cfg := Config{
		Timeout:       1 * time.Second,
		SendBatchSize: 1000,
	}
	requestCount := 5
	metricsPerRequest := 10

	// Instantiate upstream component to receive data after the batch processor.
	tms := newTestMetricsSender()

	createParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchMetricsProcessor(createParams, tms, &cfg)
	waitForCn := tms.waitFor(requestCount*metricsPerRequest, 5*time.Second)

	for requestNum := 0; requestNum < requestCount; requestNum++ {
		md := testdata.GenerateMetricDataManyMetricsSameResource(metricsPerRequest)
		pd := pdatautil.MetricsFromInternalMetrics(md)
		batcher.ConsumeMetrics(context.Background(), pd)
	}

	batcher.Shutdown(context.Background())

	err := <-waitForCn
	if err != nil {
		t.Errorf("faild to wait for sender %s", err)
	}

	tms.mtx.RLock()

	expectedBatchesNum := 1
	expectedBatchingFactor := 5

	require.Equal(t, requestCount*metricsPerRequest, tms.metricsReceived)
	require.Equal(t, expectedBatchesNum, len(tms.metricsDataReceiver))
	for _, md := range tms.metricsDataReceiver {
		require.Equal(t, expectedBatchingFactor, md.ResourceMetrics().Len())
		for i := 0; i < expectedBatchingFactor; i++ {
			require.Equal(t, metricsPerRequest, md.ResourceMetrics().At(i).InstrumentationLibraryMetrics().At(0).Metrics().Len())
		}
	}
	tms.mtx.RUnlock()
}
