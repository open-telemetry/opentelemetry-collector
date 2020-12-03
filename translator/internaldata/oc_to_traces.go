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

package internaldata

import (
	"strings"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

// OCToTraceData converts OC data format to Traces.
// Deprecated: use pdata.Traces instead. OCToTraceData may be used only by OpenCensus
// receiver and exporter implementations.
func OCToTraceData(td consumerdata.TraceData) pdata.Traces {
	traceData := pdata.NewTraces()
	if td.Node == nil && td.Resource == nil && len(td.Spans) == 0 {
		return traceData
	}

	if len(td.Spans) == 0 {
		// At least one of the td.Node or td.Resource is not nil. Set the resource and return.
		rss := traceData.ResourceSpans()
		rss.Resize(1)
		ocNodeResourceToInternal(td.Node, td.Resource, rss.At(0).Resource())
		return traceData
	}

	// We may need to split OC spans into several ResourceSpans. OC Spans can have a
	// Resource field inside them set to nil which indicates they use the Resource
	// specified in "td.Resource", or they can have the Resource field inside them set
	// to non-nil which indicates they have overridden Resource field and "td.Resource"
	// does not apply to those spans.
	//
	// Each OC Span that has its own Resource field set to non-nil must be placed in a
	// separate ResourceSpans instance, containing only that span. All other OC Spans
	// that have nil Resource field must be placed in one other ResourceSpans instance,
	// which will gets its Resource field from "td.Resource".
	//
	// We will end up with with one or more ResourceSpans like this:
	//
	// ResourceSpans           ResourceSpans  ResourceSpans
	// +-----+-----+---+-----+ +------------+ +------------+
	// |Span1|Span2|...|SpanM| |Span        | |Span        | ...
	// +-----+-----+---+-----+ +------------+ +------------+

	// Count the number of spans that have nil Resource and need to be combined
	// in one slice.
	combinedSpanCount := 0
	distinctResourceCount := 0
	for _, ocSpan := range td.Spans {
		if ocSpan == nil {
			// Skip nil spans.
			continue
		}
		if ocSpan.Resource == nil {
			combinedSpanCount++
		} else {
			distinctResourceCount++
		}
	}
	// Total number of resources is equal to:
	// 1 (for all spans with nil resource) + numSpansWithResource (distinctResourceCount).

	rss := traceData.ResourceSpans()
	rss.Resize(distinctResourceCount + 1)
	rs0 := rss.At(0)
	ocNodeResourceToInternal(td.Node, td.Resource, rs0.Resource())

	// Allocate a slice for spans that need to be combined into first ResourceSpans.
	ilss := rs0.InstrumentationLibrarySpans()
	ilss.Resize(1)
	ils0 := ilss.At(0)
	combinedSpans := ils0.Spans()
	combinedSpans.Resize(combinedSpanCount)

	// Now do the span translation and place them in appropriate ResourceSpans
	// instances.

	// Index to next available slot in "combinedSpans" slice.
	combinedSpanIdx := 0
	// First ResourceSpans is used for the default resource, so start with 1.
	resourceSpanIdx := 1
	for _, ocSpan := range td.Spans {
		if ocSpan == nil {
			// Skip nil spans.
			continue
		}

		if ocSpan.Resource == nil {
			// Add the span to the "combinedSpans". combinedSpans length is equal
			// to combinedSpanCount. The loop above that calculates combinedSpanCount
			// has exact same conditions as we have here in this loop.
			ocSpanToInternal(ocSpan, combinedSpans.At(combinedSpanIdx))
			combinedSpanIdx++
		} else {
			// This span has a different Resource and must be placed in a different
			// ResourceSpans instance. Create a separate ResourceSpans item just for this span.
			ocSpanToResourceSpans(ocSpan, td.Node, traceData.ResourceSpans().At(resourceSpanIdx))
			resourceSpanIdx++
		}
	}

	return traceData
}

func ocSpanToResourceSpans(ocSpan *octrace.Span, node *occommon.Node, dest pdata.ResourceSpans) {
	ocNodeResourceToInternal(node, ocSpan.Resource, dest.Resource())
	ilss := dest.InstrumentationLibrarySpans()
	ilss.Resize(1)
	ils0 := ilss.At(0)
	spans := ils0.Spans()
	spans.Resize(1)
	ocSpanToInternal(ocSpan, spans.At(0))
}

func ocSpanToInternal(src *octrace.Span, dest pdata.Span) {
	// Note that ocSpanKindToInternal must be called before initAttributeMapFromOC
	// since it may modify src.Attributes (remove the attribute which represents the
	// span kind).
	dest.SetKind(ocSpanKindToInternal(src.Kind, src.Attributes))

	dest.SetTraceID(traceIDToInternal(src.TraceId))
	dest.SetSpanID(spanIDToInternal(src.SpanId))
	dest.SetTraceState(ocTraceStateToInternal(src.Tracestate))
	dest.SetParentSpanID(spanIDToInternal(src.ParentSpanId))

	dest.SetName(src.Name.GetValue())
	dest.SetStartTime(pdata.TimestampToUnixNano(src.StartTime))
	dest.SetEndTime(pdata.TimestampToUnixNano(src.EndTime))

	ocStatusToInternal(src.Status, src.Attributes, dest.Status())

	initAttributeMapFromOC(src.Attributes, dest.Attributes())
	dest.SetDroppedAttributesCount(ocAttrsToDroppedAttributes(src.Attributes))
	ocEventsToInternal(src.TimeEvents, dest)
	ocLinksToInternal(src.Links, dest)
	ocSameProcessAsParentSpanToInternal(src.SameProcessAsParentSpan, dest)
}

// Transforms the byte slice trace ID into a [16]byte internal pdata.TraceID.
// If larger input then it is truncated to 16 bytes.
func traceIDToInternal(traceID []byte) pdata.TraceID {
	tid := [16]byte{}
	copy(tid[:], traceID)
	return pdata.NewTraceID(tid)
}

// Transforms the byte slice span ID into a [8]byte internal pdata.SpanID.
// If larger input then it is truncated to 8 bytes.
func spanIDToInternal(spanID []byte) pdata.SpanID {
	sid := [8]byte{}
	copy(sid[:], spanID)
	return pdata.NewSpanID(sid)
}

func ocStatusToInternal(ocStatus *octrace.Status, ocAttrs *octrace.Span_Attributes, dest pdata.SpanStatus) {
	if ocStatus == nil {
		return
	}

	var code pdata.StatusCode
	switch ocStatus.Code {
	case trace.StatusCodeOK:
		code = pdata.StatusCodeUnset
	default:
		// all other OC status codes are errors.
		code = pdata.StatusCodeError
	}

	if ocAttrs != nil {
		// tracetranslator.TagStatusCode is set it must override the status code value.
		// See the reverse translation in traces_to_oc.go:statusToOC().
		if attr, ok := ocAttrs.AttributeMap[tracetranslator.TagStatusCode]; ok {
			code = pdata.StatusCode(attr.GetIntValue())
			delete(ocAttrs.AttributeMap, tracetranslator.TagStatusCode)
		}
	}

	dest.SetCode(code)
	dest.SetMessage(ocStatus.Message)
}

// Convert tracestate to W3C format. See the https://w3c.github.io/trace-context/
func ocTraceStateToInternal(ocTracestate *octrace.Span_Tracestate) pdata.TraceState {
	if ocTracestate == nil {
		return ""
	}
	var sb strings.Builder
	for i, entry := range ocTracestate.Entries {
		sb.Grow(1 + len(entry.Key) + 1 + len(entry.Value))
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(entry.Key)
		sb.WriteString("=")
		sb.WriteString(entry.Value)
	}
	return pdata.TraceState(sb.String())
}

func ocAttrsToDroppedAttributes(ocAttrs *octrace.Span_Attributes) uint32 {
	if ocAttrs == nil {
		return 0
	}
	return uint32(ocAttrs.DroppedAttributesCount)
}

// initAttributeMapFromOC initialize AttributeMap from OC attributes
func initAttributeMapFromOC(ocAttrs *octrace.Span_Attributes, dest pdata.AttributeMap) {
	if ocAttrs == nil {
		return
	}

	if len(ocAttrs.AttributeMap) > 0 {
		dest.InitEmptyWithCapacity(len(ocAttrs.AttributeMap))
		for key, ocAttr := range ocAttrs.AttributeMap {
			switch attribValue := ocAttr.Value.(type) {
			case *octrace.AttributeValue_StringValue:
				dest.UpsertString(key, attribValue.StringValue.GetValue())

			case *octrace.AttributeValue_IntValue:
				dest.UpsertInt(key, attribValue.IntValue)

			case *octrace.AttributeValue_BoolValue:
				dest.UpsertBool(key, attribValue.BoolValue)

			case *octrace.AttributeValue_DoubleValue:
				dest.UpsertDouble(key, attribValue.DoubleValue)

			default:
				dest.UpsertString(key, "<Unknown OpenCensus attribute value type>")
			}
		}
	}
}

func ocSpanKindToInternal(ocKind octrace.Span_SpanKind, ocAttrs *octrace.Span_Attributes) pdata.SpanKind {
	switch ocKind {
	case octrace.Span_SERVER:
		return pdata.SpanKindSERVER

	case octrace.Span_CLIENT:
		return pdata.SpanKindCLIENT

	case octrace.Span_SPAN_KIND_UNSPECIFIED:
		// Span kind field is unspecified, check if TagSpanKind attribute is set.
		// This can happen if span kind had no equivalent in OC, so we could represent it in
		// the SpanKind. In that case the span kind may be a special attribute TagSpanKind.
		if ocAttrs != nil {
			kindAttr := ocAttrs.AttributeMap[tracetranslator.TagSpanKind]
			if kindAttr != nil {
				strVal, ok := kindAttr.Value.(*octrace.AttributeValue_StringValue)
				if ok && strVal != nil {
					var otlpKind pdata.SpanKind
					switch tracetranslator.OpenTracingSpanKind(strVal.StringValue.GetValue()) {
					case tracetranslator.OpenTracingSpanKindConsumer:
						otlpKind = pdata.SpanKindCONSUMER
					case tracetranslator.OpenTracingSpanKindProducer:
						otlpKind = pdata.SpanKindPRODUCER
					case tracetranslator.OpenTracingSpanKindInternal:
						otlpKind = pdata.SpanKindINTERNAL
					default:
						return pdata.SpanKindUNSPECIFIED
					}
					delete(ocAttrs.AttributeMap, tracetranslator.TagSpanKind)
					return otlpKind
				}
			}
		}
		return pdata.SpanKindUNSPECIFIED

	default:
		return pdata.SpanKindUNSPECIFIED
	}
}

func ocEventsToInternal(ocEvents *octrace.Span_TimeEvents, dest pdata.Span) {
	if ocEvents == nil {
		return
	}

	dest.SetDroppedEventsCount(uint32(ocEvents.DroppedMessageEventsCount + ocEvents.DroppedAnnotationsCount))

	if len(ocEvents.TimeEvent) == 0 {
		return
	}

	events := dest.Events()
	events.Resize(len(ocEvents.TimeEvent))

	i := 0
	for _, ocEvent := range ocEvents.TimeEvent {
		if ocEvent == nil {
			// Skip nil source events.
			continue
		}

		event := events.At(i)
		i++

		event.SetTimestamp(pdata.TimestampToUnixNano(ocEvent.Time))

		switch teValue := ocEvent.Value.(type) {
		case *octrace.Span_TimeEvent_Annotation_:
			if teValue.Annotation != nil {
				event.SetName(teValue.Annotation.Description.GetValue())
				initAttributeMapFromOC(teValue.Annotation.Attributes, event.Attributes())
				event.SetDroppedAttributesCount(ocAttrsToDroppedAttributes(teValue.Annotation.Attributes))
			}

		case *octrace.Span_TimeEvent_MessageEvent_:
			ocMessageEventToInternalAttrs(teValue.MessageEvent, event.Attributes())
			// No dropped attributes for this case.
			event.SetDroppedAttributesCount(0)

		default:
			event.SetName("An unknown OpenCensus TimeEvent type was detected when translating")
		}
	}

	// Truncate the slice to only include populated items.
	events.Resize(i)
}

func ocLinksToInternal(ocLinks *octrace.Span_Links, dest pdata.Span) {
	if ocLinks == nil {
		return
	}

	dest.SetDroppedLinksCount(uint32(ocLinks.DroppedLinksCount))

	if len(ocLinks.Link) == 0 {
		return
	}

	links := dest.Links()
	links.Resize(len(ocLinks.Link))

	i := 0
	for _, ocLink := range ocLinks.Link {
		if ocLink == nil {
			continue
		}

		link := links.At(i)
		i++

		link.SetTraceID(traceIDToInternal(ocLink.TraceId))
		link.SetSpanID(spanIDToInternal(ocLink.SpanId))
		link.SetTraceState(ocTraceStateToInternal(ocLink.Tracestate))
		initAttributeMapFromOC(ocLink.Attributes, link.Attributes())
		link.SetDroppedAttributesCount(ocAttrsToDroppedAttributes(ocLink.Attributes))
	}

	// Truncate the slice to only include populated items.
	links.Resize(i)
}

func ocMessageEventToInternalAttrs(msgEvent *octrace.Span_TimeEvent_MessageEvent, dest pdata.AttributeMap) {
	if msgEvent == nil {
		return
	}

	dest.UpsertString(conventions.OCTimeEventMessageEventType, msgEvent.Type.String())
	dest.UpsertInt(conventions.OCTimeEventMessageEventID, int64(msgEvent.Id))
	dest.UpsertInt(conventions.OCTimeEventMessageEventUSize, int64(msgEvent.UncompressedSize))
	dest.UpsertInt(conventions.OCTimeEventMessageEventCSize, int64(msgEvent.CompressedSize))
}

func ocSameProcessAsParentSpanToInternal(spaps *wrapperspb.BoolValue, dest pdata.Span) {
	if spaps == nil {
		return
	}
	dest.Attributes().UpsertBool(conventions.OCAttributeSameProcessAsParentSpan, spaps.Value)
}
