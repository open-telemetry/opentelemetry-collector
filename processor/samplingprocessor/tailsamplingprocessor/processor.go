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
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor/idbatcher"
	"github.com/open-telemetry/opentelemetry-collector/processor/samplingprocessor/tailsamplingprocessor/sampling"
)

// Policy combines a sampling policy evaluator with the destinations to be
// used for that policy.
type Policy struct {
	// Name used to identify this policy instance.
	Name string
	// Evaluator that decides if a trace is sampled or not by this policy instance.
	Evaluator sampling.PolicyEvaluator
	// ctx used to carry metric tags of each policy.
	ctx context.Context
}

// traceKey is defined since sync.Map requires a comparable type, isolating it on its own
// type to help track usage.
type traceKey string

// tailSamplingSpanProcessor handles the incoming trace data and uses the given sampling
// policy to sample traces.
type tailSamplingSpanProcessor struct {
	ctx             context.Context
	nextConsumer    consumer.TraceConsumer
	start           sync.Once
	maxNumTraces    uint64
	policies        []*Policy
	logger          *zap.Logger
	idToTrace       sync.Map
	policyTicker    tTicker
	decisionBatcher idbatcher.Batcher
	deleteChan      chan traceKey
	numTracesOnMap  uint64
}

const (
	sourceFormat = "tail_sampling"
)

var _ processor.TraceProcessor = (*tailSamplingSpanProcessor)(nil)

// NewTraceProcessor returns a processor.TraceProcessor that will perform tail sampling according to the given
// configuration.
func NewTraceProcessor(logger *zap.Logger, nextConsumer consumer.TraceConsumer, cfg Config) (processor.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}

	numDecisionBatches := uint64(cfg.DecisionWait.Seconds())
	inBatcher, err := idbatcher.New(numDecisionBatches, cfg.ExpectedNewTracesPerSec, uint64(2*runtime.NumCPU()))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	policies := []*Policy{}
	for i := range cfg.PolicyCfgs {
		policyCfg := &cfg.PolicyCfgs[i]
		policyCtx, err := tag.New(ctx, tag.Upsert(tagPolicyKey, policyCfg.Name), tag.Upsert(observability.TagKeyReceiver, sourceFormat))
		if err != nil {
			return nil, err
		}
		eval, err := getPolicyEvaluator(policyCfg)
		if err != nil {
			return nil, err
		}
		policy := &Policy{
			Name:      policyCfg.Name,
			Evaluator: eval,
			ctx:       policyCtx,
		}
		policies = append(policies, policy)
	}

	tsp := &tailSamplingSpanProcessor{
		ctx:             ctx,
		nextConsumer:    nextConsumer,
		maxNumTraces:    cfg.NumTraces,
		logger:          logger,
		decisionBatcher: inBatcher,
		policies:        policies,
	}

	tsp.policyTicker = &policyTicker{onTick: tsp.samplingPolicyOnTick}
	tsp.deleteChan = make(chan traceKey, cfg.NumTraces)

	return tsp, nil
}

func getPolicyEvaluator(cfg *PolicyCfg) (sampling.PolicyEvaluator, error) {
	switch cfg.Type {
	case AlwaysSample:
		return sampling.NewAlwaysSample(), nil
	case NumericAttribute:
		nafCfg := cfg.NumericAttributeCfg
		return sampling.NewNumericAttributeFilter(nafCfg.Key, nafCfg.MinValue, nafCfg.MaxValue), nil
	case StringAttribute:
		safCfg := cfg.StringAttributeCfg
		return sampling.NewStringAttributeFilter(safCfg.Key, safCfg.Values), nil
	case RateLimiting:
		rlfCfg := cfg.RateLimitingCfg
		return sampling.NewRateLimiting(rlfCfg.SpansPerSecond), nil
	default:
		return nil, fmt.Errorf("unknown sampling policy type %s", cfg.Type)
	}
}

func (tsp *tailSamplingSpanProcessor) samplingPolicyOnTick() {
	var idNotFoundOnMapCount, evaluateErrorCount, decisionSampled, decisionNotSampled int64
	startTime := time.Now()
	batch, _ := tsp.decisionBatcher.CloseCurrentAndTakeFirstBatch()
	batchLen := len(batch)
	tsp.logger.Debug("Sampling Policy Evaluation ticked")
	for _, id := range batch {
		d, ok := tsp.idToTrace.Load(traceKey(id))
		if !ok {
			idNotFoundOnMapCount++
			continue
		}
		trace := d.(*sampling.TraceData)
		trace.DecisionTime = time.Now()
		for i, policy := range tsp.policies {
			policyEvaluateStartTime := time.Now()
			decision, err := policy.Evaluator.Evaluate(id, trace)
			stats.Record(
				policy.ctx,
				statDecisionLatencyMicroSec.M(int64(time.Since(policyEvaluateStartTime)/time.Microsecond)))
			if err != nil {
				trace.Decisions[i] = sampling.NotSampled
				evaluateErrorCount++
				tsp.logger.Error("Sampling policy error", zap.Error(err))
				continue
			}

			trace.Decisions[i] = decision

			switch decision {
			case sampling.Sampled:
				stats.RecordWithTags(
					policy.ctx,
					[]tag.Mutator{tag.Insert(tagSampledKey, "true")},
					statCountTracesSampled.M(int64(1)),
				)
				decisionSampled++

				trace.Lock()
				traceBatches := trace.ReceivedBatches
				trace.Unlock()

				for j := 0; j < len(traceBatches); j++ {
					tsp.nextConsumer.ConsumeTraceData(policy.ctx, traceBatches[j])
				}
			case sampling.NotSampled:
				stats.RecordWithTags(
					policy.ctx,
					[]tag.Mutator{tag.Insert(tagSampledKey, "false")},
					statCountTracesSampled.M(int64(1)),
				)
				decisionNotSampled++
			}
		}

		// Sampled or not, remove the batches
		trace.Lock()
		trace.ReceivedBatches = nil
		trace.Unlock()
	}

	stats.Record(tsp.ctx,
		statOverallDecisionLatencyµs.M(int64(time.Since(startTime)/time.Microsecond)),
		statDroppedTooEarlyCount.M(idNotFoundOnMapCount),
		statPolicyEvaluationErrorCount.M(evaluateErrorCount),
		statTracesOnMemoryGauge.M(int64(atomic.LoadUint64(&tsp.numTracesOnMap))))

	tsp.logger.Debug("Sampling policy evaluation completed",
		zap.Int("batch.len", batchLen),
		zap.Int64("sampled", decisionSampled),
		zap.Int64("notSampled", decisionNotSampled),
		zap.Int64("droppedPriorToEvaluation", idNotFoundOnMapCount),
		zap.Int64("policyEvaluationErrors", evaluateErrorCount),
	)
}

// ConsumeTraceData is required by the SpanProcessor interface.
func (tsp *tailSamplingSpanProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	tsp.start.Do(func() {
		tsp.logger.Info("First trace data arrived, starting tail_sampling timers")
		tsp.policyTicker.Start(1 * time.Second)
	})

	// Groupd spans per their traceId to minimize contention on idToTrace
	idToSpans := make(map[traceKey][]*tracepb.Span)
	for _, span := range td.Spans {
		if len(span.TraceId) != 16 {
			tsp.logger.Warn("Span without valid TraceId", zap.String("SourceFormat", td.SourceFormat))
			continue
		}
		traceKey := traceKey(span.TraceId)
		idToSpans[traceKey] = append(idToSpans[traceKey], span)
	}

	var newTraceIDs int64
	singleTrace := len(idToSpans) == 1
	for id, spans := range idToSpans {
		lenSpans := int64(len(spans))
		lenPolicies := len(tsp.policies)
		initialDecisions := make([]sampling.Decision, lenPolicies)
		for i := 0; i < lenPolicies; i++ {
			initialDecisions[i] = sampling.Pending
		}
		initialTraceData := &sampling.TraceData{
			Decisions:   initialDecisions,
			ArrivalTime: time.Now(),
			SpanCount:   lenSpans,
		}
		d, loaded := tsp.idToTrace.LoadOrStore(traceKey(id), initialTraceData)

		actualData := d.(*sampling.TraceData)
		if loaded {
			atomic.AddInt64(&actualData.SpanCount, lenSpans)
		} else {
			newTraceIDs++
			tsp.decisionBatcher.AddToCurrentBatch([]byte(id))
			atomic.AddUint64(&tsp.numTracesOnMap, 1)
			postDeletion := false
			currTime := time.Now()
			for !postDeletion {
				select {
				case tsp.deleteChan <- id:
					postDeletion = true
				default:
					traceKeyToDrop := <-tsp.deleteChan
					tsp.dropTrace(traceKeyToDrop, currTime)
				}
			}
		}

		for i, policy := range tsp.policies {
			actualData.Lock()
			actualDecision := actualData.Decisions[i]
			// If decision is pending, we want to add the new spans still under the lock, so the decision doesn't happen
			// in between the transition from pending.
			if actualDecision == sampling.Pending {
				// Add the spans to the trace, but only once for all policy, otherwise same spans will
				// be duplicated in the final trace.
				traceTd := prepareTraceBatch(spans, singleTrace, td)
				actualData.ReceivedBatches = append(actualData.ReceivedBatches, traceTd)
				actualData.Unlock()
				break
			}
			actualData.Unlock()

			switch actualDecision {
			case sampling.Pending:
				// All process for pending done above, keep the case so it doesn't go to default.
			case sampling.Sampled:
				// Forward the spans to the policy destinations
				traceTd := prepareTraceBatch(spans, singleTrace, td)
				if err := tsp.nextConsumer.ConsumeTraceData(policy.ctx, traceTd); err != nil {
					tsp.logger.Warn("Error sending late arrived spans to destination",
						zap.String("policy", policy.Name),
						zap.Error(err))
				}
				fallthrough // so OnLateArrivingSpans is also called for decision Sampled.
			case sampling.NotSampled:
				policy.Evaluator.OnLateArrivingSpans(actualDecision, spans)
				stats.Record(tsp.ctx, statLateSpanArrivalAfterDecision.M(int64(time.Since(actualData.DecisionTime)/time.Second)))

			default:
				tsp.logger.Warn("Encountered unexpected sampling decision",
					zap.String("policy", policy.Name),
					zap.Int("decision", int(actualDecision)))
			}
		}
	}

	stats.Record(tsp.ctx, statNewTraceIDReceivedCount.M(newTraceIDs))
	return nil
}

func (tsp *tailSamplingSpanProcessor) GetCapabilities() processor.Capabilities {
	return processor.Capabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (tsp *tailSamplingSpanProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (tsp *tailSamplingSpanProcessor) Shutdown() error {
	return nil
}

func (tsp *tailSamplingSpanProcessor) dropTrace(traceID traceKey, deletionTime time.Time) {
	var trace *sampling.TraceData
	if d, ok := tsp.idToTrace.Load(traceID); ok {
		trace = d.(*sampling.TraceData)
		tsp.idToTrace.Delete(traceID)
		// Subtract one from numTracesOnMap per https://godoc.org/sync/atomic#AddUint64
		atomic.AddUint64(&tsp.numTracesOnMap, ^uint64(0))
	}
	if trace == nil {
		tsp.logger.Error("Attempt to delete traceID not on table")
		return
	}
	policiesLen := len(tsp.policies)
	stats.Record(tsp.ctx, statTraceRemovalAgeSec.M(int64(deletionTime.Sub(trace.ArrivalTime)/time.Second)))
	for j := 0; j < policiesLen; j++ {
		if trace.Decisions[j] == sampling.Pending {
			policy := tsp.policies[j]
			if decision, err := policy.Evaluator.OnDroppedSpans([]byte(traceID), trace); err != nil {
				tsp.logger.Warn("OnDroppedSpans",
					zap.String("policy", policy.Name),
					zap.Int("decision", int(decision)),
					zap.Error(err))
			}
		}
	}
}

func prepareTraceBatch(spans []*tracepb.Span, singleTrace bool, td consumerdata.TraceData) consumerdata.TraceData {
	var traceTd consumerdata.TraceData
	if singleTrace {
		// Special case no need to prepare a batch
		traceTd = td
	} else {
		traceTd = consumerdata.TraceData{
			Node:     td.Node,
			Resource: td.Resource,
			Spans:    spans,
		}
	}
	return traceTd
}

// tTicker interface allows easier testing of ticker related functionality used by tailSamplingProcessor
type tTicker interface {
	// Start sets the frequency of the ticker and starts the periodic calls to OnTick.
	Start(d time.Duration)
	// OnTick is called when the ticker fires.
	OnTick()
	// Stops firing the ticker.
	Stop()
}

type policyTicker struct {
	ticker *time.Ticker
	onTick func()
}

func (pt *policyTicker) Start(d time.Duration) {
	pt.ticker = time.NewTicker(d)
	go func() {
		for range pt.ticker.C {
			pt.OnTick()
		}
	}()
}
func (pt *policyTicker) OnTick() {
	pt.onTick()
}
func (pt *policyTicker) Stop() {
	pt.ticker.Stop()
}

var _ tTicker = (*policyTicker)(nil)
