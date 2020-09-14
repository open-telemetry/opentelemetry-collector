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

package sampling

import (
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type numericAttributeFilter struct {
	key                string
	minValue, maxValue int64
	logger             *zap.Logger
}

var _ PolicyEvaluator = (*numericAttributeFilter)(nil)

// NewNumericAttributeFilter creates a policy evaluator that samples all traces with
// the given attribute in the given numeric range.
func NewNumericAttributeFilter(logger *zap.Logger, key string, minValue, maxValue int64) PolicyEvaluator {
	return &numericAttributeFilter{
		key:      key,
		minValue: minValue,
		maxValue: maxValue,
		logger:   logger,
	}
}

// OnLateArrivingSpans notifies the evaluator that the given list of spans arrived
// after the sampling decision was already taken for the trace.
// This gives the evaluator a chance to log any message/metrics and/or update any
// related internal state.
func (naf *numericAttributeFilter) OnLateArrivingSpans(Decision, []*tracepb.Span) error {
	naf.logger.Debug("Triggering action for late arriving spans in numeric-attribute filter")
	return nil
}

// Evaluate looks at the trace data and returns a corresponding SamplingDecision.
func (naf *numericAttributeFilter) Evaluate(_ pdata.TraceID, trace *TraceData) (Decision, error) {
	naf.logger.Debug("Evaluating spans in numeric-attribute filter")
	trace.Lock()
	batches := trace.ReceivedBatches
	trace.Unlock()
	for _, batch := range batches {
		for _, span := range batch.Spans {
			if span == nil || span.Attributes == nil {
				continue
			}
			if v, ok := span.Attributes.AttributeMap[naf.key]; ok {
				value := v.GetIntValue()
				if value >= naf.minValue && value <= naf.maxValue {
					return Sampled, nil
				}
			}
		}
	}

	return NotSampled, nil
}

// OnDroppedSpans is called when the trace needs to be dropped, due to memory
// pressure, before the decision_wait time has been reached.
func (naf *numericAttributeFilter) OnDroppedSpans(pdata.TraceID, *TraceData) (Decision, error) {
	naf.logger.Debug("Triggering action for dropped spans in numeric-attribute filter")
	return NotSampled, nil
}
