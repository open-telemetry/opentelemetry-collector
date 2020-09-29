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
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type rateLimiting struct {
	currentSecond        int64
	spansInCurrentSecond int64
	spansPerSecond       int64
	logger               *zap.Logger
}

var _ PolicyEvaluator = (*rateLimiting)(nil)

// NewRateLimiting creates a policy evaluator the samples all traces.
func NewRateLimiting(logger *zap.Logger, spansPerSecond int64) PolicyEvaluator {
	return &rateLimiting{
		spansPerSecond: spansPerSecond,
		logger:         logger,
	}
}

// OnLateArrivingSpans notifies the evaluator that the given list of spans arrived
// after the sampling decision was already taken for the trace.
// This gives the evaluator a chance to log any message/metrics and/or update any
// related internal state.
func (r *rateLimiting) OnLateArrivingSpans(Decision, []*tracepb.Span) error {
	r.logger.Debug("Triggering action for late arriving spans in rate-limiting filter")
	return nil
}

// Evaluate looks at the trace data and returns a corresponding SamplingDecision.
func (r *rateLimiting) Evaluate(_ pdata.TraceID, trace *TraceData) (Decision, error) {
	r.logger.Debug("Evaluating spans in rate-limiting filter")
	currSecond := time.Now().Unix()
	if r.currentSecond != currSecond {
		r.currentSecond = currSecond
		r.spansInCurrentSecond = 0
	}

	spansInSecondIfSampled := r.spansInCurrentSecond + trace.SpanCount
	if spansInSecondIfSampled < r.spansPerSecond {
		r.spansInCurrentSecond = spansInSecondIfSampled
		return Sampled, nil
	}

	return NotSampled, nil
}

// OnDroppedSpans is called when the trace needs to be dropped, due to memory
// pressure, before the decision_wait time has been reached.
func (r *rateLimiting) OnDroppedSpans(pdata.TraceID, *TraceData) (Decision, error) {
	r.logger.Debug("Triggering action for dropped spans in rate-limiting filter")
	return Sampled, nil
}
