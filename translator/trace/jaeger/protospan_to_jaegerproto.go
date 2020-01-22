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
	"fmt"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	jaeger "github.com/jaegertracing/jaeger/model"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

var (
	unknownProcessProto = &jaeger.Process{ServiceName: "unknown-service-name"}
)

// OCProtoToJaegerProto translates OpenCensus trace data into the Jaeger Proto for GRPC.
func OCProtoToJaegerProto(td consumerdata.TraceData) (*jaeger.Batch, error) {
	jSpans, err := ocSpansToJaegerSpansProto(td.Spans)
	if err != nil {
		return nil, err
	}

	jb := &jaeger.Batch{
		Process: ocNodeAndResourceToJaegerProcessProto(td.Node, td.Resource),
		Spans:   jSpans,
	}

	return jb, nil
}

// Replica of protospan_to_jaegerthrift.ocNodeToJaegerProcess
func ocNodeAndResourceToJaegerProcessProto(node *commonpb.Node, resource *resourcepb.Resource) *jaeger.Process {
	if node == nil {
		return unknownProcessProto
	}

	var jTags []jaeger.KeyValue
	nodeAttribsLen := len(node.Attributes)
	if nodeAttribsLen > 0 {
		jTags = make([]jaeger.KeyValue, 0, nodeAttribsLen)
		for k, v := range node.Attributes {
			str := v
			jTag := jaeger.KeyValue{
				Key:   k,
				VType: jaeger.ValueType_STRING,
				VStr:  str,
			}
			jTags = append(jTags, jTag)
		}
	}

	if node.Identifier != nil {
		if node.Identifier.HostName != "" {
			hostTag := jaeger.KeyValue{
				Key:   "hostname",
				VType: jaeger.ValueType_STRING,
				VStr:  node.Identifier.HostName,
			}
			jTags = append(jTags, hostTag)
		}
		if node.Identifier.Pid != 0 {
			pid := int64(node.Identifier.Pid)
			hostTag := jaeger.KeyValue{
				Key:    "pid",
				VType:  jaeger.ValueType_INT64,
				VInt64: pid,
			}
			jTags = append(jTags, hostTag)
		}
		if node.Identifier.StartTimestamp != nil && node.Identifier.StartTimestamp.Seconds != 0 {
			startTimeStr := ptypes.TimestampString(node.Identifier.StartTimestamp)
			hostTag := jaeger.KeyValue{
				Key:   "start.time",
				VType: jaeger.ValueType_STRING,
				VStr:  startTimeStr,
			}
			jTags = append(jTags, hostTag)
		}
	}

	// Add OpenCensus library information as tags if available
	ocLib := node.LibraryInfo
	if ocLib != nil {
		// Only add language if specified
		if ocLib.Language != commonpb.LibraryInfo_LANGUAGE_UNSPECIFIED {
			languageStr := ocLib.Language.String()
			languageTag := jaeger.KeyValue{
				Key:   opencensusLanguage,
				VType: jaeger.ValueType_STRING,
				VStr:  languageStr,
			}
			jTags = append(jTags, languageTag)
		}
		if ocLib.ExporterVersion != "" {
			exporterTag := jaeger.KeyValue{
				Key:   opencensusExporterVersion,
				VType: jaeger.ValueType_STRING,
				VStr:  ocLib.ExporterVersion,
			}
			jTags = append(jTags, exporterTag)
		}
		if ocLib.CoreLibraryVersion != "" {
			exporterTag := jaeger.KeyValue{
				Key:   opencensusCoreLibVersion,
				VType: jaeger.ValueType_STRING,
				VStr:  ocLib.CoreLibraryVersion,
			}
			jTags = append(jTags, exporterTag)
		}
	}

	var serviceName string
	if node.ServiceInfo != nil && node.ServiceInfo.Name != "" {
		serviceName = node.ServiceInfo.Name
	}

	if resource != nil {
		resourceType := resource.GetType()
		if resourceType != "" {
			resourceTypeTag := jaeger.KeyValue{
				Key:   opencensusResourceType,
				VType: jaeger.ValueType_STRING,
				VStr:  resourceType,
			}
			jTags = append(jTags, resourceTypeTag)
		}
		for k, v := range resource.GetLabels() {
			str := v
			resourceTag := jaeger.KeyValue{
				Key:   k,
				VType: jaeger.ValueType_STRING,
				VStr:  str,
			}
			jTags = append(jTags, resourceTag)
		}
	}

	if serviceName == "" && len(jTags) == 0 {
		// No info to put in the process...
		return nil
	}

	jProc := &jaeger.Process{
		ServiceName: serviceName,
		Tags:        jTags,
	}

	return jProc
}

func truncableStringToStrProto(ts *tracepb.TruncatableString) string {
	if ts == nil {
		return ""
	}
	return ts.Value
}

func ocLinksToJaegerReferencesProto(ocSpanLinks *tracepb.Span_Links) ([]jaeger.SpanRef, error) {
	if ocSpanLinks == nil || ocSpanLinks.Link == nil {
		return nil, nil
	}

	ocLinks := ocSpanLinks.Link
	jRefs := make([]jaeger.SpanRef, 0, len(ocLinks))
	for _, ocLink := range ocLinks {
		var traceID jaeger.TraceID
		// Checks for nil/wrongLen traceID are already done in ocSpansToJaegerSpansProto()
		traceIDHigh, traceIDLow, err := tracetranslator.BytesToUInt64TraceID(ocLink.TraceId)
		if err != nil {
			return nil, err
		}
		traceID = jaeger.TraceID{
			Low:  traceIDLow,
			High: traceIDHigh,
		}

		var spanID jaeger.SpanID
		uSpanID, err := tracetranslator.BytesToUInt64SpanID(ocLink.SpanId)
		if err != nil {
			return nil, err
		}

		if uSpanID == 0 {
			return nil, errZeroSpanID
		}
		spanID = jaeger.SpanID(uSpanID)

		var jRefType jaeger.SpanRefType
		switch ocLink.Type {
		case tracepb.Span_Link_PARENT_LINKED_SPAN:
			jRefType = jaeger.SpanRefType_CHILD_OF
		default:
			// TODO: Jaeger doesn't have a unknown SpanRefType, it has FOLLOWS_FROM or CHILD_OF
			// at first mapping all others to FOLLOWS_FROM.
			jRefType = jaeger.SpanRefType_FOLLOWS_FROM
		}

		jRef := jaeger.SpanRef{
			TraceID: traceID,
			SpanID:  spanID,
			RefType: jRefType,
		}
		jRefs = append(jRefs, jRef)
	}

	return jRefs, nil
}

func timestampToTimeProto(ts *timestamp.Timestamp) (t time.Time) {
	if ts == nil {
		return
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos)).UTC()
}

// Replica of protospan_to_jaegerthrift.ocSpanAttributesToJaegerTags
func ocSpanAttributesToJaegerTagsProto(ocAttribs *tracepb.Span_Attributes) []jaeger.KeyValue {
	if ocAttribs == nil {
		return nil
	}

	// Pre-allocate assuming that few attributes, if any at all, are nil.
	jTags := make([]jaeger.KeyValue, 0, len(ocAttribs.AttributeMap))
	for key, attrib := range ocAttribs.AttributeMap {
		if attrib == nil || attrib.Value == nil {
			continue
		}

		jTag := jaeger.KeyValue{Key: key}
		switch attribValue := attrib.Value.(type) {
		case *tracepb.AttributeValue_StringValue:
			// Jaeger-to-OC maps binary tags to string attributes and encodes them as
			// base64 strings. Blindingly attempting to decode base64 seems too much.
			str := truncableStringToStrProto(attribValue.StringValue)
			jTag.VStr = str
			jTag.VType = jaeger.ValueType_STRING
		case *tracepb.AttributeValue_IntValue:
			i := attribValue.IntValue
			jTag.VInt64 = i
			jTag.VType = jaeger.ValueType_INT64
		case *tracepb.AttributeValue_BoolValue:
			b := attribValue.BoolValue
			jTag.VBool = b
			jTag.VType = jaeger.ValueType_BOOL
		case *tracepb.AttributeValue_DoubleValue:
			d := attribValue.DoubleValue
			jTag.VFloat64 = d
			jTag.VType = jaeger.ValueType_FLOAT64
		default:
			str := "<Unknown OpenCensus Attribute for key \"" + key + "\">"
			jTag.VStr = str
			jTag.VType = jaeger.ValueType_STRING
		}
		jTags = append(jTags, jTag)
	}

	return jTags
}

func ocTimeEventsToJaegerLogsProto(ocSpanTimeEvents *tracepb.Span_TimeEvents) []jaeger.Log {
	if ocSpanTimeEvents == nil || ocSpanTimeEvents.TimeEvent == nil {
		return nil
	}

	ocTimeEvents := ocSpanTimeEvents.TimeEvent

	// Assume that in general no time events are going to produce nil Jaeger logs.
	jLogs := make([]jaeger.Log, 0, len(ocTimeEvents))
	for _, ocTimeEvent := range ocTimeEvents {
		jLog := jaeger.Log{
			Timestamp: timestampToTimeProto(ocTimeEvent.Time),
		}
		switch teValue := ocTimeEvent.Value.(type) {
		case *tracepb.Span_TimeEvent_Annotation_:
			jLog.Fields = ocAnnotationToJagerTagsProto(teValue.Annotation)
		case *tracepb.Span_TimeEvent_MessageEvent_:
			jLog.Fields = ocMessageEventToJaegerTagsProto(teValue.MessageEvent)
		default:
			msg := "An unknown OpenCensus TimeEvent type was detected when translating to Jaeger"
			jKV := jaeger.KeyValue{
				Key:   ocTimeEventUnknownType,
				VStr:  msg,
				VType: jaeger.ValueType_STRING,
			}
			jLog.Fields = append(jLog.Fields, jKV)
		}

		jLogs = append(jLogs, jLog)
	}

	return jLogs
}

func ocAnnotationToJagerTagsProto(annotation *tracepb.Span_TimeEvent_Annotation) []jaeger.KeyValue {
	if annotation == nil {
		return []jaeger.KeyValue{}
	}

	strDescription := truncableStringToStr(annotation.Description)
	if strDescription == "" {
		return ocSpanAttributesToJaegerTagsProto(annotation.Attributes)
	}

	// TODO: Find a better tag for Annotation description.
	jKV := jaeger.KeyValue{
		Key:   ocTimeEventAnnotationDescription,
		VStr:  strDescription,
		VType: jaeger.ValueType_STRING,
	}
	return append(ocSpanAttributesToJaegerTagsProto(annotation.Attributes), jKV)
}

func ocMessageEventToJaegerTagsProto(msgEvent *tracepb.Span_TimeEvent_MessageEvent) []jaeger.KeyValue {
	if msgEvent == nil {
		return []jaeger.KeyValue{}
	}

	// TODO: Find a better tag for Message event.
	msgTypeStr := msgEvent.Type.String()
	jaegerKVs := []jaeger.KeyValue{
		{
			Key:   ocTimeEventMessageEventType,
			VStr:  msgTypeStr,
			VType: jaeger.ValueType_STRING,
		},
		{
			Key:    ocTimeEventMessageEventID,
			VInt64: int64(msgEvent.Id),
			VType:  jaeger.ValueType_INT64,
		},
		{
			Key:    ocTimeEventMessageEventUSize,
			VInt64: int64(msgEvent.UncompressedSize),
			VType:  jaeger.ValueType_INT64,
		},
		{
			Key:    ocTimeEventMessageEventCSize,
			VInt64: int64(msgEvent.CompressedSize),
			VType:  jaeger.ValueType_INT64,
		},
	}
	return jaegerKVs
}

// Replica of protospan_to_jaegerthrift appendJaegerTagFromOCSpanKind
func appendJaegerTagFromOCSpanKindProto(jTags []jaeger.KeyValue, ocSpanKind tracepb.Span_SpanKind) []jaeger.KeyValue {
	// TODO: (@pjanotti): Replace any OpenTracing literals by importing github.com/opentracing/opentracing-go/ext?
	var tagValue string
	switch ocSpanKind {
	case tracepb.Span_CLIENT:
		tagValue = "client"
	case tracepb.Span_SERVER:
		tagValue = "server"
	}

	if tagValue != "" {
		jTag := jaeger.KeyValue{
			Key:   tracetranslator.TagSpanKind,
			VStr:  tagValue,
			VType: jaeger.ValueType_STRING,
		}
		jTags = append(jTags, jTag)
	}

	return jTags
}

func appendJaegerTagFromOCTracestateProto(jTags []jaeger.KeyValue, ocSpanTracestate *tracepb.Span_Tracestate) []jaeger.KeyValue {
	if ocSpanTracestate == nil {
		return jTags
	}

	for _, tsEntry := range ocSpanTracestate.Entries {
		jTag := jaeger.KeyValue{
			Key:   tsEntry.Key,
			VStr:  tsEntry.Value,
			VType: jaeger.ValueType_STRING,
		}
		jTags = append(jTags, jTag)
	}

	return jTags
}

func appendJaegerTagFromOCStatusProto(jTags []jaeger.KeyValue, ocStatus *tracepb.Status) []jaeger.KeyValue {
	if ocStatus == nil {
		return jTags
	}

	jTags = append(jTags, jaeger.KeyValue{
		Key:    tracetranslator.TagStatusCode,
		VInt64: int64(ocStatus.Code),
		VType:  jaeger.ValueType_INT64,
	})

	if ocStatus.Message != "" {
		jTags = append(jTags, jaeger.KeyValue{
			Key:   tracetranslator.TagStatusMsg,
			VStr:  ocStatus.Message,
			VType: jaeger.ValueType_STRING,
		})
	}

	return jTags
}

func appendJaegerTagFromOCSameProcessAsParentSpanProto(jTags []jaeger.KeyValue, ocIsSameProcessAsParentSpan *wrappers.BoolValue) []jaeger.KeyValue {
	if ocIsSameProcessAsParentSpan == nil {
		return jTags
	}

	jTag := jaeger.KeyValue{
		Key:   ocSameProcessAsParentSpan,
		VBool: ocIsSameProcessAsParentSpan.Value,
		VType: jaeger.ValueType_BOOL,
	}
	jTags = append(jTags, jTag)

	return jTags
}

func appendJaegerTagFromOCChildSpanCountProto(jTags []jaeger.KeyValue, ocChildSpanCount *wrappers.UInt32Value) []jaeger.KeyValue {
	if ocChildSpanCount == nil {
		return jTags
	}

	jTag := jaeger.KeyValue{
		Key:    ocSpanChildCount,
		VInt64: int64(ocChildSpanCount.Value),
		VType:  jaeger.ValueType_INT64,
	}
	jTags = append(jTags, jTag)

	return jTags
}

func ocSpansToJaegerSpansProto(ocSpans []*tracepb.Span) ([]*jaeger.Span, error) {
	if ocSpans == nil {
		return nil, nil
	}

	// Pre-allocate assuming that few, if any spans, are nil.
	jSpans := make([]*jaeger.Span, 0, len(ocSpans))
	for _, ocSpan := range ocSpans {
		var traceID jaeger.TraceID
		traceIDHigh, traceIDLow, err := tracetranslator.BytesToUInt64TraceID(ocSpan.TraceId)
		if err != nil {
			return nil, err
		}
		if traceIDLow == 0 && traceIDHigh == 0 {
			return nil, errZeroTraceID
		}
		traceID = jaeger.TraceID{
			Low:  traceIDLow,
			High: traceIDHigh,
		}

		var spanID jaeger.SpanID
		uSpanID, _ := tracetranslator.BytesToUInt64SpanID(ocSpan.SpanId)
		if uSpanID == 0 {
			return nil, errZeroSpanID
		}
		spanID = jaeger.SpanID(uSpanID)

		jReferences, err := ocLinksToJaegerReferencesProto(ocSpan.Links)
		if err != nil {
			return nil, fmt.Errorf("error converting OC links to Jaeger references: %v", err)
		}

		// OC ParentSpanId can be nil/empty: only attempt conversion if not nil/empty.
		if len(ocSpan.ParentSpanId) != 0 {
			uParentSpanID, err := tracetranslator.BytesToUInt64SpanID(ocSpan.ParentSpanId)
			if err != nil {
				return nil, fmt.Errorf("OC incorrect parent span ID: %v", err)
			}

			jReferences = append(jReferences, jaeger.SpanRef{
				TraceID: traceID,
				SpanID:  jaeger.SpanID(uParentSpanID),
				RefType: jaeger.SpanRefType_CHILD_OF,
			})
		}

		startTime := timestampToTimeProto(ocSpan.StartTime)
		jSpan := &jaeger.Span{
			TraceID:       traceID,
			SpanID:        spanID,
			OperationName: truncableStringToStrProto(ocSpan.Name),
			References:    jReferences,
			StartTime:     startTime,
			Duration:      timestampToTimeProto(ocSpan.EndTime).Sub(startTime),
			Tags:          ocSpanAttributesToJaegerTagsProto(ocSpan.Attributes),
			Logs:          ocTimeEventsToJaegerLogsProto(ocSpan.TimeEvents),
			// Flags: TODO Flags might be used once sampling and other features are implemented in OC.
		}

		// Only add the "span.kind" tag if not set in the OC span attributes.
		if !tracetranslator.OCAttributeKeyExist(ocSpan.Attributes, tracetranslator.TagSpanKind) {
			jSpan.Tags = appendJaegerTagFromOCSpanKindProto(jSpan.Tags, ocSpan.Kind)
		}
		// Only add status tags if neither status.code and status.message are set in the OC span attributes.
		if !tracetranslator.OCAttributeKeyExist(ocSpan.Attributes, tracetranslator.TagStatusCode) &&
			!tracetranslator.OCAttributeKeyExist(ocSpan.Attributes, tracetranslator.TagStatusMsg) {
			jSpan.Tags = appendJaegerTagFromOCStatusProto(jSpan.Tags, ocSpan.Status)
		}
		jSpan.Tags = appendJaegerTagFromOCTracestateProto(jSpan.Tags, ocSpan.Tracestate)
		jSpan.Tags = appendJaegerTagFromOCSameProcessAsParentSpanProto(jSpan.Tags, ocSpan.SameProcessAsParentSpan)
		jSpan.Tags = appendJaegerTagFromOCChildSpanCountProto(jSpan.Tags, ocSpan.ChildSpanCount)
		jSpans = append(jSpans, jSpan)
	}

	return jSpans, nil
}
