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
	"fmt"
	"math"
	"strconv"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

type status struct {
	codePtr *int32
	message string
}

// statusMapper contains codes translated from different sources to OC status codes
type statusMapper struct {
	// oc status code extracted from "status.code" tags
	fromStatus status
	// oc status code extracted from "census.status_code" tags
	fromCensus status
	// oc status code extracted from "http.status_code" tags
	fromHTTP status
	// oc status code extracted from "error" tags
	fromErrorTag status
	// oc status code 'unknown' when the "error" tag exists but is invalid
	fromErrorTagUnknown status
}

// ocStatus returns an OC status from the best possible extraction source.
// It'll first try to return status extracted from "census.status_code" to account for zipkin
// then fallback on code extracted from "status.code" tags
// and finally fallback on code extracted and translated from "http.status_code"
// ocStatus must be called after all tags/attributes are processed with the `fromAttribute` method.
func (m *statusMapper) ocStatus() *tracepb.Status {
	var s status
	switch {
	case m.fromCensus.codePtr != nil:
		s = m.fromCensus
	case m.fromStatus.codePtr != nil:
		s = m.fromStatus
	case m.fromErrorTag.codePtr != nil:
		s = m.fromErrorTag
		if m.fromCensus.message != "" {
			s.message = m.fromCensus.message
		} else if m.fromStatus.message != "" {
			s.message = m.fromStatus.message
		}
	case m.fromHTTP.codePtr != nil:
		s = m.fromHTTP
	default:
		s = m.fromErrorTagUnknown
	}

	if s.codePtr != nil {
		code := int32(0)
		if s.codePtr != nil {
			code = *s.codePtr
		}
		return &tracepb.Status{
			Code:    code,
			Message: s.message,
		}
	}
	return nil
}

func (m *statusMapper) fromAttribute(key string, attrib *tracepb.AttributeValue) bool {
	switch key {
	case tracetranslator.TagZipkinCensusCode:
		code, err := attribToStatusCode(attrib)
		if err == nil {
			m.fromCensus.codePtr = &code
		}
		return true

	case tracetranslator.TagZipkinCensusMsg, tracetranslator.TagZipkinOpenCensusMsg:
		m.fromCensus.message = attrib.GetStringValue().GetValue()
		return true

	case tracetranslator.TagStatusCode:
		code, err := attribToStatusCode(attrib)
		if err == nil {
			m.fromStatus.codePtr = &code
		}
		return true

	case tracetranslator.TagStatusMsg:
		m.fromStatus.message = attrib.GetStringValue().GetValue()
		return true

	case tracetranslator.TagHTTPStatusCode:
		httpCode, err := attribToStatusCode(attrib)
		if err == nil {
			code := tracetranslator.OCStatusCodeFromHTTP(httpCode)
			m.fromHTTP.codePtr = &code
		}

	case tracetranslator.TagHTTPStatusMsg:
		m.fromHTTP.message = attrib.GetStringValue().GetValue()

	case tracetranslator.TagError:
		code, ok := extractStatusFromError(attrib)
		if ok {
			m.fromErrorTag.codePtr = code
			return true
		}
		m.fromErrorTagUnknown.codePtr = code
	}
	return false
}

// attribToStatusCode maps an integer or string attribute value to a status code.
// The function return nil if the value is of another type or cannot be converted to an int32 value.
func attribToStatusCode(attr *tracepb.AttributeValue) (int32, error) {
	if attr == nil {
		return 0, fmt.Errorf("nil attribute")
	}

	switch val := attr.Value.(type) {
	case *tracepb.AttributeValue_IntValue:
		return toInt32(int(val.IntValue))
	case *tracepb.AttributeValue_StringValue:
		i, err := strconv.Atoi(val.StringValue.GetValue())
		if err != nil {
			return 0, err
		}
		return toInt32(i)
	}
	return 0, fmt.Errorf("invalid attribute type")
}

func toInt32(i int) (int32, error) {
	if i <= math.MaxInt32 && i >= math.MinInt32 {
		return int32(i), nil
	}
	return 0, fmt.Errorf("outside of the int32 range")
}

func extractStatusFromError(attrib *tracepb.AttributeValue) (*int32, bool) {
	// The status is stored with the "error" key
	// See https://github.com/census-instrumentation/opencensus-go/blob/1eb9a13c7dd02141e065a665f6bf5c99a090a16a/exporter/zipkin/zipkin.go#L160-L165
	var unknown int32 = 2

	switch val := attrib.Value.(type) {
	case *tracepb.AttributeValue_StringValue:
		canonicalCodeStr := val.StringValue.GetValue()
		if canonicalCodeStr == "" {
			return nil, true
		}
		code, set := canonicalCodesMap[canonicalCodeStr]
		if set {
			return &code, true
		}
	default:
		break
	}

	return &unknown, false
}

var canonicalCodesMap = map[string]int32{
	// https://github.com/googleapis/googleapis/blob/bee79fbe03254a35db125dc6d2f1e9b752b390fe/google/rpc/code.proto#L33-L186
	"OK":                  0,
	"CANCELLED":           1,
	"UNKNOWN":             2,
	"INVALID_ARGUMENT":    3,
	"DEADLINE_EXCEEDED":   4,
	"NOT_FOUND":           5,
	"ALREADY_EXISTS":      6,
	"PERMISSION_DENIED":   7,
	"RESOURCE_EXHAUSTED":  8,
	"FAILED_PRECONDITION": 9,
	"ABORTED":             10,
	"OUT_OF_RANGE":        11,
	"UNIMPLEMENTED":       12,
	"INTERNAL":            13,
	"UNAVAILABLE":         14,
	"DATA_LOSS":           15,
	"UNAUTHENTICATED":     16,
}
