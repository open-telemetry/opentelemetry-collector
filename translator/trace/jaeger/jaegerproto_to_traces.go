// Copyright The OpenTelemetry Authors
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
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

var blankJaegerProtoSpan = new(jaeger.Span)

// ProtoBatchesToInternalTraces converts multiple Jaeger proto batches to internal traces
func ProtoBatchesToInternalTraces(batches []*model.Batch) pdata.Traces {
	traceData := pdata.NewTraces()
	if len(batches) == 0 {
		return traceData
	}

	rss := traceData.ResourceSpans()
	rss.Resize(len(batches))

	i := 0
	for _, batch := range batches {
		if batch.GetProcess() == nil && len(batch.GetSpans()) == 0 {
			continue
		}

		protoBatchToResourceSpans(*batch, rss.At(i))
		i++
	}

	// reduce traceData.ResourceSpans slice if some batched were skipped
	if i < len(batches) {
		rss.Resize(i)
	}

	return traceData
}

// ProtoBatchToInternalTraces converts Jeager proto batch to internal traces
func ProtoBatchToInternalTraces(batch model.Batch) pdata.Traces {
	traceData := pdata.NewTraces()

	if batch.GetProcess() == nil && len(batch.GetSpans()) == 0 {
		return traceData
	}

	rss := traceData.ResourceSpans()
	rss.Resize(1)
	protoBatchToResourceSpans(batch, rss.At(0))

	return traceData
}

func protoBatchToResourceSpans(batch model.Batch, dest pdata.ResourceSpans) {
	jSpans := batch.GetSpans()

	jProcessToInternalResource(batch.GetProcess(), dest.Resource())

	if len(jSpans) == 0 {
		return
	}

	ilss := dest.InstrumentationLibrarySpans()
	ilss.Resize(1)
	jSpansToInternal(jSpans, ilss.At(0).Spans())
}

func jProcessToInternalResource(process *model.Process, dest pdata.Resource) {
	if process == nil {
		return
	}

	dest.InitEmpty()

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
	jTagsToInternalAttributes(tags, attrs)

	// Handle special keys translations.
	translateHostnameAttr(attrs)
	translateJaegerVersionAttr(attrs)
}

// translateHostnameAttr translates "hostname" atttribute
func translateHostnameAttr(attrs pdata.AttributeMap) {
	hostname, hostnameFound := attrs.Get("hostname")
	_, convHostnameFound := attrs.Get(conventions.AttributeHostHostname)
	if hostnameFound && !convHostnameFound {
		attrs.Insert(conventions.AttributeHostHostname, hostname)
		attrs.Delete("hostname")
	}
}

// translateHostnameAttr translates "jaeger.version" atttribute
func translateJaegerVersionAttr(attrs pdata.AttributeMap) {
	jaegerVersion, jaegerVersionFound := attrs.Get("jaeger.version")
	_, exporterVersionFound := attrs.Get(conventions.OCAttributeExporterVersion)
	if jaegerVersionFound && !exporterVersionFound {
		attrs.InsertString(conventions.OCAttributeExporterVersion, "Jaeger-"+jaegerVersion.StringVal())
		attrs.Delete("jaeger.version")
	}
}

func jSpansToInternal(spans []*model.Span, dest pdata.SpanSlice) {
	if len(spans) == 0 {
		return
	}

	dest.Resize(len(spans))
	i := 0
	for _, span := range spans {
		if span == nil || reflect.DeepEqual(span, blankJaegerProtoSpan) {
			continue
		}
		jSpanToInternal(span, dest.At(i))
		i++
	}

	if i < len(spans) {
		dest.Resize(i)
	}
}

func jSpanToInternal(span *model.Span, dest pdata.Span) {
	dest.SetTraceID(pdata.TraceID(tracetranslator.UInt64ToByteTraceID(span.TraceID.High, span.TraceID.Low)))
	dest.SetSpanID(pdata.SpanID(tracetranslator.UInt64ToByteSpanID(uint64(span.SpanID))))
	dest.SetName(span.OperationName)
	dest.SetStartTime(pdata.TimestampUnixNano(uint64(span.StartTime.UnixNano())))
	dest.SetEndTime(pdata.TimestampUnixNano(uint64(span.StartTime.Add(span.Duration).UnixNano())))

	parentSpanID := span.ParentSpanID()
	if parentSpanID != model.SpanID(0) {
		dest.SetParentSpanID(pdata.SpanID(tracetranslator.UInt64ToByteSpanID(uint64(parentSpanID))))
	}

	attrs := dest.Attributes()
	attrs.InitEmptyWithCapacity(len(span.Tags))
	jTagsToInternalAttributes(span.Tags, attrs)
	setInternalSpanStatus(attrs, dest.Status())
	if spanKindAttr, ok := attrs.Get(tracetranslator.TagSpanKind); ok {
		dest.SetKind(jSpanKindToInternal(spanKindAttr.StringVal()))
	}

	// drop the attributes slice if all of them were replaced during translation
	if attrs.Len() == 0 {
		attrs.InitFromMap(nil)
	}

	jLogsToSpanEvents(span.Logs, dest.Events())
	jReferencesToSpanLinks(span.References, parentSpanID, dest.Links())
}

func jTagsToInternalAttributes(tags []model.KeyValue, dest pdata.AttributeMap) {
	for _, tag := range tags {
		switch tag.GetVType() {
		case model.ValueType_STRING:
			dest.UpsertString(tag.Key, tag.GetVStr())
		case model.ValueType_BOOL:
			dest.UpsertBool(tag.Key, tag.GetVBool())
		case model.ValueType_INT64:
			dest.UpsertInt(tag.Key, tag.GetVInt64())
		case model.ValueType_FLOAT64:
			dest.UpsertDouble(tag.Key, tag.GetVFloat64())
		case model.ValueType_BINARY:
			dest.UpsertString(tag.Key, base64.StdEncoding.EncodeToString(tag.GetVBinary()))
		default:
			dest.UpsertString(tag.Key, fmt.Sprintf("<Unknown Jaeger TagType %q>", tag.GetVType()))
		}
	}
}

func setInternalSpanStatus(attrs pdata.AttributeMap, dest pdata.SpanStatus) {
	dest.InitEmpty()

	var codeSet bool
	if codeAttr, ok := attrs.Get(tracetranslator.TagStatusCode); ok {
		code, err := getStatusCodeFromAttr(codeAttr)
		if err == nil {
			codeSet = true
			dest.SetCode(code)
			attrs.Delete(tracetranslator.TagStatusCode)
		}
	} else if errorVal, ok := attrs.Get(tracetranslator.TagError); ok {
		if errorVal.BoolVal() {
			dest.SetCode(pdata.StatusCode(otlptrace.Status_UnknownError))
			codeSet = true
		}
	}

	if codeSet {
		if msgAttr, ok := attrs.Get(tracetranslator.TagStatusMsg); ok {
			dest.SetMessage(msgAttr.StringVal())
			attrs.Delete(tracetranslator.TagStatusMsg)
		}
		attrs.Delete(tracetranslator.TagError)
	} else {
		httpCodeAttr, ok := attrs.Get(tracetranslator.TagHTTPStatusCode)
		if ok {
			code, err := getStatusCodeFromHTTPStatusAttr(httpCodeAttr)
			if err == nil {
				dest.SetCode(code)
			}
		} else {
			dest.SetCode(pdata.StatusCode(otlptrace.Status_Ok))
		}
		if msgAttr, ok := attrs.Get(tracetranslator.TagHTTPStatusMsg); ok {
			dest.SetMessage(msgAttr.StringVal())
		}
	}

}

func getStatusCodeFromAttr(attrVal pdata.AttributeValue) (pdata.StatusCode, error) {
	var codeVal int64
	switch attrVal.Type() {
	case pdata.AttributeValueINT:
		codeVal = attrVal.IntVal()
	case pdata.AttributeValueSTRING:
		i, err := strconv.Atoi(attrVal.StringVal())
		if err != nil {
			return pdata.StatusCode(0), err
		}
		codeVal = int64(i)
	default:
		return pdata.StatusCode(0), fmt.Errorf("invalid status code attribute type: %s", attrVal.Type().String())
	}
	if codeVal > math.MaxInt32 || codeVal < math.MinInt32 {
		return pdata.StatusCode(0), fmt.Errorf("invalid status code value: %d", codeVal)
	}
	return pdata.StatusCode(codeVal), nil
}

func getStatusCodeFromHTTPStatusAttr(attrVal pdata.AttributeValue) (pdata.StatusCode, error) {
	statusCode, err := getStatusCodeFromAttr(attrVal)
	if err != nil {
		return pdata.StatusCode(0), err
	}

	// TODO: Introduce and use new HTTP -> OTLP code translator instead
	return pdata.StatusCode(tracetranslator.OCStatusCodeFromHTTP(int32(statusCode))), nil
}

func jSpanKindToInternal(spanKind string) pdata.SpanKind {
	switch spanKind {
	case "client":
		return pdata.SpanKindCLIENT
	case "server":
		return pdata.SpanKindSERVER
	case "producer":
		return pdata.SpanKindPRODUCER
	case "consumer":
		return pdata.SpanKindCONSUMER
	}
	return pdata.SpanKindUNSPECIFIED
}

func jLogsToSpanEvents(logs []model.Log, dest pdata.SpanEventSlice) {
	if len(logs) == 0 {
		return
	}

	dest.Resize(len(logs))

	for i, log := range logs {
		event := dest.At(i)

		event.SetTimestamp(pdata.TimestampUnixNano(uint64(log.Timestamp.UnixNano())))
		if len(log.Fields) == 0 {
			continue
		}

		attrs := event.Attributes()
		attrs.InitEmptyWithCapacity(len(log.Fields))
		jTagsToInternalAttributes(log.Fields, attrs)
		if name, ok := attrs.Get("message"); ok {
			event.SetName(name.StringVal())
		}
	}
}

// jReferencesToSpanLinks sets internal span links based on jaeger span references skipping excludeParentID
func jReferencesToSpanLinks(refs []model.SpanRef, excludeParentID model.SpanID, dest pdata.SpanLinkSlice) {
	if len(refs) == 0 || len(refs) == 1 && refs[0].SpanID == excludeParentID && refs[0].RefType == model.ChildOf {
		return
	}

	dest.Resize(len(refs))
	i := 0
	for _, ref := range refs {
		link := dest.At(i)
		if ref.SpanID == excludeParentID && ref.RefType == model.ChildOf {
			continue
		}

		link.SetTraceID(pdata.NewTraceID(tracetranslator.UInt64ToByteTraceID(ref.TraceID.High, ref.TraceID.Low)))
		link.SetSpanID(pdata.NewSpanID(tracetranslator.UInt64ToByteSpanID(uint64(ref.SpanID))))
		i++
	}

	// Reduce slice size in case if excludeParentID was skipped
	if i < len(refs) {
		dest.Resize(i)
	}
}
