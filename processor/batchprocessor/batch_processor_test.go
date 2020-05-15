// Copyright 2019, OpenTelemetry Authors
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
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestBatchProcessorSpansDelivered(t *testing.T) {
	sender := newTestSender()
	cfg := generateDefaultConfig()
	cfg.SendBatchSize = 128
	creationParams := component.ProcessorCreateParams{Logger: zap.NewNop()}
	batcher := newBatchProcessor(creationParams, sender, cfg)
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
	batcher := newBatchProcessor(creationParams, sender, cfg)
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

	batcher := newBatchProcessor(creationParams, sender, cfg)
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
							if ils.IsNil() {
								continue
							}

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
