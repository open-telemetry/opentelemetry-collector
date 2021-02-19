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
	"encoding/json"
	"io/ioutil"
	"sort"
	"strconv"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/google/go-cmp/cmp"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func Test_hexIDToOCID(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		want    []byte
		wantErr error
	}{
		{
			name:    "empty hex string",
			hexStr:  "",
			want:    nil,
			wantErr: errHexIDWrongLen,
		},
		{
			name:    "wrong length",
			hexStr:  "0000",
			want:    nil,
			wantErr: errHexIDWrongLen,
		},
		{
			name:    "parse error",
			hexStr:  "000000000000000-",
			want:    nil,
			wantErr: errHexIDParsing,
		},
		{
			name:    "all zero",
			hexStr:  "0000000000000000",
			want:    nil,
			wantErr: errHexIDZero,
		},
		{
			name:    "happy path",
			hexStr:  "0706050400010203",
			want:    []byte{7, 6, 5, 4, 0, 1, 2, 3},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexIDToOCID(tt.hexStr)
			require.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_hexTraceIDToOCTraceID(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		want    []byte
		wantErr error
	}{
		{
			name:    "empty hex string",
			hexStr:  "",
			want:    nil,
			wantErr: errHexTraceIDWrongLen,
		},
		{
			name:    "wrong length",
			hexStr:  "000000000000000010",
			want:    nil,
			wantErr: errHexTraceIDWrongLen,
		},
		{
			name:    "parse error",
			hexStr:  "000000000000000X0000000000000000",
			want:    nil,
			wantErr: errHexTraceIDParsing,
		},
		{
			name:    "all zero",
			hexStr:  "00000000000000000000000000000000",
			want:    nil,
			wantErr: errHexTraceIDZero,
		},
		{
			name:    "happy path",
			hexStr:  "00000000000000010000000000000002",
			want:    []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexTraceIDToOCTraceID(tt.hexStr)
			require.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestZipkinJSONFallbackToLocalComponent(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_local_component.json")
	require.NoError(t, err, "Failed to load test data")

	reqs, err := v1JSONBatchToOCProto(blob, false)
	require.NoError(t, err, "Failed to translate zipkinv1 to OC proto")
	require.Equal(t, 2, len(reqs), "Invalid trace service requests count")

	// Ensure the order of nodes
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].Node.ServiceInfo.Name < reqs[j].Node.ServiceInfo.Name
	})

	// First span didn't have a host/endpoint to give service name, use the local component.
	got := reqs[0].Node.ServiceInfo.Name
	require.Equal(t, "myLocalComponent", got)

	// Second span have a host/endpoint to give service name, do not use local component.
	got = reqs[1].Node.ServiceInfo.Name
	require.Equal(t, "myServiceName", got)
}

func TestSingleJSONV1BatchToOCProto(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_single_batch.json")
	require.NoError(t, err, "Failed to load test data")

	parseStringTags := true // This test relies on parsing int/bool to the typed span attributes
	got, err := v1JSONBatchToOCProto(blob, parseStringTags)
	require.NoError(t, err, "Failed to translate zipkinv1 to OC proto")

	want := ocBatchesFromZipkinV1
	sortTraceByNodeName(want)
	sortTraceByNodeName(got)

	assert.EqualValues(t, got, want)
}

func TestMultipleJSONV1BatchesToOCProto(t *testing.T) {
	blob, err := ioutil.ReadFile("./testdata/zipkin_v1_multiple_batches.json")
	require.NoError(t, err, "Failed to load test data")

	var batches []interface{}
	err = json.Unmarshal(blob, &batches)
	require.NoError(t, err, "Failed to load the batches")

	nodeToTraceReqs := make(map[string]*traceData)
	var got []traceData
	for _, batch := range batches {
		jsonBatch, err := json.Marshal(batch)
		require.NoError(t, err, "Failed to marshal interface back to blob")

		parseStringTags := true // This test relies on parsing int/bool to the typed span attributes
		g, err := v1JSONBatchToOCProto(jsonBatch, parseStringTags)
		require.NoError(t, err, "Failed to translate zipkinv1 to OC proto")

		// Coalesce the nodes otherwise they will differ due to multiple
		// nodes representing same logical service
		for i := range g {
			key := g[i].Node.String()
			if pTsr, ok := nodeToTraceReqs[key]; ok {
				pTsr.Spans = append(pTsr.Spans, g[i].Spans...)
			} else {
				nodeToTraceReqs[key] = &g[i]
			}
		}
	}

	for _, tsr := range nodeToTraceReqs {
		got = append(got, *tsr)
	}

	want := ocBatchesFromZipkinV1
	sortTraceByNodeName(want)
	sortTraceByNodeName(got)

	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected difference:\n%v", diff)
	}
}

func sortTraceByNodeName(trace []traceData) {
	sort.Slice(trace, func(i, j int) bool {
		return trace[i].Node.ServiceInfo.Name < trace[j].Node.ServiceInfo.Name
	})
}

func TestZipkinAnnotationsToOCStatus(t *testing.T) {
	type test struct {
		name           string
		haveTags       []*binaryAnnotation
		wantAttributes *tracepb.Span_Attributes
		wantStatus     *tracepb.Status
	}

	cases := []test{
		{
			name: "only status.code tag",
			haveTags: []*binaryAnnotation{{
				Key:   "status.code",
				Value: "13",
			}},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code: 13,
			},
		},

		{
			name: "only status.message tag",
			haveTags: []*binaryAnnotation{{
				Key:   "status.message",
				Value: "Forbidden",
			}},
			wantAttributes: nil,
			wantStatus:     nil,
		},

		{
			name: "both status.code and status.message",
			haveTags: []*binaryAnnotation{
				{
					Key:   "status.code",
					Value: "13",
				},
				{
					Key:   "status.message",
					Value: "Forbidden",
				},
			},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code:    13,
				Message: "Forbidden",
			},
		},

		{
			name: "http status.code",
			haveTags: []*binaryAnnotation{
				{
					Key:   "http.status_code",
					Value: "404",
				},
				{
					Key:   "http.status_message",
					Value: "NotFound",
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    5,
				Message: "NotFound",
			},
		},

		{
			name: "http and oc",
			haveTags: []*binaryAnnotation{
				{
					Key:   "http.status_code",
					Value: "404",
				},
				{
					Key:   "http.status_message",
					Value: "NotFound",
				},
				{
					Key:   "status.code",
					Value: "13",
				},
				{
					Key:   "status.message",
					Value: "Forbidden",
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    13,
				Message: "Forbidden",
			},
		},

		{
			name: "http and only oc code",
			haveTags: []*binaryAnnotation{
				{
					Key:   "http.status_code",
					Value: "404",
				},
				{
					Key:   "http.status_message",
					Value: "NotFound",
				},
				{
					Key:   "status.code",
					Value: "14",
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code: 14,
			},
		},

		{
			name: "http and only oc message",
			haveTags: []*binaryAnnotation{
				{
					Key:   "http.status_code",
					Value: "404",
				},
				{
					Key:   "http.status_message",
					Value: "NotFound",
				},
				{
					Key:   "status.message",
					Value: "Forbidden",
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    5,
				Message: "NotFound",
			},
		},

		{
			name: "census tags",
			haveTags: []*binaryAnnotation{
				{
					Key:   "census.status_code",
					Value: "10",
				},
				{
					Key:   "census.status_description",
					Value: "RPCError",
				},
			},
			wantAttributes: nil,
			wantStatus: &tracepb.Status{
				Code:    10,
				Message: "RPCError",
			},
		},

		{
			name: "census tags priority over others",
			haveTags: []*binaryAnnotation{
				{
					Key:   "census.status_code",
					Value: "10",
				},
				{
					Key:   "census.status_description",
					Value: "RPCError",
				},
				{
					Key:   "http.status_code",
					Value: "404",
				},
				{
					Key:   "http.status_message",
					Value: "NotFound",
				},
				{
					Key:   "status.message",
					Value: "Forbidden",
				},
				{
					Key:   "status.code",
					Value: "7",
				},
			},
			wantAttributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					tracetranslator.TagHTTPStatusCode: {
						Value: &tracepb.AttributeValue_IntValue{
							IntValue: 404,
						},
					},
					tracetranslator.TagHTTPStatusMsg: {
						Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NotFound"},
						},
					},
				},
			},
			wantStatus: &tracepb.Status{
				Code:    10,
				Message: "RPCError",
			},
		},
	}

	fakeTraceID := "00000000000000010000000000000002"
	fakeSpanID := "0000000000000001"

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			zSpans := []*zipkinV1Span{{
				ID:                fakeSpanID,
				TraceID:           fakeTraceID,
				BinaryAnnotations: c.haveTags,
				Timestamp:         1,
			}}
			zBytes, err := json.Marshal(zSpans)
			if err != nil {
				t.Errorf("#%d: Unexpected error: %v", i, err)
				return
			}

			parseStringTags := true // This test relies on parsing int/bool to the typed span attributes
			gb, err := v1JSONBatchToOCProto(zBytes, parseStringTags)
			if err != nil {
				t.Errorf("#%d: Unexpected error: %v", i, err)
				return
			}
			gs := gb[0].Spans[0]
			require.Equal(t, c.wantAttributes, gs.Attributes, "Unsuccessful conversion %d", i)
			require.Equal(t, c.wantStatus, gs.Status, "Unsuccessful conversion %d", i)
		})
	}
}

func TestSpanWithoutTimestampGetsTag(t *testing.T) {
	fakeTraceID := "00000000000000010000000000000002"
	fakeSpanID := "0000000000000001"
	zSpans := []*zipkinV1Span{
		{
			ID:        fakeSpanID,
			TraceID:   fakeTraceID,
			Timestamp: 0, // no timestamp field
		},
	}
	zBytes, err := json.Marshal(zSpans)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	testStart := time.Now()

	gb, err := v1JSONBatchToOCProto(zBytes, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	gs := gb[0].Spans[0]
	assert.NotNil(t, gs.StartTime)
	assert.NotNil(t, gs.EndTime)

	assert.True(t, gs.StartTime.AsTime().Sub(testStart) >= 0)

	wantAttributes := &tracepb.Span_Attributes{
		AttributeMap: map[string]*tracepb.AttributeValue{
			StartTimeAbsent: {
				Value: &tracepb.AttributeValue_BoolValue{
					BoolValue: true,
				},
			},
		},
	}

	assert.EqualValues(t, gs.Attributes, wantAttributes)
}

func TestJSONHTTPToGRPCStatusCode(t *testing.T) {
	fakeTraceID := "00000000000000010000000000000002"
	fakeSpanID := "0000000000000001"
	for i := int32(100); i <= 600; i++ {
		wantStatus := tracetranslator.OCStatusCodeFromHTTP(i)
		zBytes, err := json.Marshal([]*zipkinV1Span{{
			ID:      fakeSpanID,
			TraceID: fakeTraceID,
			BinaryAnnotations: []*binaryAnnotation{
				{
					Key:   "http.status_code",
					Value: strconv.Itoa(int(i)),
				},
			},
		}})
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}
		gb, err := v1JSONBatchToOCProto(zBytes, false)
		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}

		gs := gb[0].Spans[0]
		require.Equal(t, wantStatus, gs.Status.Code, "Unsuccessful conversion %d", i)
	}
}

// ocBatches has the OpenCensus proto batches used in the test. They are hard coded because
// structs like tracepb.AttributeMap cannot be ready from JSON.
var ocBatchesFromZipkinV1 = []traceData{
	{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "front-proxy"},
			Attributes:  map[string]string{"ipv4": "172.31.0.2"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				SpanId:       []byte{0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				ParentSpanId: nil,
				Name:         &tracepb.TruncatableString{Value: "checkAvailability"},
				Kind:         tracepb.Span_CLIENT,
				StartTime:    &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 446743000},
				EndTime:      &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 459699000},
				TimeEvents:   nil,
			},
		},
	},
	{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "service1"},
			Attributes:  map[string]string{"ipv4": "172.31.0.4"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				SpanId:       []byte{0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				ParentSpanId: nil,
				Name:         &tracepb.TruncatableString{Value: "checkAvailability"},
				Kind:         tracepb.Span_SERVER,
				StartTime:    &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 448081000},
				EndTime:      &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 460102000},
				TimeEvents: &tracepb.Span_TimeEvents{
					TimeEvent: []*tracepb.Span_TimeEvent{
						{
							Time: &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 450000000},
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Description: &tracepb.TruncatableString{Value: "custom time event"},
								},
							},
						},
					},
				},
			},
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				SpanId:       []byte{0xf9, 0xeb, 0xb6, 0xe6, 0x48, 0x80, 0x61, 0x2a},
				ParentSpanId: []byte{0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				Name:         &tracepb.TruncatableString{Value: "checkStock"},
				Kind:         tracepb.Span_CLIENT,
				StartTime:    &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 453923000},
				EndTime:      &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 457663000},
				TimeEvents:   nil,
			},
		},
	},
	{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "service2"},
			Attributes:  map[string]string{"ipv4": "172.31.0.7"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				SpanId:       []byte{0xf9, 0xeb, 0xb6, 0xe6, 0x48, 0x80, 0x61, 0x2a},
				ParentSpanId: []byte{0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				Name:         &tracepb.TruncatableString{Value: "checkStock"},
				Kind:         tracepb.Span_SERVER,
				StartTime:    &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 454487000},
				EndTime:      &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 457320000},
				Status: &tracepb.Status{
					Code: 0,
				},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"http.status_code": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 200},
						},
						"http.url": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "http://localhost:9000/trace/2"}},
						},
						"success": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
						"processed": {
							Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 1.5},
						},
					},
				},
				TimeEvents: nil,
			},
		},
	},
	{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "unknown-service"},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				SpanId:       []byte{0xfe, 0x35, 0x1a, 0x05, 0x3f, 0xbc, 0xac, 0x1f},
				ParentSpanId: []byte{0x0e, 0xd2, 0xe6, 0x3c, 0xbe, 0x71, 0xf5, 0xa8},
				Name:         &tracepb.TruncatableString{Value: "checkStock"},
				Kind:         tracepb.Span_SPAN_KIND_UNSPECIFIED,
				StartTime:    &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 453923000},
				EndTime:      &timestamppb.Timestamp{Seconds: 1544805927, Nanos: 457663000},
				Attributes:   nil,
			},
		},
	},
}

func TestSpanKindTranslation(t *testing.T) {
	tests := []struct {
		zipkinV1Kind   string
		zipkinV2Kind   zipkinmodel.Kind
		ocKind         tracepb.Span_SpanKind
		ocAttrSpanKind tracetranslator.OpenTracingSpanKind
		jaegerSpanKind string
	}{
		{
			zipkinV1Kind:   "cr",
			zipkinV2Kind:   zipkinmodel.Client,
			ocKind:         tracepb.Span_CLIENT,
			jaegerSpanKind: "client",
		},

		{
			zipkinV1Kind:   "sr",
			zipkinV2Kind:   zipkinmodel.Server,
			ocKind:         tracepb.Span_SERVER,
			jaegerSpanKind: "server",
		},

		{
			zipkinV1Kind:   "ms",
			zipkinV2Kind:   zipkinmodel.Producer,
			ocKind:         tracepb.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrSpanKind: tracetranslator.OpenTracingSpanKindProducer,
			jaegerSpanKind: "producer",
		},

		{
			zipkinV1Kind:   "mr",
			zipkinV2Kind:   zipkinmodel.Consumer,
			ocKind:         tracepb.Span_SPAN_KIND_UNSPECIFIED,
			ocAttrSpanKind: tracetranslator.OpenTracingSpanKindConsumer,
			jaegerSpanKind: "consumer",
		},
	}

	for _, test := range tests {
		t.Run(test.zipkinV1Kind, func(t *testing.T) {
			// Create Zipkin V1 span.
			zSpan := &zipkinV1Span{
				TraceID: "1234567890123456",
				ID:      "0123456789123456",
				Annotations: []*annotation{
					{Value: test.zipkinV1Kind}, // note that only first annotation matters.
					{Value: "cr"},              // this will have no effect.
				},
			}

			// Translate to OC and verify that span kind is correctly translated.
			ocSpan, parsedAnnotations, err := zipkinV1ToOCSpan(zSpan, false)
			assert.NoError(t, err)
			assert.EqualValues(t, test.ocKind, ocSpan.Kind)
			assert.NotNil(t, parsedAnnotations)
			if test.ocAttrSpanKind != "" {
				require.NotNil(t, ocSpan.Attributes)
				// This is a special case, verify that TagSpanKind attribute is set.
				expected := &tracepb.AttributeValue{
					Value: &tracepb.AttributeValue_StringValue{
						StringValue: &tracepb.TruncatableString{Value: string(test.ocAttrSpanKind)},
					},
				}
				assert.EqualValues(t, expected, ocSpan.Attributes.AttributeMap[tracetranslator.TagSpanKind])
			}
		})
	}
}

func TestZipkinV1ToOCSpanInvalidTraceId(t *testing.T) {
	zSpan := &zipkinV1Span{
		TraceID: "abc",
		ID:      "0123456789123456",
		Annotations: []*annotation{
			{Value: "cr"},
		},
	}
	_, _, err := zipkinV1ToOCSpan(zSpan, false)
	assert.EqualError(t, err, "zipkinV1 span traceId: hex traceId span has wrong length (expected 16 or 32)")
}

func TestZipkinV1ToOCSpanInvalidSpanId(t *testing.T) {
	zSpan := &zipkinV1Span{
		TraceID: "1234567890123456",
		ID:      "abc",
		Annotations: []*annotation{
			{Value: "cr"},
		},
	}
	_, _, err := zipkinV1ToOCSpan(zSpan, false)
	assert.EqualError(t, err, "zipkinV1 span id: hex Id has wrong length (expected 16)")
}
