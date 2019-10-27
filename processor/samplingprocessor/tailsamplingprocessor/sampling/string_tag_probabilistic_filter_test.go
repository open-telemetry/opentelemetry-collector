// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sampling

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestEvaluate_TenPercent(t *testing.T) {
	var status = "200"
	policies := []string{status + "::0.1"}
	var stringProbabilisticAttributeFilter = NewStringProbabilisticAttributeFilter(string("http.status_code"), policies)
	j := 0
	for i := 0; i < 100; i++ {
		j = evaluateStringProbabilityAttrib(stringProbabilisticAttributeFilter, j, status)
	}
	assert.Equal(t, 10, j)
}

func TestEvaluate_HundredPercent(t *testing.T) {
	var status = "500"
	policies := []string{status + "::1.0"}
	var stringProbabilisticAttributeFilter = NewStringProbabilisticAttributeFilter(string("http.status_code"), policies)
	j := 0
	for i := 0; i < 100; i++ {
		j = evaluateStringProbabilityAttrib(stringProbabilisticAttributeFilter, j, status)
	}
	assert.Equal(t, 100, j)
}

func TestEvaluate_NodeAttribute(t *testing.T) {
	var status = "500"
	policies := []string{status + "::1.0"}
	var stringProbabilisticAttributeFilter = NewStringProbabilisticAttributeFilter(string("http.status_code"), policies)
	r := rand.New(rand.NewSource(1))
	traceKey := tracetranslator.UInt64ToByteTraceID(r.Uint64(), r.Uint64())
	tdd := genRandomTestData(1, 1, "test", status, traceKey)
	tdd[0].Node.Attributes = make(map[string]string)
	tdd[0].Node.Attributes["http.status_code"] = status
	td := TraceData{
		Mutex:           sync.Mutex{},
		Decisions:       nil,
		ArrivalTime:     time.Time{},
		DecisionTime:    time.Time{},
		SpanCount:       5,
		ReceivedBatches: tdd,
	}
	decision, _ := stringProbabilisticAttributeFilter.Evaluate(traceKey, &td)
	assert.Equal(t, decision, Sampled)
}

func TestOnDroppedSpans(t *testing.T) {
	var status = "500"
	policies := []string{status + "::1.0"}
	var stringProbabilisticAttributeFilter = NewStringProbabilisticAttributeFilter(string("http.status_code"), policies)
	decision, _ := stringProbabilisticAttributeFilter.OnDroppedSpans(nil, nil)
	assert.Equal(t, decision, NotSampled)
}

func TestOnLateArrivingSpans(t *testing.T) {
	var status = "500"
	policies := []string{status + "::1.0"}
	var stringProbabilisticAttributeFilter = NewStringProbabilisticAttributeFilter(string("http.status_code"), policies)
	error := stringProbabilisticAttributeFilter.OnLateArrivingSpans(Decision(NotSampled), nil)
	assert.Equal(t, nil, error)
}

func evaluateStringProbabilityAttrib(filter PolicyEvaluator, j int, status string) int {
	r := rand.New(rand.NewSource(1))
	traceKey := tracetranslator.UInt64ToByteTraceID(r.Uint64(), r.Uint64())
	traceData := genRandomTestData(1, 1, "test", status, traceKey)
	td := TraceData{
		Mutex:           sync.Mutex{},
		Decisions:       nil,
		ArrivalTime:     time.Time{},
		DecisionTime:    time.Time{},
		SpanCount:       5,
		ReceivedBatches: traceData,
	}
	decision, _ := filter.Evaluate(traceKey, &td)
	if decision == Sampled {
		j++
	}
	return j
}

func genRandomTestData(numBatches, numTracesPerBatch int, serviceName string, status string, traceKey []byte) (tdd []consumerdata.TraceData) {
	for i := 0; i < numBatches; i++ {
		var spans []*tracepb.Span
		for j := 0; j < numTracesPerBatch; j++ {
			span := &tracepb.Span{
				TraceId: traceKey,
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			}
			span.Attributes.AttributeMap["http.status_code"] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{
					StringValue: &tracepb.TruncatableString{Value: status},
				},
			}
			spans = append(spans, span)
		}
		td := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: serviceName},
			},
			Spans: spans,
		}
		tdd = append(tdd, td)
	}
	return tdd
}
