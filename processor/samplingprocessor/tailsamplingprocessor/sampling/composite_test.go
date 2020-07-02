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
package sampling

import (
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/consumerdata"
)

type FakeTimeProvider struct {
	second int64
}

func (f FakeTimeProvider) getCurSecond() int64 {
	return f.second
}

var traceID = []byte{1, 2, 3}

func createTrace() *TraceData {
	var span []*tracepb.Span

	receivedBatches := []consumerdata.TraceData{{
		Spans: span,
	}}
	trace := &TraceData{SpanCount: 1}
	trace.ReceivedBatches = receivedBatches

	return trace
}

func TestCompositeEvaluatorNotSampled(t *testing.T) {

	// Create 2 policies which do not match any trace
	n1 := NewNumericAttributeFilter(zap.NewNop(), "tag", 0, 100)
	n2 := NewNumericAttributeFilter(zap.NewNop(), "tag", 200, 300)
	c := NewComposite(zap.NewNop(), 1000, []SubPolicyEvalParams{{n1, 100}, {n2, 100}}, FakeTimeProvider{})

	trace := createTrace()

	decision, err := c.Evaluate(traceID, trace)
	if err != nil {
		t.Fatalf("Failed to evaluate composite policy: %v", err)
	}

	// None of the numeric filters should match since input trace data does not contain
	// the "tag", so the decision should be NotSampled.
	expected := NotSampled
	if decision != expected {
		t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
	}
}

func TestCompositeEvaluatorSampled(t *testing.T) {

	// Create 2 subpolicies. First results in 100% NotSampled, the second in 100% Sampled.
	n1 := NewNumericAttributeFilter(zap.NewNop(), "tag", 0, 100)
	n2 := NewAlwaysSample(zap.NewNop())
	c := NewComposite(zap.NewNop(), 1000, []SubPolicyEvalParams{{n1, 100}, {n2, 100}}, FakeTimeProvider{})

	trace := createTrace()

	decision, err := c.Evaluate(traceID, trace)
	if err != nil {
		t.Fatalf("Failed to evaluate composite policy: %v", err)
	}

	// The second policy is AlwaysSample, so the decision should be Sampled.
	expected := Sampled
	if decision != expected {
		t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
	}
}

func TestCompositeEvaluatorThrottling(t *testing.T) {

	// Create only one subpolicy, with 100% Sampled policy.
	n1 := NewAlwaysSample(zap.NewNop())
	timeProvider := &FakeTimeProvider{second: 0}
	const totalSPS = 100
	c := NewComposite(zap.NewNop(), totalSPS, []SubPolicyEvalParams{{n1, totalSPS}}, timeProvider)

	trace := createTrace()

	// First totalSPS traces should be 100% Sampled
	for i := 0; i < totalSPS; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := Sampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}

	// Now we hit the rate limit, so subsequent evaluations should result in 100% NotSampled
	for i := 0; i < totalSPS; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := NotSampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}

	// Let the time advance by one second.
	timeProvider.second = timeProvider.second + 1

	// Subsequent sampling should be Sampled again because it is a new second.
	for i := 0; i < totalSPS; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := Sampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}
}

func TestCompositeEvaluator2SubpolicyThrottling(t *testing.T) {

	n1 := NewNumericAttributeFilter(zap.NewNop(), "tag", 0, 100)
	n2 := NewAlwaysSample(zap.NewNop())
	timeProvider := &FakeTimeProvider{second: 0}
	const totalSPS = 100
	c := NewComposite(zap.NewNop(), totalSPS, []SubPolicyEvalParams{{n1, totalSPS / 2}, {n2, totalSPS / 2}}, timeProvider)

	trace := createTrace()

	// We have 2 subpolicies, so each should initially get half the bandwidth

	// First totalSPS/2 should be Sampled until we hit the rate limit
	for i := 0; i < totalSPS/2; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := Sampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}

	// Now we hit the rate limit for second subpolicy, so subsequent evaluations should result in NotSampled
	for i := 0; i < totalSPS/2; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := NotSampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}

	// Let the time advance by one second.
	timeProvider.second = timeProvider.second + 1

	// It is a new second, so we should start sampling again.
	for i := 0; i < totalSPS/2; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := Sampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}

	// Now let's hit the hard limit and exceed the total by a factor of 2
	for i := 0; i < 2*totalSPS; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := NotSampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}

	// Let the time advance by one second.
	timeProvider.second = timeProvider.second + 1

	// It is a new second, so we should start sampling again.
	for i := 0; i < totalSPS/2; i++ {
		decision, err := c.Evaluate(traceID, trace)
		if err != nil {
			t.Fatalf("Failed to evaluate composite policy: %v", err)
		}

		expected := Sampled
		if decision != expected {
			t.Fatalf("Incorrect decision by composite policy evaluator: expected %v, actual %v", expected, decision)
		}
	}
}
