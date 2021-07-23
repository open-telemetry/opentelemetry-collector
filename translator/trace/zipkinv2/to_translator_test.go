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
	"testing"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.opentelemetry.io/collector/translator/trace/internal/zipkin"
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
			td:   pdata.NewTraces(),
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
		{
			name: "errorTag",
			zs:   generateSpanErrorTags(),
			td:   generateTraceSingleSpanErrorStatus(),
			err:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			td, err := ToTranslator{}.ToTraces(test.zs)
			assert.EqualValues(t, test.err, err)
			if test.name != "nilSpan" {
				assert.Equal(t, len(test.zs), td.SpanCount())
			}
			assert.EqualValues(t, test.td, td)
		})
	}
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

func generateSpanErrorTags() []*zipkinmodel.SpanModel {
	errorTags := make(map[string]string)
	errorTags["error"] = "true"

	spans := generateSpanNoEndpoints()
	spans[0].Tags = errorTags
	return spans
}

func generateTraceSingleSpanNoResourceOrInstrLibrary() pdata.Traces {
	td := pdata.NewTraces()
	span := td.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(
		pdata.NewTraceID([16]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8}))
	span.SetName("MinimalData")
	span.SetKind(pdata.SpanKindClient)
	span.SetStartTimestamp(1596911098294000000)
	span.SetEndTimestamp(1596911098295000000)
	return td
}

func generateTraceSingleSpanMinmalResource() pdata.Traces {
	td := generateTraceSingleSpanNoResourceOrInstrLibrary()
	rs := td.ResourceSpans().At(0)
	rsc := rs.Resource()
	rsc.Attributes().UpsertString(conventions.AttributeServiceName, "SoleAttr")
	return td
}

func generateTraceSingleSpanErrorStatus() pdata.Traces {
	td := pdata.NewTraces()
	span := td.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(
		pdata.NewTraceID([16]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80}))
	span.SetSpanID(pdata.NewSpanID([8]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8}))
	span.SetName("MinimalData")
	span.SetKind(pdata.SpanKindClient)
	span.SetStartTimestamp(1596911098294000000)
	span.SetEndTimestamp(1596911098295000000)
	span.Status().SetCode(pdata.StatusCodeError)
	return td
}

func TestV2SpanWithoutTimestampGetsTag(t *testing.T) {
	duration := int64(2948533333)
	spans := make([]*zipkinmodel.SpanModel, 1)
	spans[0] = &zipkinmodel.SpanModel{
		SpanContext: zipkinmodel.SpanContext{
			TraceID: convertTraceID(
				pdata.NewTraceID([16]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})),
			ID: convertSpanID(pdata.NewSpanID([8]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8})),
		},
		Name:           "NoTimestamps",
		Kind:           zipkinmodel.Client,
		Duration:       time.Duration(duration),
		Shared:         false,
		LocalEndpoint:  nil,
		RemoteEndpoint: nil,
		Annotations:    nil,
		Tags:           nil,
	}

	gb, err := ToTranslator{}.ToTraces(spans)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	gs := gb.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
	assert.NotNil(t, gs.StartTimestamp)
	assert.NotNil(t, gs.EndTimestamp)

	// expect starttime to be set to zero (unix time)
	unixTime := gs.StartTimestamp().AsTime().Unix()
	assert.Equal(t, int64(0), unixTime)

	// expect end time to be zero (unix time) plus the duration
	assert.Equal(t, duration, gs.EndTimestamp().AsTime().UnixNano())

	wasAbsent, mapContainedKey := gs.Attributes().Get(zipkin.StartTimeAbsent)
	assert.True(t, mapContainedKey)
	assert.True(t, wasAbsent.BoolVal())
}
