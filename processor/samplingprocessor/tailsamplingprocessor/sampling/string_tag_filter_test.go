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
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestStringTagFilter(t *testing.T) {

	var empty = map[string]string{}
	filter := NewStringAttributeFilter(zap.NewNop(), "example", []string{"value"})

	cases := []struct {
		Desc     string
		Trace    *TraceData
		Decision Decision
	}{
		{
			Desc:     "nonmatching node attribute key",
			Trace:    newTraceStringAttrs(map[string]string{"non_matching": "value"}, nil),
			Decision: NotSampled,
		},
		{
			Desc:     "nonmatching node attribute value",
			Trace:    newTraceStringAttrs(map[string]string{"example": "non_matching"}, nil),
			Decision: NotSampled,
		},
		{
			Desc:     "matching node attribute",
			Trace:    newTraceStringAttrs(map[string]string{"example": "value"}, nil),
			Decision: Sampled,
		},
		{
			Desc:     "nonmatching span attribute key",
			Trace:    newTraceStringAttrs(empty, newSpan("nonmatching", "value")),
			Decision: NotSampled,
		},
		{
			Desc:     "nonmatching span attribute value",
			Trace:    newTraceStringAttrs(empty, newSpan("example", "nonmatching")),
			Decision: NotSampled,
		},
		{
			Desc:     "matching span attribute",
			Trace:    newTraceStringAttrs(empty, newSpan("example", "value")),
			Decision: Sampled,
		},
	}

	for _, c := range cases {
		t.Run(c.Desc, func(t *testing.T) {
			u, _ := uuid.NewRandom()
			decision, err := filter.Evaluate(pdata.NewTraceID(u[:]), c.Trace)
			assert.NoError(t, err)
			assert.Equal(t, decision, c.Decision)
		})
	}
}

func newSpan(attrKey string, attrValue string) *tracepb.Span {
	return &tracepb.Span{
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: map[string]*tracepb.AttributeValue{
				attrKey: {
					Value: &tracepb.AttributeValue_StringValue{
						StringValue: &tracepb.TruncatableString{
							Value: attrValue,
						},
					},
				},
			},
		},
	}
}

func newTraceStringAttrs(nodeAttrs map[string]string, span *tracepb.Span) *TraceData {
	return &TraceData{
		ReceivedBatches: []consumerdata.TraceData{
			{
				Node: &commonpb.Node{
					Attributes: nodeAttrs,
				},
				Spans: []*tracepb.Span{
					span,
				},
			},
		},
	}
}
