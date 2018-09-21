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

/*
Package trace defines translators from Trace proto spans to OpenCensus Go spanData
*/
package tracetranslator

import (
	"errors"
	"time"

	"go.opencensus.io/trace"
	"go.opencensus.io/trace/tracestate"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
)

var errNilSpan = errors.New("expected a non-nil span")

func ProtoSpanToOCSpanData(span *tracepb.Span) (*trace.SpanData, error) {
	if span == nil {
		return nil, errNilSpan
	}

	sTracestate, err := protoTracestateToOCTracestate(span.Tracestate)
	if err != nil {
		return nil, err
	}

	sc := trace.SpanContext{Tracestate: sTracestate}
	copy(sc.TraceID[:], span.TraceId)
	copy(sc.SpanID[:], span.SpanId)
	var parentSpanID trace.SpanID
	copy(parentSpanID[:], span.ParentSpanId)
	sd := &trace.SpanData{
		SpanContext:     sc,
		ParentSpanID:    parentSpanID,
		StartTime:       timestampToTime(span.StartTime),
		EndTime:         timestampToTime(span.EndTime),
		Name:            derefTruncatableString(span.Name),
		Attributes:      protoAttributesToOCAttributes(span.Attributes),
		Links:           protoLinksToOCLinks(span.Links),
		Status:          protoStatusToOCStatus(span.Status),
		SpanKind:        protoSpanKindToOCSpanKind(span.Kind),
		MessageEvents:   protoTimeEventsToOCMessageEvents(span.TimeEvents),
		HasRemoteParent: protoSameProcessAsParentToOCHasRemoteParent(span.SameProcessAsParentSpan),
	}

	return sd, nil
}

func timestampToTime(ts *timestamp.Timestamp) (t time.Time) {
	if ts == nil {
		return
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos))
}

func derefTruncatableString(ts *tracepb.TruncatableString) string {
	if ts == nil {
		return ""
	}
	return ts.Value
}

func protoStatusToOCStatus(s *tracepb.Status) (ts trace.Status) {
	if s == nil {
		return
	}
	return trace.Status{
		Code:    s.Code,
		Message: s.Message,
	}
}

func protoTracestateToOCTracestate(ts *tracepb.Span_Tracestate) (*tracestate.Tracestate, error) {
	if ts == nil {
		return nil, nil
	}
	return tracestate.New(nil, protoEntriesToOCEntries(ts.Entries)...)
}

func protoEntriesToOCEntries(protoEntries []*tracepb.Span_Tracestate_Entry) []tracestate.Entry {
	ocEntries := make([]tracestate.Entry, 0, len(protoEntries))
	for _, protoEntry := range protoEntries {
		var entry tracestate.Entry
		if protoEntry != nil {
			entry.Key = protoEntry.Key
			entry.Value = protoEntry.Value
		}
		ocEntries = append(ocEntries, entry)
	}
	return ocEntries
}

func protoLinksToOCLinks(sls *tracepb.Span_Links) []trace.Link {
	if sls == nil || len(sls.Link) == 0 {
		return nil
	}
	links := make([]trace.Link, 0, len(sls.Link))
	for _, sl := range sls.Link {
		var traceID trace.TraceID
		var spanID trace.SpanID
		copy(traceID[:], sl.TraceId)
		copy(spanID[:], sl.SpanId)
		links = append(links, trace.Link{
			TraceID: traceID,
			SpanID:  spanID,
			Type:    protoLinkTypeToOCLinkType(sl.Type),
		})
	}
	return links
}

func protoLinkTypeToOCLinkType(lt tracepb.Span_Link_Type) trace.LinkType {
	switch lt {
	case tracepb.Span_Link_CHILD_LINKED_SPAN:
		return trace.LinkTypeChild
	case tracepb.Span_Link_PARENT_LINKED_SPAN:
		return trace.LinkTypeParent
	default:
		return trace.LinkTypeUnspecified
	}
}

func protoAttributesToOCAttributes(attrs *tracepb.Span_Attributes) map[string]interface{} {
	if attrs == nil {
		return nil
	}

	ocAttrsMap := make(map[string]interface{})
	if len(attrs.AttributeMap) == 0 {
		return ocAttrsMap
	}
	for key, attr := range attrs.AttributeMap {
		if attr == nil || attr.Value == nil {
			continue
		}
		switch value := attr.Value.(type) {
		case *tracepb.AttributeValue_BoolValue:
			ocAttrsMap[key] = value.BoolValue

		case *tracepb.AttributeValue_IntValue:
			ocAttrsMap[key] = value.IntValue

		case *tracepb.AttributeValue_StringValue:
			ocAttrsMap[key] = derefTruncatableString(value.StringValue)
		}
	}
	return ocAttrsMap
}

func protoTimeEventsToOCMessageEvents(tes *tracepb.Span_TimeEvents) []trace.MessageEvent {
	// TODO: (@odeke-em) file a bug with OpenCensus-Go and ask them why
	// only MessageEvents are implemented.
	if tes == nil || len(tes.TimeEvent) == 0 {
		return nil
	}

	ocmes := make([]trace.MessageEvent, 0, len(tes.TimeEvent))
	for _, te := range tes.TimeEvent {
		if te == nil {
			continue
		}
		var ocme trace.MessageEvent
		switch typ := te.Value.(type) {
		case *tracepb.Span_TimeEvent_MessageEvent_:
			me := typ.MessageEvent
			// TODO: (@odeke-em) file an issue with OpenCensus-Go to ask why
			// they have these attributes as int64 yet the proto definitions
			// are uint64, this could be a potential loss of precision particularly
			// in very high traffic systems.
			ocme.MessageID = int64(me.Id)
			ocme.UncompressedByteSize = int64(me.UncompressedSize)
			ocme.CompressedByteSize = int64(me.CompressedSize)
			ocme.EventType = protoMessageEventTypeToOCEventType(me.Type)
			ocme.Time = timestampToTime(te.Time)
		}
		ocmes = append(ocmes, ocme)
	}
	return ocmes
}

func protoMessageEventTypeToOCEventType(st tracepb.Span_TimeEvent_MessageEvent_Type) trace.MessageEventType {
	switch st {
	case tracepb.Span_TimeEvent_MessageEvent_SENT:
		return trace.MessageEventTypeSent
	case tracepb.Span_TimeEvent_MessageEvent_RECEIVED:
		return trace.MessageEventTypeRecv
	default:
		return trace.MessageEventTypeUnspecified
	}
}

func protoSpanKindToOCSpanKind(kind tracepb.Span_SpanKind) int {
	switch kind {
	case tracepb.Span_CLIENT:
		return trace.SpanKindClient
	case tracepb.Span_SERVER:
		return trace.SpanKindServer
	default:
		return trace.SpanKindUnspecified
	}
}

// Translating a variable that states if the parent and the current span are in the same process
// to a variable that indicates whether the current span and parent are in different process.
func protoSameProcessAsParentToOCHasRemoteParent(sameProcessAsParent *wrappers.BoolValue) bool {
	if sameProcessAsParent == nil {
		return false
	}
	return !sameProcessAsParent.Value
}
