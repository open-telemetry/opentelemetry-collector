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

package zipkinv2

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"

	"go.opentelemetry.io/collector/internal/idutils"
	"go.opentelemetry.io/collector/internal/occonventions"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
	"go.opentelemetry.io/collector/translator/trace/internal/zipkin"
)

// ToTranslator converts from Zipkin data model to pdata.
type ToTranslator struct {
	// ParseStringTags should be set to true if tags should be converted to numbers when possible.
	ParseStringTags bool
}

// ToTraces translates Zipkin v2 spans into pdata.Traces.
func (t ToTranslator) ToTraces(zipkinSpans []*zipkinmodel.SpanModel) (pdata.Traces, error) {
	traceData := pdata.NewTraces()
	if len(zipkinSpans) == 0 {
		return traceData, nil
	}

	sort.Sort(byOTLPTypes(zipkinSpans))

	rss := traceData.ResourceSpans()
	prevServiceName := ""
	prevInstrLibName := ""
	ilsIsNew := true
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
			curRscSpans = rss.AppendEmpty()
			populateResourceFromZipkinSpan(tags, localServiceName, curRscSpans.Resource())
			prevInstrLibName = ""
			ilsIsNew = true
		}
		instrLibName := extractInstrumentationLibrary(zspan)
		if instrLibName != prevInstrLibName || ilsIsNew {
			prevInstrLibName = instrLibName
			curILSpans = curRscSpans.InstrumentationLibrarySpans().AppendEmpty()
			ilsIsNew = false
			populateILFromZipkinSpan(tags, instrLibName, curILSpans.InstrumentationLibrary())
			curSpans = curILSpans.Spans()
		}
		err := zSpanToInternal(zspan, tags, curSpans.AppendEmpty(), t.ParseStringTags)
		if err != nil {
			return traceData, err
		}
	}

	return traceData, nil
}

var nonSpanAttributes = func() map[string]struct{} {
	attrs := make(map[string]struct{})
	for _, key := range conventions.GetResourceSemanticConventionAttributeNames() {
		attrs[key] = struct{}{}
	}
	attrs[zipkin.TagServiceNameSource] = struct{}{}
	attrs[conventions.InstrumentationLibraryName] = struct{}{}
	attrs[conventions.InstrumentationLibraryVersion] = struct{}{}
	attrs[occonventions.AttributeProcessStartTime] = struct{}{}
	attrs[occonventions.AttributeExporterVersion] = struct{}{}
	attrs[conventions.AttributeProcessID] = struct{}{}
	attrs[occonventions.AttributeResourceType] = struct{}{}
	return attrs
}()

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

func zSpanToInternal(zspan *zipkinmodel.SpanModel, tags map[string]string, dest pdata.Span, parseStringTags bool) error {
	dest.SetTraceID(idutils.UInt64ToTraceID(zspan.TraceID.High, zspan.TraceID.Low))
	dest.SetSpanID(idutils.UInt64ToSpanID(uint64(zspan.ID)))
	if value, ok := tags[tracetranslator.TagW3CTraceState]; ok {
		dest.SetTraceState(pdata.TraceState(value))
		delete(tags, tracetranslator.TagW3CTraceState)
	}
	parentID := zspan.ParentID
	if parentID != nil && *parentID != zspan.ID {
		dest.SetParentSpanID(idutils.UInt64ToSpanID(uint64(*parentID)))
	}

	dest.SetName(zspan.Name)
	dest.SetKind(zipkinKindToSpanKind(zspan.Kind, tags))

	populateSpanStatus(tags, dest.Status())
	if err := zTagsToSpanLinks(tags, dest.Links()); err != nil {
		return err
	}

	attrs := dest.Attributes()
	attrs.Clear()
	attrs.EnsureCapacity(len(tags))
	if err := zTagsToInternalAttrs(zspan, tags, attrs, parseStringTags); err != nil {
		return err
	}

	setTimestampsV2(zspan, dest, attrs)

	err := populateSpanEvents(zspan, dest.Events())
	return err
}

func populateSpanStatus(tags map[string]string, status pdata.SpanStatus) {
	if value, ok := tags[tracetranslator.TagStatusCode]; ok {
		status.SetCode(pdata.StatusCode(statusCodeValue[value]))
		delete(tags, tracetranslator.TagStatusCode)
		if value, ok := tags[tracetranslator.TagStatusMsg]; ok {
			status.SetMessage(value)
			delete(tags, tracetranslator.TagStatusMsg)
		}
	}

	if val, ok := tags[tracetranslator.TagError]; ok {
		if val == "true" {
			status.SetCode(pdata.StatusCodeError)
			delete(tags, tracetranslator.TagError)
		}
	}
}

func zipkinKindToSpanKind(kind zipkinmodel.Kind, tags map[string]string) pdata.SpanKind {
	switch kind {
	case zipkinmodel.Client:
		return pdata.SpanKindClient
	case zipkinmodel.Server:
		return pdata.SpanKindServer
	case zipkinmodel.Producer:
		return pdata.SpanKindProducer
	case zipkinmodel.Consumer:
		return pdata.SpanKindConsumer
	default:
		if value, ok := tags[tracetranslator.TagSpanKind]; ok {
			delete(tags, tracetranslator.TagSpanKind)
			if value == "internal" {
				return pdata.SpanKindInternal
			}
		}
		return pdata.SpanKindUnspecified
	}
}

func zTagsToSpanLinks(tags map[string]string, dest pdata.SpanLinkSlice) error {
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
		link := dest.AppendEmpty()

		// Convert trace id.
		rawTrace := [16]byte{}
		errTrace := unmarshalJSON(rawTrace[:], []byte(parts[0]))
		if errTrace != nil {
			return errTrace
		}
		link.SetTraceID(pdata.NewTraceID(rawTrace))

		// Convert span id.
		rawSpan := [8]byte{}
		errSpan := unmarshalJSON(rawSpan[:], []byte(parts[1]))
		if errSpan != nil {
			return errSpan
		}
		link.SetSpanID(pdata.NewSpanID(rawSpan))

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
	events.EnsureCapacity(len(zspan.Annotations))
	for _, anno := range zspan.Annotations {
		event := events.AppendEmpty()
		event.SetTimestamp(pdata.TimestampFromTime(anno.Timestamp))

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
			switch zipkin.DetermineValueType(val) {
			case pdata.AttributeValueTypeInt:
				iValue, _ := strconv.ParseInt(val, 10, 64)
				dest.UpsertInt(key, iValue)
			case pdata.AttributeValueTypeDouble:
				fValue, _ := strconv.ParseFloat(val, 64)
				dest.UpsertDouble(key, fValue)
			case pdata.AttributeValueTypeBool:
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

	snSource := tags[zipkin.TagServiceNameSource]
	if snSource == "" {
		resource.Attributes().InsertString(conventions.AttributeServiceName, localServiceName)
	} else {
		resource.Attributes().InsertString(snSource, localServiceName)
	}
	delete(tags, zipkin.TagServiceNameSource)

	for key := range nonSpanAttributes {
		if key == conventions.InstrumentationLibraryName || key == conventions.InstrumentationLibraryVersion {
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
	if value, ok := tags[conventions.InstrumentationLibraryName]; ok {
		library.SetName(value)
		delete(tags, conventions.InstrumentationLibraryName)
	}
	if value, ok := tags[conventions.InstrumentationLibraryVersion]; ok {
		library.SetVersion(value)
		delete(tags, conventions.InstrumentationLibraryVersion)
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
	return zspan.Tags[conventions.InstrumentationLibraryName]
}

func setTimestampsV2(zspan *zipkinmodel.SpanModel, dest pdata.Span, destAttrs pdata.AttributeMap) {
	// zipkin allows timestamp to be unset, but otel span expects startTimestamp to have a value.
	// unset gets converted to zero on the zspan object during json deserialization because
	// time.Time (the type of Timestamp field) cannot be nil.  If timestamp is zero, the
	// conversion from this internal format back to zipkin format in zipkin exporter fails.
	// Instead, set to *unix* time zero, and convert back in traces_to_zipkinv2.go
	if zspan.Timestamp.IsZero() {
		unixTimeZero := pdata.TimestampFromTime(time.Unix(0, 0))
		zeroPlusDuration := pdata.TimestampFromTime(time.Unix(0, 0).Add(zspan.Duration))
		dest.SetStartTimestamp(unixTimeZero)
		dest.SetEndTimestamp(zeroPlusDuration)

		destAttrs.InsertBool(zipkin.StartTimeAbsent, true)
	} else {
		dest.SetStartTimestamp(pdata.TimestampFromTime(zspan.Timestamp))
		dest.SetEndTimestamp(pdata.TimestampFromTime(zspan.Timestamp.Add(zspan.Duration)))
	}
}

// unmarshalJSON inflates trace id from hex string, possibly enclosed in quotes.
// TODO: Find a way to avoid this duplicate code. Consider to expose this in model/pdata.
func unmarshalJSON(dst []byte, src []byte) error {
	if l := len(src); l >= 2 && src[0] == '"' && src[l-1] == '"' {
		src = src[1 : l-1]
	}
	nLen := len(src)
	if nLen == 0 {
		return nil
	}

	if len(dst) != hex.DecodedLen(nLen) {
		return errors.New("invalid length for ID")
	}

	_, err := hex.Decode(dst, src)
	if err != nil {
		return fmt.Errorf("cannot unmarshal ID from string '%s': %w", string(src), err)
	}
	return nil
}

// TODO: Find a way to avoid this duplicate code. Consider to expose this in model/pdata.
var statusCodeValue = map[string]int32{
	"STATUS_CODE_UNSET": 0,
	"STATUS_CODE_OK":    1,
	"STATUS_CODE_ERROR": 2,
}
