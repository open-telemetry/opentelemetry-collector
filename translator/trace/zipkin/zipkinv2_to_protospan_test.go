// Copyright 2019, OpenTelemetry Authors
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

package zipkin

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"testing"
	"time"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func TestShortIDSpanConversion(t *testing.T) {
	shortID, _ := zipkinmodel.TraceIDFromHex("0102030405060708")
	assert.Equal(t, uint64(0), shortID.High, "wanted 64bit traceID, so TraceID.High must be zero")

	zc := zipkinmodel.SpanContext{
		TraceID: shortID,
		ID:      zipkinmodel.ID(shortID.Low),
	}
	zs := zipkinmodel.SpanModel{
		SpanContext: zc,
	}

	ocSpan, _ := zipkinSpanToTraceSpan(&zs)
	require.Len(t, ocSpan.TraceId, 16, "incorrect OC proto trace id length")

	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8}
	assert.Equal(t, want, ocSpan.TraceId)
}

func TestV2SpanKindTranslation(t *testing.T) {
	tests := []struct {
		zipkinKind zipkinmodel.Kind
		ocKind     tracepb.Span_SpanKind
		otKind     tracetranslator.OpenTracingSpanKind
	}{
		{
			zipkinKind: zipkinmodel.Client,
			ocKind:     tracepb.Span_CLIENT,
			otKind:     "",
		},
		{
			zipkinKind: zipkinmodel.Server,
			ocKind:     tracepb.Span_SERVER,
			otKind:     "",
		},
		{
			zipkinKind: zipkinmodel.Producer,
			ocKind:     tracepb.Span_SPAN_KIND_UNSPECIFIED,
			otKind:     tracetranslator.OpenTracingSpanKindProducer,
		},
		{
			zipkinKind: zipkinmodel.Consumer,
			ocKind:     tracepb.Span_SPAN_KIND_UNSPECIFIED,
			otKind:     tracetranslator.OpenTracingSpanKindConsumer,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.zipkinKind), func(t *testing.T) {
			zs := &zipkinmodel.SpanModel{
				SpanContext: zipkinmodel.SpanContext{
					TraceID: zipkinmodel.TraceID{Low: 123},
					ID:      456,
				},
				Kind:      tt.zipkinKind,
				Timestamp: time.Now(),
			}
			ocSpan, _ := zipkinSpanToTraceSpan(zs)
			assert.EqualValues(t, tt.ocKind, ocSpan.Kind)
			if tt.otKind != "" {
				otSpanKind := ocSpan.Attributes.AttributeMap[tracetranslator.TagSpanKind]
				assert.EqualValues(t, tt.otKind, otSpanKind.GetStringValue().Value)
			} else {
				assert.True(t, ocSpan.Attributes == nil)
			}
		})
	}
}

func TestV2ParsesTags(t *testing.T) {
	jsonBlob, err := ioutil.ReadFile("./testdata/zipkin_v2_single.json")
	require.NoError(t, err, "Failed to read sample JSON file: %v", err)

	var zs []*zipkinmodel.SpanModel
	require.NoError(t, json.Unmarshal(jsonBlob, &zs), "Failed to unmarshal zipkin spans")

	reqs, err := V2BatchToOCProto(zs)
	require.NoError(t, err, "Failed to convert Zipkin spans to Trace spans: %v", err)
	require.Len(t, reqs, 1, "Expecting only one request", len(reqs))
	require.Len(t, reqs, 1, "Expecting only one span", len(reqs[0].Spans))

	var expected = &tracepb.Span_Attributes{
		AttributeMap: map[string]*tracepb.AttributeValue{
			"http.path": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "/api"},
			}},
			"http.status_code": {Value: &tracepb.AttributeValue_IntValue{IntValue: 500}},
			"cache_hit":        {Value: &tracepb.AttributeValue_BoolValue{BoolValue: true}},
			"ping_count":       {Value: &tracepb.AttributeValue_IntValue{IntValue: 25}},
			"timeout": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "12.3"},
			}},
			"ipv6": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "7::80:807f"}},
			},
			"clnt/finagle.version": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "6.45.0"}},
			},
			"zipkin.remoteEndpoint.ipv4": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "192.168.99.101"}},
			},
			"zipkin.remoteEndpoint.port": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "9000"}},
			},
			"zipkin.remoteEndpoint.serviceName": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "backend"}},
			},
		},
	}

	var span = reqs[0].Spans[0]
	assert.EqualValues(t, expected, span.Attributes)

	var expectedStatus = &tracepb.Status{
		Code: tracetranslator.OCInternal,
	}
	assert.EqualValues(t, expectedStatus, span.Status)
}

func TestZipkinTagsToTraceAttributesDropTag(t *testing.T) {
	tags := map[string]string{
		"status.code":    "13",
		"status.message": "a message",
		"http.path":      "/api",
	}

	attrs, status := zipkinTagsToTraceAttributes(tags, zipkinmodel.Client)

	var expected = &tracepb.Span_Attributes{
		AttributeMap: map[string]*tracepb.AttributeValue{
			"http.path": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "/api"},
			}},
		},
	}
	assert.EqualValues(t, expected, attrs)
	assert.EqualValues(t, status, &tracepb.Status{Code: 13, Message: "a message"})
}

func TestNodeFromZipkinEndpointsSetsAttributeOnNode(t *testing.T) {
	zc := zipkinmodel.SpanContext{
		TraceID: zipkinmodel.TraceID{
			High: 0x0001020304050607,
			Low:  0x08090A0B0C0D0E0F},
		ID: zipkinmodel.ID(uint64(0xF1F2F3F4F5F6F7F8)),
	}
	zs := &zipkinmodel.SpanModel{
		SpanContext: zc,
		LocalEndpoint: &zipkinmodel.Endpoint{
			ServiceName: "my-service",
			IPv4:        net.IPv4(1, 2, 3, 4),
		},
	}

	pbs := &tracepb.Span{}
	_ = nodeFromZipkinEndpoints(zs, pbs)

	var expected = &tracepb.Span_Attributes{
		AttributeMap: map[string]*tracepb.AttributeValue{
			"ipv4": {Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: "1.2.3.4"},
			}},
		},
	}

	assert.EqualValues(t, expected, pbs.Attributes)
}

func TestV2ParsesTagsHandleNil(t *testing.T) {
	zs := []*zipkinmodel.SpanModel{
		nil,
	}

	reqs, err := V2BatchToOCProto(zs)
	assert.Nil(t, err)
	assert.EqualValues(t, 0, len(reqs))
}
