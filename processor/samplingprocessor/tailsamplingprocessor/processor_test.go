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

package tailsamplingprocessor

import (
	"context"
	"sync"
	"testing"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/processor/tailsamplingprocessor/idbatcher"
	"github.com/open-telemetry/opentelemetry-collector/processor/tailsamplingprocessor/sampling"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

const (
	defaultTestDecisionWait = 30 * time.Second
)

var testPolicy = []PolicyCfg{{Name: "test-policy", Type: AlwaysSample}}

func TestSequentialTraceArrival(t *testing.T) {
	traceIds, batches := generateIdsAndBatches(128)
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(2 * len(traceIds)),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := NewTraceProcessor(zap.NewNop(), &exportertest.SinkTraceExporter{}, cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		tsp.ConsumeTraceData(context.Background(), batch)
	}

	for i := range traceIds {
		d, ok := tsp.idToTrace.Load(traceKey(traceIds[i]))
		v := d.(*sampling.TraceData)
		if !ok {
			t.Fatal("Missing expected traceId")
		} else if v.SpanCount != int64(i+1) {
			t.Fatalf("Incorrect number of spans for entry %d, got %d, want %d", i, v.SpanCount, i+1)
		}
	}
}

func TestConcurrentTraceArrival(t *testing.T) {
	traceIds, batches := generateIdsAndBatches(128)

	var wg sync.WaitGroup
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(2 * len(traceIds)),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := NewTraceProcessor(zap.NewNop(), &exportertest.SinkTraceExporter{}, cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		// Add the same traceId twice.
		wg.Add(2)
		go func(td consumerdata.TraceData) {
			td.SourceFormat = "test-0"
			tsp.ConsumeTraceData(context.Background(), td)
			wg.Done()
		}(batch)
		go func(td consumerdata.TraceData) {
			td.SourceFormat = "test-1"
			tsp.ConsumeTraceData(context.Background(), td)
			wg.Done()
		}(batch)
	}

	wg.Wait()

	for i := range traceIds {
		d, ok := tsp.idToTrace.Load(traceKey(traceIds[i]))
		v := d.(*sampling.TraceData)
		if !ok {
			t.Fatal("Missing expected traceId")
		} else if v.SpanCount != int64(i+1)*2 {
			t.Fatalf("Incorrect number of spans for entry %d, got %d, want %d", i, v.SpanCount, i+1)
		}
	}
}

func TestSequentialTraceMapSize(t *testing.T) {
	traceIds, batches := generateIdsAndBatches(210)
	const maxSize = 100
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(maxSize),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := NewTraceProcessor(zap.NewNop(), &exportertest.SinkTraceExporter{}, cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		tsp.ConsumeTraceData(context.Background(), batch)
	}

	// On sequential insertion it is possible to know exactly which traces should be still on the map.
	for i := 0; i < len(traceIds)-maxSize; i++ {
		if _, ok := tsp.idToTrace.Load(traceKey(traceIds[i])); ok {
			t.Fatalf("Found unexpected traceId[%d] still on map (id: %v)", i, traceIds[i])
		}
	}
}

func TestConcurrentTraceMapSize(t *testing.T) {
	_, batches := generateIdsAndBatches(210)
	const maxSize = 100
	var wg sync.WaitGroup
	cfg := Config{
		DecisionWait:            defaultTestDecisionWait,
		NumTraces:               uint64(maxSize),
		ExpectedNewTracesPerSec: 64,
		PolicyCfgs:              testPolicy,
	}
	sp, _ := NewTraceProcessor(zap.NewNop(), &exportertest.SinkTraceExporter{}, cfg)
	tsp := sp.(*tailSamplingSpanProcessor)
	for _, batch := range batches {
		wg.Add(1)
		go func(td consumerdata.TraceData) {
			tsp.ConsumeTraceData(context.Background(), td)
			wg.Done()
		}(batch)
	}

	wg.Wait()

	// Since we can't guarantee the order of insertion the only thing that can be checked is
	// if the number of traces on the map matches the expected value.
	cnt := 0
	tsp.idToTrace.Range(func(_ interface{}, _ interface{}) bool {
		cnt++
		return true
	})
	if cnt != maxSize {
		t.Fatalf("got %d traces on idToTrace, want %d", cnt, maxSize)
	}
}

func TestSamplingPolicyTypicalPath(t *testing.T) {
	const maxSize = 100
	const decisionWaitSeconds = 5
	// For this test explicitly control the timer calls and batcher, and set a mock
	// sampling policy evaluator.
	msp := &mockSpanProcessor{}
	mpe := &mockPolicyEvaluator{}
	mtt := &manualTTicker{}
	tsp := &tailSamplingSpanProcessor{
		ctx:             context.Background(),
		nextConsumer:    msp,
		maxNumTraces:    maxSize,
		logger:          zap.NewNop(),
		decisionBatcher: newSyncIDBatcher(decisionWaitSeconds),
		policies:        []*Policy{{Name: "mock-policy", Evaluator: mpe, ctx: context.TODO()}},
		deleteChan:      make(chan traceKey, maxSize),
		policyTicker:    mtt,
	}

	_, batches := generateIdsAndBatches(210)
	currItem := 0
	numSpansPerBatchWindow := 10
	// First evaluations shouldn't have anything to evaluate, until decision wait time passed.
	for evalNum := 0; evalNum < decisionWaitSeconds; evalNum++ {
		for ; currItem < numSpansPerBatchWindow*(evalNum+1); currItem++ {
			tsp.ConsumeTraceData(context.Background(), batches[currItem])
			if !mtt.Started {
				t.Fatalf("Time ticker was expected to have started")
			}
		}
		tsp.samplingPolicyOnTick()
		if msp.TotalSpans != 0 || mpe.EvaluationCount != 0 {
			t.Fatalf("policy for initial items was evaluated before decision wait period")
		}
	}

	// Now the first batch that waited the decision period.
	mpe.NextDecision = sampling.Sampled
	tsp.samplingPolicyOnTick()
	if msp.TotalSpans == 0 || mpe.EvaluationCount == 0 {
		t.Fatalf("policy should have been evaluated totalspans == %d and evaluationcount == %d", msp.TotalSpans, mpe.EvaluationCount)
	}

	if msp.TotalSpans != numSpansPerBatchWindow {
		t.Fatalf("not all spans of first window were accounted for: got %d, want %d", msp.TotalSpans, numSpansPerBatchWindow)
	}

	// Late span of a sampled trace should be sent directly down the pipeline exporter
	tsp.ConsumeTraceData(context.Background(), batches[0])
	expectedNumWithLateSpan := numSpansPerBatchWindow + 1
	if msp.TotalSpans != expectedNumWithLateSpan {
		t.Fatalf("late span was not accounted for: got %d, want %d", msp.TotalSpans, expectedNumWithLateSpan)
	}
	if mpe.LateArrivingSpansCount != 1 {
		t.Fatalf("policy was not notified of the late span")
	}
}

func generateIdsAndBatches(numIds int) ([][]byte, []consumerdata.TraceData) {
	traceIds := make([][]byte, numIds)
	for i := 0; i < numIds; i++ {
		traceIds[i] = tracetranslator.UInt64ToByteTraceID(1, uint64(i+1))
	}

	tds := []consumerdata.TraceData{}
	for i := range traceIds {
		spans := make([]*tracepb.Span, i+1)
		for j := range spans {
			spans[j] = &tracepb.Span{
				TraceId: traceIds[i],
				SpanId:  tracetranslator.UInt64ToByteSpanID(uint64(i + 1)),
			}
		}

		// Send each span in a separate batch
		for _, span := range spans {
			td := consumerdata.TraceData{
				Spans:        []*tracepb.Span{span},
				SourceFormat: "test",
			}
			tds = append(tds, td)
		}
	}

	return traceIds, tds
}

type mockPolicyEvaluator struct {
	NextDecision           sampling.Decision
	NextError              error
	EvaluationCount        int
	LateArrivingSpansCount int
	OnDroppedSpansCount    int
}

var _ (sampling.PolicyEvaluator) = (*mockPolicyEvaluator)(nil)

func (m *mockPolicyEvaluator) OnLateArrivingSpans(earlyDecision sampling.Decision, spans []*tracepb.Span) error {
	m.LateArrivingSpansCount++
	return m.NextError
}
func (m *mockPolicyEvaluator) Evaluate(traceID []byte, trace *sampling.TraceData) (sampling.Decision, error) {
	m.EvaluationCount++
	return m.NextDecision, m.NextError
}
func (m *mockPolicyEvaluator) OnDroppedSpans(traceID []byte, trace *sampling.TraceData) (sampling.Decision, error) {
	m.OnDroppedSpansCount++
	return m.NextDecision, m.NextError
}

type manualTTicker struct {
	Started bool
}

var _ tTicker = (*manualTTicker)(nil)

func (t *manualTTicker) Start(d time.Duration) {
	t.Started = true
}
func (t *manualTTicker) OnTick() {
}
func (t *manualTTicker) Stop() {
}

type syncIDBatcher struct {
	sync.Mutex
	openBatch idbatcher.Batch
	batchPipe chan idbatcher.Batch
}

var _ (idbatcher.Batcher) = (*syncIDBatcher)(nil)

func newSyncIDBatcher(numBatches uint64) idbatcher.Batcher {
	batches := make(chan idbatcher.Batch, numBatches)
	for i := uint64(0); i < numBatches; i++ {
		batches <- nil
	}
	return &syncIDBatcher{
		batchPipe: batches,
	}
}
func (s *syncIDBatcher) AddToCurrentBatch(id idbatcher.ID) {
	s.Lock()
	s.openBatch = append(s.openBatch, id)
	s.Unlock()
}
func (s *syncIDBatcher) CloseCurrentAndTakeFirstBatch() (idbatcher.Batch, bool) {
	s.Lock()
	defer s.Unlock()
	firstBatch := <-s.batchPipe
	s.batchPipe <- s.openBatch
	s.openBatch = nil
	return firstBatch, true
}
func (s *syncIDBatcher) Stop() {
}

type mockSpanProcessor struct {
	TotalSpans int
}

var _ processor.TraceProcessor = &mockSpanProcessor{}

func (p *mockSpanProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	batchSize := len(td.Spans)
	p.TotalSpans += batchSize
	return nil
}

func (p *mockSpanProcessor) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

// Shutdown is invoked during service shutdown.
func (p *mockSpanProcessor) Shutdown() error {
	return nil
}
