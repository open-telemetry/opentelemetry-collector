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

package zipkinv2

import (
	"errors"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestInternalTracesToZipkinSpans(t *testing.T) {
	tests := []struct {
		name string
		td   pdata.Traces
		zs   []*zipkinmodel.SpanModel
		err  error
	}{
		{
			name: "empty",
			td:   pdata.NewTraces(),
			err:  nil,
		},
		{
			name: "oneEmpty",
			td:   testdata.GenerateTracesOneEmptyResourceSpans(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "noLibs",
			td:   testdata.GenerateTracesNoLibraries(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneEmptyLib",
			td:   testdata.GenerateTracesOneEmptyInstrumentationLibrary(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneSpanNoResrouce",
			td:   testdata.GenerateTracesOneSpanNoResource(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  errors.New("TraceID is invalid"),
		},
		{
			name: "oneSpan",
			td:   generateTraceOneSpanOneTraceID(),
			zs:   []*zipkinmodel.SpanModel{zipkinOneSpan()},
			err:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spans, err := FromTranslator{}.FromTraces(test.td)
			assert.EqualValues(t, test.err, err)
			if test.name == "empty" {
				assert.Nil(t, spans)
			} else {
				assert.Equal(t, len(test.zs), len(spans))
				assert.EqualValues(t, test.zs, spans)
			}
		})
	}
}

func TestInternalTracesToZipkinSpansAndBack(t *testing.T) {
	tds, err := goldendataset.GenerateTraces(
		"../../../internal/goldendataset/testdata/generated_pict_pairs_traces.txt",
		"../../../internal/goldendataset/testdata/generated_pict_pairs_spans.txt")
	assert.NoError(t, err)
	for _, td := range tds {
		zipkinSpans, err := FromTranslator{}.FromTraces(td)
		assert.NoError(t, err)
		assert.Equal(t, td.SpanCount(), len(zipkinSpans))
		tdFromZS, zErr := ToTranslator{}.ToTraces(zipkinSpans)
		assert.NoError(t, zErr, zipkinSpans)
		assert.NotNil(t, tdFromZS)
		assert.Equal(t, td.SpanCount(), tdFromZS.SpanCount())

		// check that all timestamps converted back and forth without change
		for i := 0; i < td.ResourceSpans().Len(); i++ {
			instSpans := td.ResourceSpans().At(i).InstrumentationLibrarySpans()
			for j := 0; j < instSpans.Len(); j++ {
				spans := instSpans.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)

					// search for the span with the same id to compare to
					spanFromZS := findSpanByID(tdFromZS.ResourceSpans(), span.SpanID())

					assert.Equal(t, span.StartTimestamp().AsTime().UnixNano(), spanFromZS.StartTimestamp().AsTime().UnixNano())
					assert.Equal(t, span.EndTimestamp().AsTime().UnixNano(), spanFromZS.EndTimestamp().AsTime().UnixNano())
				}
			}
		}
	}
}

func findSpanByID(rs pdata.ResourceSpansSlice, spanID pdata.SpanID) *pdata.Span {
	for i := 0; i < rs.Len(); i++ {
		instSpans := rs.At(i).InstrumentationLibrarySpans()
		for j := 0; j < instSpans.Len(); j++ {
			spans := instSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				if span.SpanID() == spanID {
					return &span
				}
			}
		}
	}
	return nil
}

func generateTraceOneSpanOneTraceID() pdata.Traces {
	td := testdata.GenerateTracesOneSpan()
	span := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
	span.SetTraceID(pdata.NewTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}))
	return td
}

func zipkinOneSpan() *zipkinmodel.SpanModel {
	trueBool := true
	return &zipkinmodel.SpanModel{
		SpanContext: zipkinmodel.SpanContext{
			TraceID:  zipkinmodel.TraceID{High: 72623859790382856, Low: 651345242494996240},
			ID:       72623859790382856,
			ParentID: nil,
			Debug:    false,
			Sampled:  &trueBool,
			Err:      errors.New("status-cancelled"),
		},
		LocalEndpoint: &zipkinmodel.Endpoint{
			ServiceName: "OTLPResourceNoServiceName",
		},
		RemoteEndpoint: nil,
		Annotations: []zipkinmodel.Annotation{
			{
				Timestamp: testdata.TestSpanEventTime,
				Value:     "event-with-attr|{\"span-event-attr\":\"span-event-attr-val\"}|2",
			},
			{
				Timestamp: testdata.TestSpanEventTime,
				Value:     "event|{}|2",
			},
		},
		Tags: map[string]string{
			"resource-attr":  "resource-attr-val-1",
			"status.code":    "STATUS_CODE_ERROR",
			"status.message": "status-cancelled",
		},
		Name:      "operationA",
		Timestamp: testdata.TestSpanStartTime,
		Duration:  1000000468,
		Shared:    false,
	}
}
