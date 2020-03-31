// Copyright 2020 OpenTelemetry Authors
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

package internaldata

import (
	"strings"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	octrace "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

// OCToTraceData converts OC data format to TraceData.
func OCToTraceData(td consumerdata.TraceData) data.TraceData {
	traceData := data.NewTraceData()
	if td.Node == nil && td.Resource == nil && len(td.Spans) == 0 {
		return traceData
	}

	if len(td.Spans) == 0 {
		// At least one of the td.Node or td.Resource is not nil. Set the resource and return.
		traceData.SetResourceSpans(data.NewResourceSpansSlice(1))
		ocNodeResourceToInternal(td.Node, td.Resource, traceData.ResourceSpans().Get(0))
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
	traceData.SetResourceSpans(data.NewResourceSpansSlice(distinctResourceCount + 1))
	rs0 := traceData.ResourceSpans().Get(0)
	ocNodeResourceToInternal(td.Node, td.Resource, rs0)

	// Allocate a slice for spans that need to be combined into first ResourceSpans.
	rs0.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	ils0 := rs0.InstrumentationLibrarySpans().Get(0)
	ils0.SetSpans(data.NewSpanSlice(combinedSpanCount))
	combinedSpans := ils0.Spans()

	// Now do the span translation and place them in appropriate ResourceSpans
	// instances.

	// Index to next available slot in "combinedSpans" slice.
	combinedSpanIdx := 0
	// First resourcespan is used for the default resource, so start with 1.
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
			ocSpanToInternal(combinedSpans.Get(combinedSpanIdx), ocSpan)
			combinedSpanIdx++
		} else {
			// This span has a different Resource and must be placed in a different
			// ResourceSpans instance. Create a separate ResourceSpans item just for this span.
			ocSpanToResourceSpans(ocSpan, td.Node, traceData.ResourceSpans().Get(resourceSpanIdx))
			resourceSpanIdx++
		}
	}

	return traceData
}

func ocSpanToResourceSpans(ocSpan *octrace.Span, node *occommon.Node, out data.ResourceSpans) {
	ocNodeResourceToInternal(node, ocSpan.Resource, out)
	out.SetInstrumentationLibrarySpans(data.NewInstrumentationLibrarySpansSlice(1))
	ils0 := out.InstrumentationLibrarySpans().Get(0)
	ils0.SetSpans(data.NewSpanSlice(1))
	ocSpanToInternal(ils0.Spans().Get(0), ocSpan)
}

func ocSpanToInternal(dest data.Span, src *octrace.Span) {
	events, droppedEventCount := ocEventsToInternal(src.TimeEvents)
	links, droppedLinkCount := ocLinksToInternal(src.Links)

	// Note that ocSpanKindToInternal must be called before ocAttrsToInternal
	// since it may modify src.Attributes (remove the attribute which represents the
	// span kind).
	kind := ocSpanKindToInternal(src.Kind, src.Attributes)
	attrs, droppedAttrs := ocAttrsToInternal(src.Attributes)

	dest.SetTraceID(data.NewTraceID(src.TraceId))
	dest.SetSpanID(data.NewSpanID(src.SpanId))
	dest.SetTraceState(ocTraceStateToInternal(src.Tracestate))
	dest.SetParentSpanID(data.NewSpanID(src.ParentSpanId))
	dest.SetName(src.Name.GetValue())
	dest.SetKind(kind)
	dest.SetStartTime(internal.TimestampToUnixNano(src.StartTime))
	dest.SetEndTime(internal.TimestampToUnixNano(src.EndTime))
	dest.SetAttributes(attrs)
	dest.SetDroppedAttributesCount(droppedAttrs)
	dest.SetEvents(events)
	dest.SetDroppedEventsCount(droppedEventCount)
	dest.SetLinks(links)
	dest.SetDroppedLinksCount(droppedLinkCount)
	ocStatusToInternal(src.Status, dest)
}

func ocStatusToInternal(ocStatus *octrace.Status, out data.Span) {
	if ocStatus == nil {
		return
	}
	out.InitStatusIfNil()
	out.Status().SetCode(data.StatusCode(ocStatus.Code))
	out.Status().SetMessage(ocStatus.Message)
}

// Convert tracestate to W3C format. See the https://w3c.github.io/trace-context/
func ocTraceStateToInternal(ocTracestate *octrace.Span_Tracestate) data.TraceState {
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
	return data.TraceState(sb.String())
}

func ocAttrsToInternal(ocAttrs *octrace.Span_Attributes) (data.AttributeMap, uint32) {
	if ocAttrs == nil {
		return data.NewAttributeMap(nil), 0
	}

	attrCount := len(ocAttrs.AttributeMap)
	attrMap := make(data.AttributesMap, attrCount)
	if attrCount > 0 {
		values := data.NewAttributeValueSlice(attrCount)
		i := 0
		for key, ocAttr := range ocAttrs.AttributeMap {
			switch attribValue := ocAttr.Value.(type) {
			case *octrace.AttributeValue_StringValue:
				values[i].SetString(attribValue.StringValue.GetValue())

			case *octrace.AttributeValue_IntValue:
				values[i].SetInt(attribValue.IntValue)

			case *octrace.AttributeValue_BoolValue:
				values[i].SetBool(attribValue.BoolValue)

			case *octrace.AttributeValue_DoubleValue:
				values[i].SetDouble(attribValue.DoubleValue)

			default:
				str := "<Unknown OpenCensus attribute value type>"
				values[i].SetString(str)
			}
			attrMap[key] = values[i]
			i++
		}
	}
	return data.NewAttributeMap(attrMap), uint32(ocAttrs.DroppedAttributesCount)
}

func ocSpanKindToInternal(ocKind octrace.Span_SpanKind, ocAttrs *octrace.Span_Attributes) data.SpanKind {
	switch ocKind {
	case octrace.Span_SERVER:
		return data.SpanKindSERVER

	case octrace.Span_CLIENT:
		return data.SpanKindCLIENT

	case octrace.Span_SPAN_KIND_UNSPECIFIED:
		// Span kind field is unspecified, check if TagSpanKind attribute is set.
		// This can happen if span kind had no equivalent in OC, so we could represent it in
		// the SpanKind. In that case the span kind may be a special attribute TagSpanKind.
		if ocAttrs != nil {
			kindAttr := ocAttrs.AttributeMap[tracetranslator.TagSpanKind]
			if kindAttr != nil {
				strVal, ok := kindAttr.Value.(*octrace.AttributeValue_StringValue)
				if ok && strVal != nil {
					var otlpKind data.SpanKind
					switch tracetranslator.OpenTracingSpanKind(strVal.StringValue.GetValue()) {
					case tracetranslator.OpenTracingSpanKindConsumer:
						otlpKind = data.SpanKindCONSUMER
					case tracetranslator.OpenTracingSpanKindProducer:
						otlpKind = data.SpanKindPRODUCER
					default:
						return data.SpanKindUNSPECIFIED
					}
					delete(ocAttrs.AttributeMap, tracetranslator.TagSpanKind)
					return otlpKind
				}
			}
		}
		return data.SpanKindUNSPECIFIED

	default:
		return data.SpanKindUNSPECIFIED
	}
}

func ocEventsToInternal(ocEvents *octrace.Span_TimeEvents) (data.SpanEventSlice, uint32) {
	if ocEvents == nil {
		return data.NewSpanEventSlice(0), 0
	}

	droppedCount := uint32(ocEvents.DroppedMessageEventsCount + ocEvents.DroppedAnnotationsCount)

	if len(ocEvents.TimeEvent) == 0 {
		return data.NewSpanEventSlice(0), droppedCount
	}

	events := data.NewSpanEventSlice(len(ocEvents.TimeEvent))

	i := 0
	for _, ocEvent := range ocEvents.TimeEvent {
		if ocEvent == nil {
			// Skip nil source events.
			continue
		}

		event := events.Get(i)
		i++

		event.SetTimestamp(internal.TimestampToUnixNano(ocEvent.Time))

		switch teValue := ocEvent.Value.(type) {
		case *octrace.Span_TimeEvent_Annotation_:
			if teValue.Annotation != nil {
				event.SetName(teValue.Annotation.Description.GetValue())
				attrs, droppedAttrs := ocAttrsToInternal(teValue.Annotation.Attributes)
				event.SetAttributes(attrs)
				event.SetDroppedAttributesCount(droppedAttrs)
			}

		case *octrace.Span_TimeEvent_MessageEvent_:
			attrs, droppedAttrs := ocMessageEventToInternalAttrs(teValue.MessageEvent)
			event.SetAttributes(attrs)
			event.SetDroppedAttributesCount(droppedAttrs)

		default:
			event.SetName("An unknown OpenCensus TimeEvent type was detected when translating")
		}
	}

	// Truncate the slice to only include populated items.
	events.Resize(0, i)

	return events, droppedCount
}

func ocLinksToInternal(ocLinks *octrace.Span_Links) (data.SpanLinkSlice, uint32) {
	if ocLinks == nil {
		return data.NewSpanLinkSlice(0), 0
	}

	if len(ocLinks.Link) == 0 {
		return data.NewSpanLinkSlice(0), uint32(ocLinks.DroppedLinksCount)
	}

	links := data.NewSpanLinkSlice(len(ocLinks.Link))

	i := 0
	for _, ocLink := range ocLinks.Link {
		if ocLink == nil {
			continue
		}

		link := links.Get(i)
		i++

		link.SetTraceID(data.NewTraceID(ocLink.TraceId))
		link.SetSpanID(data.NewSpanID(ocLink.SpanId))
		link.SetTraceState(ocTraceStateToInternal(ocLink.Tracestate))
		attr, droppedAttr := ocAttrsToInternal(ocLink.Attributes)
		link.SetAttributes(attr)
		link.SetDroppedAttributesCount(droppedAttr)
	}

	// Truncate the slice to only include populated items.
	links.Resize(0, i)

	return links, uint32(ocLinks.DroppedLinksCount)
}

func ocMessageEventToInternalAttrs(msgEvent *octrace.Span_TimeEvent_MessageEvent) (data.AttributeMap, uint32) {
	if msgEvent == nil {
		return data.NewAttributeMap(nil), 0
	}

	attrs := data.NewAttributeValueSlice(4)
	attrs[0].SetString(msgEvent.Type.String())
	attrs[1].SetInt(int64(msgEvent.Id))
	attrs[2].SetInt(int64(msgEvent.UncompressedSize))
	attrs[3].SetInt(int64(msgEvent.CompressedSize))

	return data.NewAttributeMap(data.AttributesMap{
		conventions.OCTimeEventMessageEventType:  attrs[0],
		conventions.OCTimeEventMessageEventID:    attrs[1],
		conventions.OCTimeEventMessageEventUSize: attrs[2],
		conventions.OCTimeEventMessageEventCSize: attrs[3],
	}), 0
}

func ocNodeResourceToInternal(ocNode *occommon.Node, ocResource *ocresource.Resource, out data.ResourceSpans) {
	// Number of special fields in the Node. See the code below that deals with special fields.
	const specialNodeAttrCount = 7

	// Number of special fields in the Resource.
	const specialResourceAttrCount = 1

	// Calculate maximum total number of attributes. It is OK if we are a bit higher than
	// the exact number since this is only needed for capacity reservation.
	maxTotalAttrCount := 0
	if ocNode != nil {
		maxTotalAttrCount += len(ocNode.Attributes) + specialNodeAttrCount
	}
	if ocResource != nil {
		maxTotalAttrCount += len(ocResource.Labels) + specialResourceAttrCount
	}

	// Create a map where we will place all attributes from the Node and Resource.
	attrs := make(data.AttributesMap, maxTotalAttrCount)

	if ocNode != nil {
		// Copy all Attributes.
		for k, v := range ocNode.Attributes {
			attrs[k] = data.NewAttributeValueString(v)
		}

		// Add all special fields.
		if ocNode.ServiceInfo != nil {
			if ocNode.ServiceInfo.Name != "" {
				attrs[conventions.AttributeServiceName] = data.NewAttributeValueString(
					ocNode.ServiceInfo.Name)
			}
		}
		if ocNode.Identifier != nil {
			if ocNode.Identifier.StartTimestamp != nil {
				attrs[conventions.OCAttributeProcessStartTime] = data.NewAttributeValueString(
					ptypes.TimestampString(ocNode.Identifier.StartTimestamp))
			}
			if ocNode.Identifier.HostName != "" {
				attrs[conventions.AttributeHostHostname] = data.NewAttributeValueString(
					ocNode.Identifier.HostName)
			}
			if ocNode.Identifier.Pid != 0 {
				attrs[conventions.OCAttributeProcessID] = data.NewAttributeValueInt(int64(ocNode.Identifier.Pid))
			}
		}
		if ocNode.LibraryInfo != nil {
			if ocNode.LibraryInfo.CoreLibraryVersion != "" {
				attrs[conventions.AttributeLibraryVersion] = data.NewAttributeValueString(
					ocNode.LibraryInfo.CoreLibraryVersion)
			}
			if ocNode.LibraryInfo.ExporterVersion != "" {
				attrs[conventions.OCAttributeExporterVersion] = data.NewAttributeValueString(
					ocNode.LibraryInfo.ExporterVersion)
			}
			if ocNode.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				attrs[conventions.AttributeLibraryLanguage] = data.NewAttributeValueString(
					ocNode.LibraryInfo.Language.String())
			}
		}
	}

	if ocResource != nil {
		// Copy resource Labels.
		for k, v := range ocResource.Labels {
			attrs[k] = data.NewAttributeValueString(v)
		}
		// Add special fields.
		if ocResource.Type != "" {
			attrs[conventions.OCAttributeResourceType] = data.NewAttributeValueString(ocResource.Type)
		}
	}

	if len(attrs) != 0 {
		out.InitResourceIfNil()
		// TODO: Re-evaluate if we want to construct a map first, or we can construct directly
		// a slice of AttributeKeyValue.
		out.Resource().SetAttributes(data.NewAttributeMap(attrs))
	}
}
