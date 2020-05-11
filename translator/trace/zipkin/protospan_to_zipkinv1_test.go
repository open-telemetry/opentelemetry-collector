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

package zipkin

import (
	"net"
	"testing"
	"time"

	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/trace/tracestate"

	"github.com/open-telemetry/opentelemetry-collector/internal"
)

func TestZipkinEndpointFromNode(t *testing.T) {
	type args struct {
		attributes   map[string]*tracepb.AttributeValue
		serviceName  string
		endpointType zipkinDirection
	}
	type want struct {
		endpoint      *zipkinmodel.Endpoint
		redundantKeys map[string]bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Nil attributes",
			args: args{attributes: nil, serviceName: "", endpointType: isLocalEndpoint},
			want: want{
				redundantKeys: make(map[string]bool),
			},
		},
		{
			name: "Only svc name",
			args: args{
				attributes:   make(map[string]*tracepb.AttributeValue),
				serviceName:  "test",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{ServiceName: "test"},
				redundantKeys: make(map[string]bool),
			},
		},
		{
			name: "Only ipv4",
			args: args{
				attributes: map[string]*tracepb.AttributeValue{
					"ipv4": {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "1.2.3.4"}}},
				},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{IPv4: net.ParseIP("1.2.3.4")},
				redundantKeys: map[string]bool{"ipv4": true},
			},
		},
		{
			name: "Only ipv6 remote",
			args: args{
				attributes: map[string]*tracepb.AttributeValue{
					RemoteEndpointIPv6: {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "2001:0db8:85a3:0000:0000:8a2e:0370:7334"}}},
				},
				serviceName:  "",
				endpointType: isRemoteEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{IPv6: net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")},
				redundantKeys: map[string]bool{RemoteEndpointIPv6: true},
			},
		},
		{
			name: "Only port",
			args: args{
				attributes: map[string]*tracepb.AttributeValue{
					"port": {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "42"}}},
				},
				serviceName:  "",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{Port: 42},
				redundantKeys: map[string]bool{"port": true},
			},
		},
		{
			name: "Service name, ipv4, and port",
			args: args{
				attributes: map[string]*tracepb.AttributeValue{
					"ipv4": {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "4.3.2.1"}}},
					"port": {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "2"}}},
				},
				serviceName:  "test-svc",
				endpointType: isLocalEndpoint,
			},
			want: want{
				endpoint:      &zipkinmodel.Endpoint{ServiceName: "test-svc", IPv4: net.ParseIP("4.3.2.1"), Port: 2},
				redundantKeys: map[string]bool{"ipv4": true, "port": true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redundantKeys := make(map[string]bool)
			endpoint := zipkinEndpointFromAttributes(
				tt.args.attributes,
				tt.args.serviceName,
				tt.args.endpointType,
				redundantKeys)
			assert.Equal(t, tt.want.endpoint, endpoint)
			assert.Equal(t, tt.want.redundantKeys, redundantKeys)
		})
	}
}

func TestOCSpanProtoToZipkin_endToEnd(t *testing.T) {
	// The goal of this test is to ensure that each
	// tracepb.Span is transformed to its *trace.SpanData correctly!

	endTime := time.Now().Round(time.Second)
	startTime := endTime.Add(-90 * time.Second)

	protoSpan := &tracepb.Span{
		TraceId:      []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
		SpanId:       []byte{0xF1, 0xF2, 0xF3, 0xF4, 0xF5, 0xF6, 0xF7, 0xF8},
		ParentSpanId: []byte{0xEF, 0xEE, 0xED, 0xEC, 0xEB, 0xEA, 0xE9, 0xE8},
		Name:         &tracepb.TruncatableString{Value: "End-To-End Here"},
		Kind:         tracepb.Span_SERVER,
		StartTime:    internal.TimeToTimestamp(startTime),
		EndTime:      internal.TimeToTimestamp(endTime),
		Status: &tracepb.Status{
			Code:    13,
			Message: "This is not a drill!",
		},
		SameProcessAsParentSpan: &wrappers.BoolValue{Value: false},
		TimeEvents: &tracepb.Span_TimeEvents{
			TimeEvent: []*tracepb.Span_TimeEvent{
				{
					Time: internal.TimeToTimestamp(startTime),
					Value: &tracepb.Span_TimeEvent_MessageEvent_{
						MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
							Type: tracepb.Span_TimeEvent_MessageEvent_RECEIVED, UncompressedSize: 1024, CompressedSize: 512,
						},
					},
				},
				{
					Time: internal.TimeToTimestamp(endTime),
					Value: &tracepb.Span_TimeEvent_MessageEvent_{
						MessageEvent: &tracepb.Span_TimeEvent_MessageEvent{
							Type: tracepb.Span_TimeEvent_MessageEvent_SENT, UncompressedSize: 1024, CompressedSize: 1000,
						},
					},
				},
				{
					Time: internal.TimeToTimestamp(endTime),
					Value: &tracepb.Span_TimeEvent_Annotation_{
						Annotation: &tracepb.Span_TimeEvent_Annotation{
							Description: &tracepb.TruncatableString{Value: "Description"},
						},
					},
				},
			},
		},
		Links: &tracepb.Span_Links{
			Link: []*tracepb.Span_Link{
				{
					TraceId: []byte{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7, 0xC8, 0xC9, 0xCA, 0xCB, 0xCC, 0xCD, 0xCE, 0xCF},
					SpanId:  []byte{0xB0, 0xB1, 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7},
					Type:    tracepb.Span_Link_PARENT_LINKED_SPAN,
				},
				{
					TraceId: []byte{0xE0, 0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8, 0xE9, 0xEA, 0xEB, 0xEC, 0xED, 0xEE, 0xEF},
					SpanId:  []byte{0xD0, 0xD1, 0xD2, 0xD3, 0xD4, 0xD5, 0xD6, 0xD7},
					Type:    tracepb.Span_Link_CHILD_LINKED_SPAN,
				},
			},
		},
		Tracestate: &tracepb.Span_Tracestate{
			Entries: []*tracepb.Span_Tracestate_Entry{
				{Key: "foo", Value: "bar"},
				{Key: "a", Value: "b"},
			},
		},
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: map[string]*tracepb.AttributeValue{
				"cache_hit":  {Value: &tracepb.AttributeValue_BoolValue{BoolValue: true}},
				"ping_count": {Value: &tracepb.AttributeValue_IntValue{IntValue: 25}},
				"timeout":    {Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 12.3}},
				"agent": {Value: &tracepb.AttributeValue_StringValue{
					StringValue: &tracepb.TruncatableString{Value: "ocagent"},
				}},
			},
		},
	}

	resource := &resourcepb.Resource{
		Type: "k8s",
		Labels: map[string]string{
			"namespace": "kube-system",
		},
	}

	gotOCSpanData, err := OCSpanProtoToZipkin(nil, resource, protoSpan, "default_service")
	if err != nil {
		t.Fatalf("Failed to convert from ProtoSpan to OCSpanData: %v", err)
	}

	ocTracestate, err := tracestate.New(new(tracestate.Tracestate), tracestate.Entry{Key: "foo", Value: "bar"},
		tracestate.Entry{Key: "a", Value: "b"})
	if err != nil || ocTracestate == nil {
		t.Fatalf("Failed to create ocTracestate: %v", err)
	}

	sampled := true
	parentID := zipkinmodel.ID(uint64(0xEFEEEDECEBEAE9E8))
	wantZipkinSpan := &zipkinmodel.SpanModel{
		SpanContext: zipkinmodel.SpanContext{
			TraceID: zipkinmodel.TraceID{
				High: 0x0001020304050607,
				Low:  0x08090A0B0C0D0E0F},
			ID:       zipkinmodel.ID(uint64(0xF1F2F3F4F5F6F7F8)),
			ParentID: &parentID,
			Sampled:  &sampled,
		},
		LocalEndpoint: &zipkinmodel.Endpoint{
			ServiceName: "default_service",
		},
		Kind:      zipkinmodel.Server,
		Name:      "End-To-End Here",
		Timestamp: startTime,
		Duration:  endTime.Sub(startTime),
		Annotations: []zipkinmodel.Annotation{
			{
				Timestamp: startTime,
				Value:     "RECEIVED",
			},
			{
				Timestamp: endTime,
				Value:     "SENT",
			},
			{
				Timestamp: endTime,
				Value:     "Description",
			},
		},
		Tags: map[string]string{
			"namespace":             "kube-system",
			"timeout":               "12.3",
			"agent":                 "ocagent",
			"cache_hit":             "true",
			"ping_count":            "25",
			statusCodeTagKey:        "INTERNAL",
			statusDescriptionTagKey: "This is not a drill!",
		},
	}

	assert.EqualValues(t, wantZipkinSpan, gotOCSpanData)
}

func TestOCSpanProtoToZipkin_NilSpan(t *testing.T) {
	span, err := OCSpanProtoToZipkin(nil, nil, nil, "")
	assert.Error(t, err)
	assert.Nil(t, span)
}
