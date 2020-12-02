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

package zipkin

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

var nonSpanAttributes = getNonSpanAttributes()

func getNonSpanAttributes() map[string]struct{} {
	attrs := make(map[string]struct{})
	for _, key := range conventions.GetResourceSemanticConventionAttributeNames() {
		attrs[key] = struct{}{}
	}
	attrs[tracetranslator.TagServiceNameSource] = struct{}{}
	attrs[tracetranslator.TagInstrumentationName] = struct{}{}
	attrs[tracetranslator.TagInstrumentationVersion] = struct{}{}
	attrs[conventions.OCAttributeProcessStartTime] = struct{}{}
	attrs[conventions.OCAttributeExporterVersion] = struct{}{}
	attrs[conventions.OCAttributeProcessID] = struct{}{}
	attrs[conventions.OCAttributeResourceType] = struct{}{}
	return attrs
}

// Custom Sort on
type byOTLPTypes []*zipkinmodel.SpanModel

func (b byOTLPTypes) Len() int {
	return len(b)
}

func (b byOTLPTypes) Less(i, j int) bool {
	diff := strings.Compare(extractLocalServiceName(b[i]), extractLocalServiceName(b[j]))
	if diff != 0 {
		return diff <= 0
	}
	diff = strings.Compare(extractInstrumentationLibrary(b[i]), extractInstrumentationLibrary(b[j]))
	return diff <= 0
}

func (b byOTLPTypes) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// V2SpansToInternalTraces translates Zipkin v2 spans into internal trace data.
func V2SpansToInternalTraces(zipkinSpans []*zipkinmodel.SpanModel, parseStringTags bool) (pdata.Traces, error) {
	traceData := pdata.NewTraces()
	if len(zipkinSpans) == 0 {
		return traceData, nil
	}

	sort.Sort(byOTLPTypes(zipkinSpans))

	rss := traceData.ResourceSpans()
	prevServiceName := ""
	prevInstrLibName := ""
	rsCount := rss.Len()
	ilsCount := 0
	spanCount := 0
	var curRscSpans pdata.ResourceSpans
	var curILSpans pdata.InstrumentationLibrarySpans
	var curSpans pdata.SpanSlice
	for _, zspan := range zipkinSpans {
		if zspan == nil {
			continue
		}
		tags := copySpanTags(zspan.Tags)
		localServiceName := extractLocalServiceName(zspan)
		if localServiceName != prevServiceName {
			prevServiceName = localServiceName
			rss.Resize(rsCount + 1)
			curRscSpans = rss.At(rsCount)
			rsCount++
			populateResourceFromZipkinSpan(tags, localServiceName, curRscSpans.Resource())
			prevInstrLibName = ""
			ilsCount = 0
		}
		instrLibName := extractInstrumentationLibrary(zspan)
		if instrLibName != prevInstrLibName || ilsCount == 0 {
			prevInstrLibName = instrLibName
			curRscSpans.InstrumentationLibrarySpans().Resize(ilsCount + 1)
			curILSpans = curRscSpans.InstrumentationLibrarySpans().At(ilsCount)
			ilsCount++
			populateILFromZipkinSpan(tags, instrLibName, curILSpans.InstrumentationLibrary())
			spanCount = 0
			curSpans = curILSpans.Spans()
		}
		curSpans.Resize(spanCount + 1)
		err := zSpanToInternal(zspan, tags, curSpans.At(spanCount), parseStringTags)
		if err != nil {
			return traceData, err
		}
		spanCount++
	}

	return traceData, nil
}

func zSpanToInternal(zspan *zipkinmodel.SpanModel, tags map[string]string, dest pdata.Span, parseStringTags bool) error {
	dest.SetTraceID(tracetranslator.UInt64ToTraceID(zspan.TraceID.High, zspan.TraceID.Low))
	dest.SetSpanID(tracetranslator.UInt64ToSpanID(uint64(zspan.ID)))
	if value, ok := tags[tracetranslator.TagW3CTraceState]; ok {
		dest.SetTraceState(pdata.TraceState(value))
		delete(tags, tracetranslator.TagW3CTraceState)
	}
	parentID := zspan.ParentID
	if parentID != nil && *parentID != zspan.ID {
		dest.SetParentSpanID(tracetranslator.UInt64ToSpanID(uint64(*parentID)))
	}

	dest.SetName(zspan.Name)
	startNano := zspan.Timestamp.UnixNano()
	dest.SetStartTime(pdata.TimestampUnixNano(startNano))
	dest.SetEndTime(pdata.TimestampUnixNano(startNano + zspan.Duration.Nanoseconds()))
	dest.SetKind(zipkinKindToSpanKind(zspan.Kind, tags))

	populateSpanStatus(tags, dest.Status())
	if err := zTagsToSpanLinks(tags, dest.Links()); err != nil {
		return err
	}

	attrs := dest.Attributes()
	attrs.InitEmptyWithCapacity(len(tags))
	if err := zTagsToInternalAttrs(zspan, tags, attrs, parseStringTags); err != nil {
		return err
	}

	err := populateSpanEvents(zspan, dest.Events())
	return err
}

func populateSpanStatus(tags map[string]string, status pdata.SpanStatus) {
	if value, ok := tags[tracetranslator.TagStatusCode]; ok {
		status.SetCode(pdata.StatusCode(otlptrace.Status_StatusCode_value[value]))
		delete(tags, tracetranslator.TagStatusCode)
		if value, ok := tags[tracetranslator.TagStatusMsg]; ok {
			status.SetMessage(value)
			delete(tags, tracetranslator.TagStatusMsg)
		}
	}
}

func zipkinKindToSpanKind(kind zipkinmodel.Kind, tags map[string]string) pdata.SpanKind {
	switch kind {
	case zipkinmodel.Client:
		return pdata.SpanKindCLIENT
	case zipkinmodel.Server:
		return pdata.SpanKindSERVER
	case zipkinmodel.Producer:
		return pdata.SpanKindPRODUCER
	case zipkinmodel.Consumer:
		return pdata.SpanKindCONSUMER
	default:
		if value, ok := tags[tracetranslator.TagSpanKind]; ok {
			delete(tags, tracetranslator.TagSpanKind)
			if value == "internal" {
				return pdata.SpanKindINTERNAL
			}
		}
		return pdata.SpanKindUNSPECIFIED
	}
}

func zTagsToSpanLinks(tags map[string]string, dest pdata.SpanLinkSlice) error {
	index := 0
	for i := 0; i < 128; i++ {
		key := fmt.Sprintf("otlp.link.%d", i)
		val, ok := tags[key]
		if !ok {
			return nil
		}
		delete(tags, key)

		parts := strings.Split(val, "|")
		partCnt := len(parts)
		if partCnt < 5 {
			continue
		}
		dest.Resize(index + 1)
		link := dest.At(index)
		index++

		// Convert trace id.
		rawTrace := data.TraceID{}
		errTrace := rawTrace.UnmarshalJSON([]byte(parts[0]))
		if errTrace != nil {
			return errTrace
		}
		link.SetTraceID(pdata.TraceID(rawTrace))

		// Convert span id.
		rawSpan := data.SpanID{}
		errSpan := rawSpan.UnmarshalJSON([]byte(parts[1]))
		if errSpan != nil {
			return errSpan
		}
		link.SetSpanID(pdata.SpanID(rawSpan))

		link.SetTraceState(pdata.TraceState(parts[2]))

		var jsonStr string
		if partCnt == 5 {
			jsonStr = parts[3]
		} else {
			jsonParts := parts[3 : partCnt-1]
			jsonStr = strings.Join(jsonParts, "|")
		}
		var attrs map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &attrs); err != nil {
			return err
		}
		if err := jsonMapToAttributeMap(attrs, link.Attributes()); err != nil {
			return err
		}

		dropped, errDropped := strconv.ParseUint(parts[partCnt-1], 10, 32)
		if errDropped != nil {
			return errDropped
		}
		link.SetDroppedAttributesCount(uint32(dropped))
	}
	return nil
}

func populateSpanEvents(zspan *zipkinmodel.SpanModel, events pdata.SpanEventSlice) error {
	events.Resize(len(zspan.Annotations))
	for ix, anno := range zspan.Annotations {
		event := events.At(ix)
		startNano := anno.Timestamp.UnixNano()
		event.SetTimestamp(pdata.TimestampUnixNano(startNano))

		parts := strings.Split(anno.Value, "|")
		partCnt := len(parts)
		event.SetName(parts[0])
		if partCnt < 3 {
			continue
		}

		var jsonStr string
		if partCnt == 3 {
			jsonStr = parts[1]
		} else {
			jsonParts := parts[1 : partCnt-1]
			jsonStr = strings.Join(jsonParts, "|")
		}
		var attrs map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &attrs); err != nil {
			return err
		}
		if err := jsonMapToAttributeMap(attrs, event.Attributes()); err != nil {
			return err
		}

		dropped, errDropped := strconv.ParseUint(parts[partCnt-1], 10, 32)
		if errDropped != nil {
			return errDropped
		}
		event.SetDroppedAttributesCount(uint32(dropped))
	}
	return nil
}

func jsonMapToAttributeMap(attrs map[string]interface{}, dest pdata.AttributeMap) error {
	for key, val := range attrs {
		if s, ok := val.(string); ok {
			dest.InsertString(key, s)
		} else if d, ok := val.(float64); ok {
			if math.Mod(d, 1.0) == 0.0 {
				dest.InsertInt(key, int64(d))
			} else {
				dest.InsertDouble(key, d)
			}
		} else if b, ok := val.(bool); ok {
			dest.InsertBool(key, b)
		}
	}
	return nil
}

func zTagsToInternalAttrs(zspan *zipkinmodel.SpanModel, tags map[string]string, dest pdata.AttributeMap, parseStringTags bool) error {
	parseErr := tagsToAttributeMap(tags, dest, parseStringTags)
	if zspan.LocalEndpoint != nil {
		if zspan.LocalEndpoint.IPv4 != nil {
			dest.InsertString(conventions.AttributeNetHostIP, zspan.LocalEndpoint.IPv4.String())
		}
		if zspan.LocalEndpoint.IPv6 != nil {
			dest.InsertString(conventions.AttributeNetHostIP, zspan.LocalEndpoint.IPv6.String())
		}
		if zspan.LocalEndpoint.Port > 0 {
			dest.UpsertInt(conventions.AttributeNetHostPort, int64(zspan.LocalEndpoint.Port))
		}
	}
	if zspan.RemoteEndpoint != nil {
		if zspan.RemoteEndpoint.ServiceName != "" {
			dest.InsertString(conventions.AttributePeerService, zspan.RemoteEndpoint.ServiceName)
		}
		if zspan.RemoteEndpoint.IPv4 != nil {
			dest.InsertString(conventions.AttributeNetPeerIP, zspan.RemoteEndpoint.IPv4.String())
		}
		if zspan.RemoteEndpoint.IPv6 != nil {
			dest.InsertString(conventions.AttributeNetPeerIP, zspan.RemoteEndpoint.IPv6.String())
		}
		if zspan.RemoteEndpoint.Port > 0 {
			dest.UpsertInt(conventions.AttributeNetPeerPort, int64(zspan.RemoteEndpoint.Port))
		}
	}
	return parseErr
}

func tagsToAttributeMap(tags map[string]string, dest pdata.AttributeMap, parseStringTags bool) error {
	var parseErr error
	for key, val := range tags {
		if _, ok := nonSpanAttributes[key]; ok {
			continue
		}

		if parseStringTags {
			switch tracetranslator.DetermineValueType(val, false) {
			case pdata.AttributeValueINT:
				iValue, _ := strconv.ParseInt(val, 10, 64)
				dest.UpsertInt(key, iValue)
			case pdata.AttributeValueDOUBLE:
				fValue, _ := strconv.ParseFloat(val, 64)
				dest.UpsertDouble(key, fValue)
			case pdata.AttributeValueBOOL:
				bValue, _ := strconv.ParseBool(val)
				dest.UpsertBool(key, bValue)
			default:
				dest.UpsertString(key, val)
			}
		} else {
			dest.UpsertString(key, val)
		}
	}
	return parseErr
}

func populateResourceFromZipkinSpan(tags map[string]string, localServiceName string, resource pdata.Resource) {
	if localServiceName == tracetranslator.ResourceNoServiceName {
		return
	}

	if len(tags) == 0 {
		resource.Attributes().InsertString(conventions.AttributeServiceName, localServiceName)
		return
	}

	snSource := tags[tracetranslator.TagServiceNameSource]
	if snSource == "" {
		resource.Attributes().InsertString(conventions.AttributeServiceName, localServiceName)
	} else {
		resource.Attributes().InsertString(snSource, localServiceName)
	}
	delete(tags, tracetranslator.TagServiceNameSource)

	for key := range getNonSpanAttributes() {
		if key == tracetranslator.TagInstrumentationName || key == tracetranslator.TagInstrumentationVersion {
			continue
		}
		if value, ok := tags[key]; ok {
			resource.Attributes().UpsertString(key, value)
			delete(tags, key)
		}
	}
}

func populateILFromZipkinSpan(tags map[string]string, instrLibName string, library pdata.InstrumentationLibrary) {
	if instrLibName == "" {
		return
	}
	if value, ok := tags[tracetranslator.TagInstrumentationName]; ok {
		library.SetName(value)
		delete(tags, tracetranslator.TagInstrumentationName)
	}
	if value, ok := tags[tracetranslator.TagInstrumentationVersion]; ok {
		library.SetVersion(value)
		delete(tags, tracetranslator.TagInstrumentationVersion)
	}
}

func copySpanTags(tags map[string]string) map[string]string {
	dest := make(map[string]string, len(tags))
	for key, val := range tags {
		dest[key] = val
	}
	return dest
}

func extractLocalServiceName(zspan *zipkinmodel.SpanModel) string {
	if zspan == nil || zspan.LocalEndpoint == nil || zspan.LocalEndpoint.ServiceName == "" {
		return tracetranslator.ResourceNoServiceName
	}
	return zspan.LocalEndpoint.ServiceName
}

func extractInstrumentationLibrary(zspan *zipkinmodel.SpanModel) string {
	if zspan == nil || len(zspan.Tags) == 0 {
		return ""
	}
	return zspan.Tags[tracetranslator.TagInstrumentationName]
}
