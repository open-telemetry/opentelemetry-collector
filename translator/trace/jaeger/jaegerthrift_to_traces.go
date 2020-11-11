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
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func ThriftBatchToInternalTraces(batch *jaeger.Batch) pdata.Traces {
	traceData := pdata.NewTraces()
	jProcess := batch.GetProcess()
	jSpans := batch.GetSpans()

	if jProcess == nil && len(jSpans) == 0 {
		return traceData
	}

	rss := traceData.ResourceSpans()
	rss.Resize(1)
	rs := rss.At(0)
	jThriftProcessToInternalResource(jProcess, rs.Resource())

	if len(jSpans) == 0 {
		return traceData
	}

	ilss := rs.InstrumentationLibrarySpans()
	ilss.Resize(1)
	jThriftSpansToInternal(jSpans, ilss.At(0).Spans())

	return traceData
}

func jThriftProcessToInternalResource(process *jaeger.Process, dest pdata.Resource) {
	if process == nil {
		return
	}

	serviceName := process.GetServiceName()
	tags := process.GetTags()
	if serviceName == "" && tags == nil {
		return
	}

	attrs := dest.Attributes()
	if serviceName != "" {
		attrs.InitEmptyWithCapacity(len(tags) + 1)
		attrs.UpsertString(conventions.AttributeServiceName, serviceName)
	} else {
		attrs.InitEmptyWithCapacity(len(tags))
	}
	jThriftTagsToInternalAttributes(tags, attrs)

	// Handle special keys translations.
	translateHostnameAttr(attrs)
	translateJaegerVersionAttr(attrs)
}

func jThriftSpansToInternal(spans []*jaeger.Span, dest pdata.SpanSlice) {
	if len(spans) == 0 {
		return
	}

	dest.Resize(len(spans))
	i := 0
	for _, span := range spans {
		if span == nil || reflect.DeepEqual(span, blankJaegerProtoSpan) {
			continue
		}
		jThriftSpanToInternal(span, dest.At(i))
		i++
	}

	if i < len(spans) {
		dest.Resize(i)
	}
}

func jThriftSpanToInternal(span *jaeger.Span, dest pdata.Span) {
	dest.SetTraceID(tracetranslator.Int64ToTraceID(span.TraceIdHigh, span.TraceIdLow))
	dest.SetSpanID(tracetranslator.Int64ToSpanID(span.SpanId))
	dest.SetName(span.OperationName)
	dest.SetStartTime(microsecondsToUnixNano(span.StartTime))
	dest.SetEndTime(microsecondsToUnixNano(span.StartTime + span.Duration))

	parentSpanID := span.ParentSpanId
	if parentSpanID != 0 {
		dest.SetParentSpanID(tracetranslator.Int64ToSpanID(parentSpanID))
	}

	attrs := dest.Attributes()
	attrs.InitEmptyWithCapacity(len(span.Tags))
	jThriftTagsToInternalAttributes(span.Tags, attrs)
	setInternalSpanStatus(attrs, dest.Status())
	if spanKindAttr, ok := attrs.Get(tracetranslator.TagSpanKind); ok {
		dest.SetKind(jSpanKindToInternal(spanKindAttr.StringVal()))
		attrs.Delete(tracetranslator.TagSpanKind)
	}

	// drop the attributes slice if all of them were replaced during translation
	if attrs.Len() == 0 {
		attrs.InitFromMap(nil)
	}

	jThriftLogsToSpanEvents(span.Logs, dest.Events())
	jThriftReferencesToSpanLinks(span.References, parentSpanID, dest.Links())
}

// jThriftTagsToInternalAttributes sets internal span links based on jaeger span references skipping excludeParentID
func jThriftTagsToInternalAttributes(tags []*jaeger.Tag, dest pdata.AttributeMap) {
	for _, tag := range tags {
		switch tag.GetVType() {
		case jaeger.TagType_STRING:
			dest.UpsertString(tag.Key, tag.GetVStr())
		case jaeger.TagType_BOOL:
			dest.UpsertBool(tag.Key, tag.GetVBool())
		case jaeger.TagType_LONG:
			dest.UpsertInt(tag.Key, tag.GetVLong())
		case jaeger.TagType_DOUBLE:
			dest.UpsertDouble(tag.Key, tag.GetVDouble())
		case jaeger.TagType_BINARY:
			dest.UpsertString(tag.Key, base64.StdEncoding.EncodeToString(tag.GetVBinary()))
		default:
			dest.UpsertString(tag.Key, fmt.Sprintf("<Unknown Jaeger TagType %q>", tag.GetVType()))
		}
	}
}

func jThriftLogsToSpanEvents(logs []*jaeger.Log, dest pdata.SpanEventSlice) {
	if len(logs) == 0 {
		return
	}

	dest.Resize(len(logs))

	for i, log := range logs {
		event := dest.At(i)

		event.SetTimestamp(microsecondsToUnixNano(log.Timestamp))
		if len(log.Fields) == 0 {
			continue
		}

		attrs := event.Attributes()
		attrs.InitEmptyWithCapacity(len(log.Fields))
		jThriftTagsToInternalAttributes(log.Fields, attrs)
		if name, ok := attrs.Get(tracetranslator.TagMessage); ok {
			event.SetName(name.StringVal())
			attrs.Delete(tracetranslator.TagMessage)
		}
	}
}

func jThriftReferencesToSpanLinks(refs []*jaeger.SpanRef, excludeParentID int64, dest pdata.SpanLinkSlice) {
	if len(refs) == 0 || len(refs) == 1 && refs[0].SpanId == excludeParentID && refs[0].RefType == jaeger.SpanRefType_CHILD_OF {
		return
	}

	dest.Resize(len(refs))
	i := 0
	for _, ref := range refs {
		link := dest.At(i)
		if ref.SpanId == excludeParentID && ref.RefType == jaeger.SpanRefType_CHILD_OF {
			continue
		}

		link.SetTraceID(tracetranslator.Int64ToTraceID(ref.TraceIdHigh, ref.TraceIdLow))
		link.SetSpanID(pdata.NewSpanID(tracetranslator.Int64ToByteSpanID(ref.SpanId)))
		i++
	}

	// Reduce slice size in case if excludeParentID was skipped
	if i < len(refs) {
		dest.Resize(i)
	}
}

// microsecondsToUnixNano converts epoch microseconds to pdata.TimestampUnixNano
func microsecondsToUnixNano(ms int64) pdata.TimestampUnixNano {
	return pdata.TimestampUnixNano(uint64(ms) * 1000)
}
