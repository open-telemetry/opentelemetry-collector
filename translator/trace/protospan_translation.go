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

package tracetranslator

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// Some of the keys used to represent OTLP constructs as tags or annotations in other formats.
const (
	AnnotationDescriptionKey = "description"

	MessageEventIDKey               = "message.id"
	MessageEventTypeKey             = "message.type"
	MessageEventCompressedSizeKey   = "message.compressed_size"
	MessageEventUncompressedSizeKey = "message.uncompressed_size"

	TagMessage = "message"

	TagSpanKind = "span.kind"

	TagStatusCode          = "status.code"
	TagStatusMsg           = "status.message"
	TagError               = "error"
	TagHTTPStatusCode      = "http.status_code"
	TagHTTPStatusMsg       = "http.status_message"
	TagZipkinCensusCode    = "census.status_code"
	TagZipkinCensusMsg     = "census.status_description"
	TagZipkinOpenCensusMsg = "opencensus.status_description"

	TagW3CTraceState          = "w3c.tracestate"
	TagServiceNameSource      = "otlp.service.name.source"
	TagInstrumentationName    = "otlp.instrumentation.library.name"
	TagInstrumentationVersion = "otlp.instrumentation.library.version"
)

// Constants used for signifying batch-level attribute values where not supplied by OTLP data but required
// by other protocols.
const (
	ResourceNoServiceName = "OTLPResourceNoServiceName"
)

// OpenTracingSpanKind are possible values for TagSpanKind and match the OpenTracing
// conventions: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
// These values are used for representing span kinds that have no
// equivalents in OpenCensus format. They are stored as values of TagSpanKind
type OpenTracingSpanKind string

const (
	OpenTracingSpanKindUnspecified OpenTracingSpanKind = ""
	OpenTracingSpanKindClient      OpenTracingSpanKind = "client"
	OpenTracingSpanKindServer      OpenTracingSpanKind = "server"
	OpenTracingSpanKindConsumer    OpenTracingSpanKind = "consumer"
	OpenTracingSpanKindProducer    OpenTracingSpanKind = "producer"
	OpenTracingSpanKindInternal    OpenTracingSpanKind = "internal"
)

const (
	SpanLinkDataFormat  = "%s|%s|%s|%s|%d"
	SpanEventDataFormat = "%s|%s|%d"
)

type attrValDescript struct {
	regex    *regexp.Regexp
	attrType pdata.AttributeValueType
}

var attrValDescriptions = getAttrValDescripts()
var complexAttrValDescriptions = getComplexAttrValDescripts()

func getAttrValDescripts() []*attrValDescript {
	descriptions := make([]*attrValDescript, 0, 5)
	descriptions = append(descriptions, constructAttrValDescript("^$", pdata.AttributeValueNULL))
	descriptions = append(descriptions, constructAttrValDescript(`^-?\d+$`, pdata.AttributeValueINT))
	descriptions = append(descriptions, constructAttrValDescript(`^-?\d+\.\d+$`, pdata.AttributeValueDOUBLE))
	descriptions = append(descriptions, constructAttrValDescript(`^(true|false)$`, pdata.AttributeValueBOOL))
	descriptions = append(descriptions, constructAttrValDescript(`^\{"\w+":.+\}$`, pdata.AttributeValueMAP))
	descriptions = append(descriptions, constructAttrValDescript(`^\[.*\]$`, pdata.AttributeValueARRAY))
	return descriptions
}

func getComplexAttrValDescripts() []*attrValDescript {
	descriptions := getAttrValDescripts()
	return descriptions[4:]
}

func constructAttrValDescript(regex string, attrType pdata.AttributeValueType) *attrValDescript {
	regexc := regexp.MustCompile(regex)
	return &attrValDescript{
		regex:    regexc,
		attrType: attrType,
	}
}

// AttributeValueToString converts an OTLP AttributeValue object to its equivalent string representation
func AttributeValueToString(attr pdata.AttributeValue, jsonLike bool) string {
	switch attr.Type() {
	case pdata.AttributeValueNULL:
		if jsonLike {
			return "null"
		}
		return ""
	case pdata.AttributeValueSTRING:
		if jsonLike {
			return fmt.Sprintf("%q", attr.StringVal())
		}
		return attr.StringVal()

	case pdata.AttributeValueBOOL:
		return strconv.FormatBool(attr.BoolVal())

	case pdata.AttributeValueDOUBLE:
		return strconv.FormatFloat(attr.DoubleVal(), 'f', -1, 64)

	case pdata.AttributeValueINT:
		return strconv.FormatInt(attr.IntVal(), 10)

	case pdata.AttributeValueMAP:
		jsonStr, _ := json.Marshal(AttributeMapToMap(attr.MapVal()))
		return string(jsonStr)

	case pdata.AttributeValueARRAY:
		jsonStr, _ := json.Marshal(AttributeArrayToSlice(attr.ArrayVal()))
		return string(jsonStr)

	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type())
	}
}

// AttributeMapToMap converts an OTLP AttributeMap to a standard go map
func AttributeMapToMap(attrMap pdata.AttributeMap) map[string]interface{} {
	rawMap := make(map[string]interface{})
	attrMap.ForEach(func(k string, v pdata.AttributeValue) {
		switch v.Type() {
		case pdata.AttributeValueSTRING:
			rawMap[k] = v.StringVal()
		case pdata.AttributeValueINT:
			rawMap[k] = v.IntVal()
		case pdata.AttributeValueDOUBLE:
			rawMap[k] = v.DoubleVal()
		case pdata.AttributeValueBOOL:
			rawMap[k] = v.BoolVal()
		case pdata.AttributeValueNULL:
			rawMap[k] = nil
		case pdata.AttributeValueMAP:
			rawMap[k] = AttributeMapToMap(v.MapVal())
		case pdata.AttributeValueARRAY:
			rawMap[k] = AttributeArrayToSlice(v.ArrayVal())
		}
	})
	return rawMap
}

func AttributeArrayToSlice(attrArray pdata.AnyValueArray) []interface{} {
	rawSlice := make([]interface{}, 0, attrArray.Len())
	for i := 0; i < attrArray.Len(); i++ {
		v := attrArray.At(i)
		switch v.Type() {
		case pdata.AttributeValueSTRING:
			rawSlice = append(rawSlice, v.StringVal())
		case pdata.AttributeValueINT:
			rawSlice = append(rawSlice, v.IntVal())
		case pdata.AttributeValueDOUBLE:
			rawSlice = append(rawSlice, v.DoubleVal())
		case pdata.AttributeValueBOOL:
			rawSlice = append(rawSlice, v.BoolVal())
		case pdata.AttributeValueNULL:
			rawSlice = append(rawSlice, nil)
		default:
			rawSlice = append(rawSlice, "<Invalid array value>")
		}
	}
	return rawSlice
}

// UpsertStringToAttributeMap upserts a string value to the specified key as it's native OTLP type
func UpsertStringToAttributeMap(key string, val string, dest pdata.AttributeMap, omitSimpleTypes bool) {
	switch DetermineValueType(val, omitSimpleTypes) {
	case pdata.AttributeValueINT:
		iVal, _ := strconv.ParseInt(val, 10, 64)
		dest.UpsertInt(key, iVal)
	case pdata.AttributeValueDOUBLE:
		fVal, _ := strconv.ParseFloat(val, 64)
		dest.UpsertDouble(key, fVal)
	case pdata.AttributeValueBOOL:
		bVal, _ := strconv.ParseBool(val)
		dest.UpsertBool(key, bVal)
	case pdata.AttributeValueMAP:
		var attrs map[string]interface{}
		err := json.Unmarshal([]byte(val), &attrs)
		if err == nil {
			attrMap := pdata.NewAttributeValueMap()
			jsonMapToAttributeMap(attrs, attrMap.MapVal())
			dest.Upsert(key, attrMap)
		} else {
			dest.UpsertString(key, "")
		}
	case pdata.AttributeValueARRAY:
		var jArray []interface{}
		err := json.Unmarshal([]byte(val), &jArray)
		if err == nil {
			attrArr := pdata.NewAttributeValueArray()
			jsonArrayToAttributeArray(jArray, attrArr.ArrayVal())
			dest.Upsert(key, attrArr)
		} else {
			dest.UpsertString(key, "")
		}
	default:
		dest.UpsertString(key, val)
	}
}

// DetermineValueType returns the native OTLP attribute type the string translates to.
func DetermineValueType(value string, omitSimpleTypes bool) pdata.AttributeValueType {
	if omitSimpleTypes {
		for _, desc := range complexAttrValDescriptions {
			if desc.regex.MatchString(value) {
				return desc.attrType
			}
		}
	} else {
		for _, desc := range attrValDescriptions {
			if desc.regex.MatchString(value) {
				return desc.attrType
			}
		}
	}
	return pdata.AttributeValueSTRING
}

func jsonMapToAttributeMap(attrs map[string]interface{}, dest pdata.AttributeMap) {
	for key, val := range attrs {
		if val == nil {
			dest.Upsert(key, pdata.NewAttributeValueNull())
			continue
		}
		if s, ok := val.(string); ok {
			dest.UpsertString(key, s)
		} else if d, ok := val.(float64); ok {
			if math.Mod(d, 1.0) == 0.0 {
				dest.UpsertInt(key, int64(d))
			} else {
				dest.UpsertDouble(key, d)
			}
		} else if b, ok := val.(bool); ok {
			dest.UpsertBool(key, b)
		} else if m, ok := val.(map[string]interface{}); ok {
			value := pdata.NewAttributeValueMap()
			jsonMapToAttributeMap(m, value.MapVal())
			dest.Upsert(key, value)
		} else if a, ok := val.([]interface{}); ok {
			value := pdata.NewAttributeValueArray()
			jsonArrayToAttributeArray(a, value.ArrayVal())
			dest.Upsert(key, value)
		}
	}
}

func jsonArrayToAttributeArray(jArray []interface{}, dest pdata.AnyValueArray) {
	for _, val := range jArray {
		if val == nil {
			dest.Append(pdata.NewAttributeValueNull())
			continue
		}
		if s, ok := val.(string); ok {
			dest.Append(pdata.NewAttributeValueString(s))
		} else if d, ok := val.(float64); ok {
			if math.Mod(d, 1.0) == 0.0 {
				dest.Append(pdata.NewAttributeValueInt(int64(d)))
			} else {
				dest.Append(pdata.NewAttributeValueDouble(d))
			}
		} else if b, ok := val.(bool); ok {
			dest.Append(pdata.NewAttributeValueBool(b))
		} else {
			dest.Append(pdata.NewAttributeValueString("<Invalid array value>"))
		}
	}
}

// StatusCodeFromHTTP takes an HTTP status code and return the appropriate OpenTelemetry status code
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/http.md#status
func StatusCodeFromHTTP(httpStatusCode int) pdata.StatusCode {
	if httpStatusCode >= 100 && httpStatusCode < 399 {
		return pdata.StatusCodeUnset
	}
	return pdata.StatusCodeError
}
