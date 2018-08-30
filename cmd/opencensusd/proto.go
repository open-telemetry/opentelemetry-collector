// Copyright 2018, OpenCensus Authors
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

package main

import (
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"
	"go.opencensus.io/trace"
)

func protoToSpanData(s *tracepb.Span) *trace.SpanData {
	sd := &trace.SpanData{
		SpanContext: trace.SpanContext{
			TraceID: protoToTraceID(s.TraceId),
			SpanID:  protoToSpanID(s.SpanId),
		},
		ParentSpanID: protoToSpanID(s.ParentSpanId),
	}

	if s.Name != nil {
		sd.Name = s.Name.Value
	}

	switch s.Kind {
	case tracepb.Span_SERVER:
		sd.SpanKind = trace.SpanKindServer
	case tracepb.Span_CLIENT:
		sd.SpanKind = trace.SpanKindClient
	}

	sd.StartTime, _ = ptypes.Timestamp(s.StartTime)
	sd.EndTime, _ = ptypes.Timestamp(s.EndTime)
	sd.Attributes = protoToAttributes(s.Attributes)

	if s.Links != nil {
		for _, pbLink := range s.Links.Link {
			link := trace.Link{
				TraceID:    protoToTraceID(pbLink.TraceId),
				SpanID:     protoToSpanID(pbLink.SpanId),
				Attributes: protoToAttributes(pbLink.Attributes),
				Type:       trace.LinkType(pbLink.Type),
			}
			sd.Links = append(sd.Links, link)
		}
	}

	if s.Status != nil {
		sd.Status = trace.Status{
			Code:    s.Status.Code,
			Message: s.Status.Message,
		}
	}

	// TODO(jbd): Add annotations.
	// TODO(jbd): Add message events.
	if s.SameProcessAsParentSpan != nil {
		sd.HasRemoteParent = !s.SameProcessAsParentSpan.Value
	}
	return sd
}

func protoToAttributes(pbAttrs *tracepb.Span_Attributes) map[string]interface{} {
	if pbAttrs == nil {
		return nil
	}
	attrs := make(map[string]interface{}, len(pbAttrs.AttributeMap))
	for key, a := range pbAttrs.AttributeMap {
		switch v := a.Value.(type) {
		case *tracepb.AttributeValue_StringValue:
			attrs[key] = v.StringValue.Value
		case *tracepb.AttributeValue_IntValue:
			attrs[key] = v.IntValue
		case *tracepb.AttributeValue_BoolValue:
			attrs[key] = v.BoolValue
		}
	}
	return attrs
}

func protoToTraceID(tid []byte) trace.TraceID {
	var traceID trace.TraceID
	copy(traceID[:], tid)
	return traceID
}

func protoToSpanID(id []byte) trace.SpanID {
	var spanID trace.SpanID
	copy(spanID[:], id)
	return spanID
}
