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
	"testing"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestZipkinSpansToInternalTraces(t *testing.T) {
	tests := []struct {
		name string
		zs   []*zipkinmodel.SpanModel
		td   pdata.Traces
		err  error
	}{
		{
			name: "empty",
			zs:   make([]*zipkinmodel.SpanModel, 0),
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},
		{
			name: "nilSpan",
			zs:   generateNilSpan(),
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},
		{
			name: "minimalSpan",
			zs:   generateSpanNoEndpoints(),
			td:   generateTraceSingleSpanNoResourceOrInstrLibrary(),
			err:  nil,
		},
		{
			name: "onlyLocalEndpointSpan",
			zs:   generateSpanNoTags(),
			td:   generateTraceSingleSpanMinmalResource(),
			err:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			td, err := V2SpansToInternalTraces(test.zs, false)
			assert.EqualValues(t, test.err, err)
			if test.name != "nilSpan" {
				assert.Equal(t, len(test.zs), td.SpanCount())
			}
			assert.EqualValues(t, test.td, td)
		})
	}
}

func generateNilSpan() []*zipkinmodel.SpanModel {
	return make([]*zipkinmodel.SpanModel, 1)
}

func generateSpanNoEndpoints() []*zipkinmodel.SpanModel {
	spans := make([]*zipkinmodel.SpanModel, 1)
	spans[0] = &zipkinmodel.SpanModel{
		SpanContext: zipkinmodel.SpanContext{
			TraceID: convertTraceID(
				pdata.NewTraceID([16]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})),
			ID: convertSpanID(pdata.NewSpanID([8]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8})),
		},
		Name:           "MinimalData",
		Kind:           zipkinmodel.Client,
		Timestamp:      time.Unix(1596911098, 294000000),
		Duration:       1000000,
		Shared:         false,
		LocalEndpoint:  nil,
		RemoteEndpoint: nil,
		Annotations:    nil,
		Tags:           nil,
	}
	return spans
}

func generateSpanNoTags() []*zipkinmodel.SpanModel {
	spans := generateSpanNoEndpoints()
	spans[0].LocalEndpoint = &zipkinmodel.Endpoint{ServiceName: "SoleAttr"}
	return spans
}

func generateTraceSingleSpanNoResourceOrInstrLibrary() pdata.Traces {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	rs := td.ResourceSpans().At(0)
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(
		pdata.NewTraceID([16]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8}))
	span.SetName("MinimalData")
	span.SetKind(pdata.SpanKindCLIENT)
	span.SetStartTime(1596911098294000000)
	span.SetEndTime(1596911098295000000)
	span.Attributes().InitEmptyWithCapacity(0)
	return td
}

func generateTraceSingleSpanMinmalResource() pdata.Traces {
	td := generateTraceSingleSpanNoResourceOrInstrLibrary()
	rs := td.ResourceSpans().At(0)
	rsc := rs.Resource()
	rsc.Attributes().InitEmptyWithCapacity(1)
	rsc.Attributes().UpsertString(conventions.AttributeServiceName, "SoleAttr")
	return td
}
