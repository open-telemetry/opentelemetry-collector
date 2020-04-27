// Copyright 2020, OpenTelemetry Authors
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
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

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
		attrs.UpsertString(conventions.AttributeServiceName, serviceName)
	}

	for _, tag := range tags {
		switch tag.GetVType() {
		case model.ValueType_STRING:
			attrs.UpsertString(tag.Key, tag.GetVStr())
		case model.ValueType_BOOL:
			attrs.UpsertBool(tag.Key, tag.GetVBool())
		case model.ValueType_INT64:
			attrs.UpsertInt(tag.Key, tag.GetVInt64())
		case model.ValueType_FLOAT64:
			attrs.UpsertDouble(tag.Key, tag.GetVFloat64())
		case model.ValueType_BINARY:
			attrs.UpsertString(tag.Key, base64.StdEncoding.EncodeToString(tag.GetVBinary()))
		default:
			attrs.UpsertString(tag.Key, fmt.Sprintf("<Unknown Jaeger TagType %q>", tag.GetVType()))
		}
	}

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

	attrs := jTagsToInternalAttributes(span.Tags)
	setInternalSpanStatus(attrs, dest.Status())
	if spanKindAttr, ok := attrs[tracetranslator.TagSpanKind]; ok {
		dest.SetKind(jSpanKindToInternal(spanKindAttr.StringVal()))
	}
	dest.Attributes().InitFromMap(attrs)

	jLogsToSpanEvents(span.Logs, dest.Events())
	jReferencesToSpanLinks(span.References, parentSpanID, dest.Links())
}

func jTagsToInternalAttributes(tags []model.KeyValue) map[string]pdata.AttributeValue {
	attrs := make(map[string]pdata.AttributeValue)

	for _, tag := range tags {
		switch tag.GetVType() {
		case model.ValueType_STRING:
			attrs[tag.Key] = pdata.NewAttributeValueString(tag.GetVStr())
		case model.ValueType_BOOL:
			attrs[tag.Key] = pdata.NewAttributeValueBool(tag.GetVBool())
		case model.ValueType_INT64:
			attrs[tag.Key] = pdata.NewAttributeValueInt(tag.GetVInt64())
		case model.ValueType_FLOAT64:
			attrs[tag.Key] = pdata.NewAttributeValueDouble(tag.GetVFloat64())
		case model.ValueType_BINARY:
			attrs[tag.Key] = pdata.NewAttributeValueString(base64.StdEncoding.EncodeToString(tag.GetVBinary()))
		default:
			attrs[tag.Key] = pdata.NewAttributeValueString(fmt.Sprintf("<Unknown Jaeger TagType %q>", tag.GetVType()))
		}
	}

	return attrs
}

func setInternalSpanStatus(attrs map[string]pdata.AttributeValue, dest pdata.SpanStatus) {
	dest.InitEmpty()

	var codeSet bool
	if codeAttr, ok := attrs[tracetranslator.TagStatusCode]; ok {
		code, err := getStatusCodeFromAttr(codeAttr)
		if err == nil {
			codeSet = true
			dest.SetCode(code)
			delete(attrs, tracetranslator.TagStatusCode)
		}
	} else if errorVal, ok := attrs[tracetranslator.TagError]; ok {
		if errorVal.BoolVal() {
			dest.SetCode(pdata.StatusCode(otlptrace.Status_UnknownError))
			codeSet = true
		}
	}

	if codeSet {
		if msgAttr, ok := attrs[tracetranslator.TagStatusMsg]; ok {
			dest.SetMessage(msgAttr.StringVal())
			delete(attrs, tracetranslator.TagStatusMsg)
		}
		delete(attrs, tracetranslator.TagError)
	} else {
		httpCodeAttr, ok := attrs[tracetranslator.TagHTTPStatusCode]
		if ok {
			code, err := getStatusCodeFromHTTPStatusAttr(httpCodeAttr)
			if err == nil {
				dest.SetCode(code)
			}
		} else {
			dest.SetCode(pdata.StatusCode(otlptrace.Status_Ok))
		}
		if msgAttr, ok := attrs[tracetranslator.TagHTTPStatusMsg]; ok {
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

		attrs := jTagsToInternalAttributes(log.Fields)
		if name, ok := attrs["message"]; ok {
			event.SetName(name.StringVal())
		}
		if len(attrs) > 0 {
			event.Attributes().InitFromMap(attrs)
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
