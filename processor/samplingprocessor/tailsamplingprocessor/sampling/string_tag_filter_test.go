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
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestStringTagFilter(t *testing.T) {

	var empty = map[string]pdata.AttributeValue{}
	filter := NewStringAttributeFilter(zap.NewNop(), "example", []string{"value"})

	cases := []struct {
		Desc     string
		Trace    *TraceData
		Decision Decision
	}{
		{
			Desc:     "nonmatching node attribute key",
			Trace:    newTraceStringAttrs(map[string]pdata.AttributeValue{"non_matching": pdata.NewAttributeValueString("value")}, "", ""),
			Decision: NotSampled,
		},
		{
			Desc:     "nonmatching node attribute value",
			Trace:    newTraceStringAttrs(map[string]pdata.AttributeValue{"example": pdata.NewAttributeValueString("non_matching")}, "", ""),
			Decision: NotSampled,
		},
		{
			Desc:     "matching node attribute",
			Trace:    newTraceStringAttrs(map[string]pdata.AttributeValue{"example": pdata.NewAttributeValueString("value")}, "", ""),
			Decision: Sampled,
		},
		{
			Desc:     "nonmatching span attribute key",
			Trace:    newTraceStringAttrs(empty, "nonmatching", "value"),
			Decision: NotSampled,
		},
		{
			Desc:     "nonmatching span attribute value",
			Trace:    newTraceStringAttrs(empty, "example", "nonmatching"),
			Decision: NotSampled,
		},
		{
			Desc:     "matching span attribute",
			Trace:    newTraceStringAttrs(empty, "example", "value"),
			Decision: Sampled,
		},
	}

	for _, c := range cases {
		t.Run(c.Desc, func(t *testing.T) {
			decision, err := filter.Evaluate(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}), c.Trace)
			assert.NoError(t, err)
			assert.Equal(t, decision, c.Decision)
		})
	}
}

func newTraceStringAttrs(nodeAttrs map[string]pdata.AttributeValue, spanAttrKey string, spanAttrValue string) *TraceData {
	var traceBatches []pdata.Traces
	traces := pdata.NewTraces()
	traces.ResourceSpans().Resize(1)
	rs := traces.ResourceSpans().At(0)
	rs.Resource().InitEmpty()
	rs.Resource().Attributes().InitFromMap(nodeAttrs)
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	attributes := make(map[string]pdata.AttributeValue)
	attributes[spanAttrKey] = pdata.NewAttributeValueString(spanAttrValue)
	span.Attributes().InitFromMap(attributes)
	traceBatches = append(traceBatches, traces)
	return &TraceData{
		ReceivedBatches: traceBatches,
	}
}

func TestOnDroppedSpans_StringAttribute(t *testing.T) {
	var empty = map[string]pdata.AttributeValue{}
	u, _ := uuid.NewRandom()
	filter := NewStringAttributeFilter(zap.NewNop(), "example", []string{"value"})
	decision, err := filter.OnDroppedSpans(pdata.NewTraceID(u), newTraceIntAttrs(empty, "example", math.MaxInt32+1))
	assert.Nil(t, err)
	assert.Equal(t, decision, NotSampled)
}

func TestOnLateArrivingSpans_StringAttribute(t *testing.T) {
	filter := NewStringAttributeFilter(zap.NewNop(), "example", []string{"value"})
	err := filter.OnLateArrivingSpans(NotSampled, nil)
	assert.Nil(t, err)
}
