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

package jaeger

import (
	"fmt"

	"github.com/jaegertracing/jaeger/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

// InternalTracesToJaegerProto translates internal trace data into the Jaeger Proto for GRPC.
// Returns slice of translated Jaeger batches and error if translation failed.
func InternalTracesToJaegerProto(td pdata.Traces) ([]*model.Batch, error) {
	resourceSpans := td.ResourceSpans()

	if resourceSpans.Len() == 0 {
		return nil, nil
	}

	batches := make([]*model.Batch, 0, resourceSpans.Len())

	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		batch, err := resourceSpansToJaegerProto(rs)
		if err != nil {
			return nil, err
		}
		if batch != nil {
			batches = append(batches, batch)
		}
	}

	return batches, nil
}

func resourceSpansToJaegerProto(rs pdata.ResourceSpans) (*model.Batch, error) {
	resource := rs.Resource()
	ilss := rs.InstrumentationLibrarySpans()

	if resource.Attributes().Len() == 0 && ilss.Len() == 0 {
		return nil, nil
	}

	batch := &model.Batch{
		Process: resourceToJaegerProtoProcess(resource),
	}

	if ilss.Len() == 0 {
		return batch, nil
	}

	// Approximate the number of the spans as the number of the spans in the first
	// instrumentation library info.
	jSpans := make([]*model.Span, 0, ilss.At(0).Spans().Len())

	for i := 0; i < ilss.Len(); i++ {
		ils := ilss.At(i)
		spans := ils.Spans()
		for j := 0; j < spans.Len(); j++ {
			span := spans.At(j)
			jSpan, err := spanToJaegerProto(span, ils.InstrumentationLibrary())
			if err != nil {
				return nil, err
			}
			if jSpan != nil {
				jSpans = append(jSpans, jSpan)
			}
		}
	}

	batch.Spans = jSpans

	return batch, nil
}

func resourceToJaegerProtoProcess(resource pdata.Resource) *model.Process {
	process := &model.Process{}
	attrs := resource.Attributes()
	if attrs.Len() == 0 {
		process.ServiceName = tracetranslator.ResourceNoServiceName
		return process
	}
	attrsCount := attrs.Len()
	if serviceName, ok := attrs.Get(conventions.AttributeServiceName); ok {
		process.ServiceName = serviceName.StringVal()
		attrsCount--
	}
	if attrsCount == 0 {
		return process
	}

	tags := make([]model.KeyValue, 0, attrsCount)
	process.Tags = appendTagsFromResourceAttributes(tags, attrs)
	return process

}

func appendTagsFromResourceAttributes(dest []model.KeyValue, attrs pdata.AttributeMap) []model.KeyValue {
	if attrs.Len() == 0 {
		return dest
	}

	attrs.ForEach(func(key string, attr pdata.AttributeValue) {
		if key == conventions.AttributeServiceName {
			return
		}
		dest = append(dest, attributeToJaegerProtoTag(key, attr))
	})
	return dest
}

func appendTagsFromAttributes(dest []model.KeyValue, attrs pdata.AttributeMap) []model.KeyValue {
	if attrs.Len() == 0 {
		return dest
	}
	attrs.ForEach(func(key string, attr pdata.AttributeValue) {
		dest = append(dest, attributeToJaegerProtoTag(key, attr))
	})
	return dest
}

func attributeToJaegerProtoTag(key string, attr pdata.AttributeValue) model.KeyValue {
	tag := model.KeyValue{Key: key}
	switch attr.Type() {
	case pdata.AttributeValueSTRING:
		// Jaeger-to-Internal maps binary tags to string attributes and encodes them as
		// base64 strings. Blindingly attempting to decode base64 seems too much.
		tag.VType = model.ValueType_STRING
		tag.VStr = attr.StringVal()
	case pdata.AttributeValueINT:
		tag.VType = model.ValueType_INT64
		tag.VInt64 = attr.IntVal()
	case pdata.AttributeValueBOOL:
		tag.VType = model.ValueType_BOOL
		tag.VBool = attr.BoolVal()
	case pdata.AttributeValueDOUBLE:
		tag.VType = model.ValueType_FLOAT64
		tag.VFloat64 = attr.DoubleVal()
	case pdata.AttributeValueMAP, pdata.AttributeValueARRAY:
		tag.VType = model.ValueType_STRING
		tag.VStr = tracetranslator.AttributeValueToString(attr, false)
	}
	return tag
}

func spanToJaegerProto(span pdata.Span, libraryTags pdata.InstrumentationLibrary) (*model.Span, error) {
	traceID, err := traceIDToJaegerProto(span.TraceID())
	if err != nil {
		return nil, err
	}

	spanID, err := spanIDToJaegerProto(span.SpanID())
	if err != nil {
		return nil, err
	}

	jReferences, err := makeJaegerProtoReferences(span.Links(), span.ParentSpanID(), traceID)
	if err != nil {
		return nil, fmt.Errorf("error converting span links to Jaeger references: %w", err)
	}

	startTime := pdata.UnixNanoToTime(span.StartTime())

	return &model.Span{
		TraceID:       traceID,
		SpanID:        spanID,
		OperationName: span.Name(),
		References:    jReferences,
		StartTime:     startTime,
		Duration:      pdata.UnixNanoToTime(span.EndTime()).Sub(startTime),
		Tags:          getJaegerProtoSpanTags(span, libraryTags),
		Logs:          spanEventsToJaegerProtoLogs(span.Events()),
	}, nil
}

func getJaegerProtoSpanTags(span pdata.Span, instrumentationLibrary pdata.InstrumentationLibrary) []model.KeyValue {
	var spanKindTag, statusCodeTag, errorTag, statusMsgTag model.KeyValue
	var spanKindTagFound, statusCodeTagFound, errorTagFound, statusMsgTagFound bool

	libraryTags, libraryTagsFound := getTagsFromInstrumentationLibrary(instrumentationLibrary)

	tagsCount := span.Attributes().Len() + len(libraryTags)

	spanKindTag, spanKindTagFound = getTagFromSpanKind(span.Kind())
	if spanKindTagFound {
		tagsCount++
	}
	status := span.Status()
	statusCodeTag, statusCodeTagFound = getTagFromStatusCode(status.Code())
	if statusCodeTagFound {
		tagsCount++
	}

	errorTag, errorTagFound = getErrorTagFromStatusCode(status.Code())
	if errorTagFound {
		tagsCount++
	}

	statusMsgTag, statusMsgTagFound = getTagFromStatusMsg(status.Message())
	if statusMsgTagFound {
		tagsCount++
	}

	traceStateTags, traceStateTagsFound := getTagsFromTraceState(span.TraceState())
	if traceStateTagsFound {
		tagsCount += len(traceStateTags)
	}

	if tagsCount == 0 {
		return nil
	}

	tags := make([]model.KeyValue, 0, tagsCount)
	if libraryTagsFound {
		tags = append(tags, libraryTags...)
	}
	tags = appendTagsFromAttributes(tags, span.Attributes())
	if spanKindTagFound {
		tags = append(tags, spanKindTag)
	}
	if statusCodeTagFound {
		tags = append(tags, statusCodeTag)
	}
	if errorTagFound {
		tags = append(tags, errorTag)
	}
	if statusMsgTagFound {
		tags = append(tags, statusMsgTag)
	}
	if traceStateTagsFound {
		tags = append(tags, traceStateTags...)
	}
	return tags
}

func traceIDToJaegerProto(traceID pdata.TraceID) (model.TraceID, error) {
	traceIDHigh, traceIDLow := tracetranslator.TraceIDToUInt64Pair(traceID)
	if traceIDLow == 0 && traceIDHigh == 0 {
		return model.TraceID{}, errZeroTraceID
	}
	return model.TraceID{
		Low:  traceIDLow,
		High: traceIDHigh,
	}, nil
}

func spanIDToJaegerProto(spanID pdata.SpanID) (model.SpanID, error) {
	uSpanID := tracetranslator.BytesToUInt64SpanID(spanID.Bytes())
	if uSpanID == 0 {
		return model.SpanID(0), errZeroSpanID
	}
	return model.SpanID(uSpanID), nil
}

// makeJaegerProtoReferences constructs jaeger span references based on parent span ID and span links
func makeJaegerProtoReferences(
	links pdata.SpanLinkSlice,
	parentSpanID pdata.SpanID,
	traceID model.TraceID,
) ([]model.SpanRef, error) {
	parentSpanIDSet := parentSpanID.IsValid()
	if !parentSpanIDSet && links.Len() == 0 {
		return nil, nil
	}

	refsCount := links.Len()
	if parentSpanIDSet {
		refsCount++
	}

	refs := make([]model.SpanRef, 0, refsCount)

	// Put parent span ID at the first place because usually backends look for it
	// as the first CHILD_OF item in the model.SpanRef slice.
	if parentSpanIDSet {
		jParentSpanID, err := spanIDToJaegerProto(parentSpanID)
		if err != nil {
			return nil, fmt.Errorf("OC incorrect parent span ID: %v", err)
		}

		refs = append(refs, model.SpanRef{
			TraceID: traceID,
			SpanID:  jParentSpanID,
			RefType: model.SpanRefType_CHILD_OF,
		})
	}

	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		traceID, err := traceIDToJaegerProto(link.TraceID())
		if err != nil {
			continue // skip invalid link
		}

		spanID, err := spanIDToJaegerProto(link.SpanID())
		if err != nil {
			continue // skip invalid link
		}

		refs = append(refs, model.SpanRef{
			TraceID: traceID,
			SpanID:  spanID,

			// Since Jaeger RefType is not captured in internal data,
			// use SpanRefType_FOLLOWS_FROM by default.
			// SpanRefType_CHILD_OF supposed to be set only from parentSpanID.
			RefType: model.SpanRefType_FOLLOWS_FROM,
		})
	}

	return refs, nil
}

func spanEventsToJaegerProtoLogs(events pdata.SpanEventSlice) []model.Log {
	if events.Len() == 0 {
		return nil
	}

	logs := make([]model.Log, 0, events.Len())
	for i := 0; i < events.Len(); i++ {
		event := events.At(i)
		fields := make([]model.KeyValue, 0, event.Attributes().Len()+1)
		if event.Name() != "" {
			fields = append(fields, model.KeyValue{
				Key:   tracetranslator.TagMessage,
				VType: model.ValueType_STRING,
				VStr:  event.Name(),
			})
		}
		fields = appendTagsFromAttributes(fields, event.Attributes())
		logs = append(logs, model.Log{
			Timestamp: pdata.UnixNanoToTime(event.Timestamp()),
			Fields:    fields,
		})
	}

	return logs
}

func getTagFromSpanKind(spanKind pdata.SpanKind) (model.KeyValue, bool) {
	var tagStr string
	switch spanKind {
	case pdata.SpanKindCLIENT:
		tagStr = string(tracetranslator.OpenTracingSpanKindClient)
	case pdata.SpanKindSERVER:
		tagStr = string(tracetranslator.OpenTracingSpanKindServer)
	case pdata.SpanKindPRODUCER:
		tagStr = string(tracetranslator.OpenTracingSpanKindProducer)
	case pdata.SpanKindCONSUMER:
		tagStr = string(tracetranslator.OpenTracingSpanKindConsumer)
	case pdata.SpanKindINTERNAL:
		tagStr = string(tracetranslator.OpenTracingSpanKindInternal)
	default:
		return model.KeyValue{}, false
	}

	return model.KeyValue{
		Key:   tracetranslator.TagSpanKind,
		VType: model.ValueType_STRING,
		VStr:  tagStr,
	}, true
}

func getTagFromStatusCode(statusCode pdata.StatusCode) (model.KeyValue, bool) {
	return model.KeyValue{
		Key:    tracetranslator.TagStatusCode,
		VInt64: int64(statusCode),
		VType:  model.ValueType_INT64,
	}, true
}

func getErrorTagFromStatusCode(statusCode pdata.StatusCode) (model.KeyValue, bool) {
	if statusCode == pdata.StatusCodeError {
		return model.KeyValue{
			Key:   tracetranslator.TagError,
			VBool: true,
			VType: model.ValueType_BOOL,
		}, true
	}
	return model.KeyValue{}, false

}

func getTagFromStatusMsg(statusMsg string) (model.KeyValue, bool) {
	if statusMsg == "" {
		return model.KeyValue{}, false
	}
	return model.KeyValue{
		Key:   tracetranslator.TagStatusMsg,
		VStr:  statusMsg,
		VType: model.ValueType_STRING,
	}, true
}

func getTagsFromTraceState(traceState pdata.TraceState) ([]model.KeyValue, bool) {
	keyValues := make([]model.KeyValue, 0)
	exists := traceState != pdata.TraceStateEmpty
	if exists {
		// TODO Bring this inline with solution for jaegertracing/jaeger-client-java #702 once available
		kv := model.KeyValue{
			Key:   tracetranslator.TagW3CTraceState,
			VStr:  string(traceState),
			VType: model.ValueType_STRING,
		}
		keyValues = append(keyValues, kv)
	}
	return keyValues, exists
}

func getTagsFromInstrumentationLibrary(il pdata.InstrumentationLibrary) ([]model.KeyValue, bool) {
	keyValues := make([]model.KeyValue, 0)
	if ilName := il.Name(); ilName != "" {
		kv := model.KeyValue{
			Key:   tracetranslator.TagInstrumentationName,
			VStr:  ilName,
			VType: model.ValueType_STRING,
		}
		keyValues = append(keyValues, kv)
	}
	if ilVersion := il.Version(); ilVersion != "" {
		kv := model.KeyValue{
			Key:   tracetranslator.TagInstrumentationVersion,
			VStr:  ilVersion,
			VType: model.ValueType_STRING,
		}
		keyValues = append(keyValues, kv)
	}

	return keyValues, true
}
