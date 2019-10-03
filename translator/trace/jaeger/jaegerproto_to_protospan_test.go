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

package jaeger

import (
	"encoding/binary"
	"testing"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/google/go-cmp/cmp"
	model "github.com/jaegertracing/jaeger/model"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestOpenCensusToJaeger(t *testing.T) {
	now := time.Unix(1542158650, 536343000).UTC()
	d10min := 10 * time.Minute
	d2sec := 2 * time.Second
	nowPlus10min := now.Add(d10min)
	nowPlus10min2sec := now.Add(d10min).Add(d2sec)

	jaeger, err := ProtoBatchToOCProto(grpcFixture(now, d10min, d2sec))
	assert.NoError(t, err, "should not have failed to convert Jaeger Protobuf to OC Proto")

	oc := expectedTraceData(now, nowPlus10min, nowPlus10min2sec)

	if diff := cmp.Diff(jaeger, oc); diff != "" {
		t.Errorf("Mismatched responses\n-Got +Want:\n\t%s", diff)
	}
}

func expectedTraceData(t1, t2, t3 time.Time) consumerdata.TraceData {
	traceID := []byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80}
	parentSpanID := []byte{0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18}
	childSpanID := []byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8}

	return consumerdata.TraceData{
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{Name: "issaTest"},
			LibraryInfo: &commonpb.LibraryInfo{
				ExporterVersion: "Jaeger-Go-2.15.1dev",
			},
			Identifier: &commonpb.ProcessIdentifier{
				HostName: "acme.example.com",
			},
			Attributes: map[string]string{
				"bool":   "true",
				"string": "yes",
				"int64":  "10000000",
				"bin":    "3q3A3g==",
				"f":      "1",
			},
		},
		Spans: []*tracepb.Span{
			{
				TraceId:      traceID,
				SpanId:       childSpanID,
				ParentSpanId: parentSpanID,
				Name:         &tracepb.TruncatableString{Value: "DBSearch"},
				StartTime:    internal.TimeToTimestamp(t1),
				EndTime:      internal.TimeToTimestamp(t2),
				Kind:         tracepb.Span_SERVER,
				Status: &tracepb.Status{
					Code:    trace.StatusCodeNotFound,
					Message: "Stale indices",
				},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"error": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
						"bin": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "3q3A3g=="}},
						},
						"f": {
							Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 1},
						},
						"span.kind": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "server"},
							},
						},
					},
				},
				Links: &tracepb.Span_Links{
					Link: []*tracepb.Span_Link{
						{
							TraceId: traceID,
							SpanId:  parentSpanID,
							Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
						},
					},
				},
				TimeEvents: &tracepb.Span_TimeEvents{
					TimeEvent: []*tracepb.Span_TimeEvent{
						{
							Time: internal.TimeToTimestamp(t1),
							Value: &tracepb.Span_TimeEvent_Annotation_{
								Annotation: &tracepb.Span_TimeEvent_Annotation{
									Description: &tracepb.TruncatableString{Value: "executing DB search"},
									Attributes: &tracepb.Span_Attributes{
										AttributeMap: map[string]*tracepb.AttributeValue{
											"message": {
												Value: &tracepb.AttributeValue_StringValue{
													StringValue: &tracepb.TruncatableString{Value: "executing DB search"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			{
				TraceId:   traceID,
				SpanId:    parentSpanID,
				Name:      &tracepb.TruncatableString{Value: "ProxyFetch"},
				StartTime: internal.TimeToTimestamp(t2),
				EndTime:   internal.TimeToTimestamp(t3),
				Kind:      tracepb.Span_CLIENT,
				Status: &tracepb.Status{
					Code:    trace.StatusCodeInternal,
					Message: "Frontend crash",
				},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"error": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
						"http.status_code": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 500},
						},
						"http.status_message": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "Internal server error"},
							},
						},
						"span.kind": {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "client"},
							},
						},
					},
				},
			},
		},
	}

}

func grpcFixture(t1 time.Time, d1, d2 time.Duration) model.Batch {
	traceID := model.TraceID{}
	traceID.Unmarshal([]byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8, 0xF9, 0xFA, 0xFB, 0xFC, 0xFD, 0xFE, 0xFF, 0x80})
	parentSpanID := model.NewSpanID(binary.BigEndian.Uint64([]byte{0x1F, 0x1E, 0x1D, 0x1C, 0x1B, 0x1A, 0x19, 0x18}))
	childSpanID := model.NewSpanID(binary.BigEndian.Uint64([]byte{0xAF, 0xAE, 0xAD, 0xAC, 0xAB, 0xAA, 0xA9, 0xA8}))

	return model.Batch{
		Process: &model.Process{
			ServiceName: "issaTest",
			Tags: []model.KeyValue{
				model.String("hostname", "acme.example.com"),
				model.String("jaeger.version", "Go-2.15.1dev"),
				model.Bool("bool", true),
				model.String("string", "yes"),
				model.Int64("int64", 1e7),
				model.Float64("f", 1.),
				model.Binary("bin", []byte{0xde, 0xad, 0xc0, 0xde}),
			},
		},
		Spans: []*model.Span{
			{
				TraceID:       traceID,
				SpanID:        childSpanID,
				OperationName: "DBSearch",
				StartTime:     t1,
				Duration:      d1,
				Tags: []model.KeyValue{
					model.String(tracetranslator.TagStatusMsg, "Stale indices"),
					model.Int64(tracetranslator.TagStatusCode, trace.StatusCodeNotFound),
					model.Bool("error", true),
					model.String("span.kind", "server"),
					model.Float64("f", 1.),
					model.Binary("bin", []byte{0xde, 0xad, 0xc0, 0xde}),
				},
				References: []model.SpanRef{
					{
						TraceID: traceID,
						SpanID:  parentSpanID,
						RefType: model.SpanRefType_CHILD_OF,
					},
				},
				Logs: []model.Log{
					{
						Timestamp: t1,
						Fields: []model.KeyValue{
							model.String("message", "executing DB search"),
						},
					},
				},
			},
			{
				TraceID:       traceID,
				SpanID:        parentSpanID,
				OperationName: "ProxyFetch",
				StartTime:     t1.Add(d1),
				Duration:      d2,
				Tags: []model.KeyValue{
					model.String(tracetranslator.TagStatusMsg, "Frontend crash"),
					model.Int64(tracetranslator.TagStatusCode, trace.StatusCodeInternal),
					model.String(tracetranslator.TagHTTPStatusMsg, "Internal server error"),
					model.Int64("http.status_code", 500),
					model.Bool("error", true),
					model.String("span.kind", "client"),
				},
			},
		},
	}
}
