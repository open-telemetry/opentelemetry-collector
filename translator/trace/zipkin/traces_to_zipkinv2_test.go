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

package zipkin

import (
	"errors"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/testdata"
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
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},
		{
			name: "oneEmpty",
			td:   testdata.GenerateTraceDataOneEmptyResourceSpans(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "noLibs",
			td:   testdata.GenerateTraceDataNoLibraries(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneEmptyLib",
			td:   testdata.GenerateTraceDataOneEmptyInstrumentationLibrary(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneSpanNoResrouce",
			td:   testdata.GenerateTraceDataOneSpanNoResource(),
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
			zss, err := InternalTracesToZipkinSpans(test.td)
			assert.EqualValues(t, test.err, err)
			if test.name == "empty" {
				assert.Nil(t, zss)
			} else {
				assert.Equal(t, len(test.zs), len(zss))
				assert.EqualValues(t, test.zs, zss)
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
		zipkinSpans, err := InternalTracesToZipkinSpans(td)
		assert.NoError(t, err)
		assert.Equal(t, td.SpanCount(), len(zipkinSpans))
		tdFromZS, zErr := V2SpansToInternalTraces(zipkinSpans, false)
		assert.NoError(t, zErr, zipkinSpans)
		assert.NotNil(t, tdFromZS)
		assert.Equal(t, td.SpanCount(), tdFromZS.SpanCount())
	}
}

func generateTraceOneSpanOneTraceID() pdata.Traces {
	td := testdata.GenerateTraceDataOneSpan()
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
