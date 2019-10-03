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

package zipkin

import (
	"math"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
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
	default:
		s = m.fromHTTP
	}

	if s.codePtr != nil || s.message != "" {
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
		m.fromCensus.codePtr = attribToStatusCode(attrib)
		return true

	case tracetranslator.TagZipkinCensusMsg:
		m.fromCensus.message = attrib.GetStringValue().GetValue()
		return true

	case tracetranslator.TagStatusCode:
		m.fromStatus.codePtr = attribToStatusCode(attrib)
		return true

	case tracetranslator.TagStatusMsg:
		m.fromStatus.message = attrib.GetStringValue().GetValue()
		return true

	case tracetranslator.TagHTTPStatusCode:
		codePtr := attribToStatusCode(attrib)
		if codePtr != nil {
			*codePtr = tracetranslator.OCStatusCodeFromHTTP(*codePtr)
			m.fromHTTP.codePtr = codePtr
		}

	case tracetranslator.TagHTTPStatusMsg:
		m.fromHTTP.message = attrib.GetStringValue().GetValue()
	}
	return false
}

// attribToStatusCode maps an integer attribute value to a status code (int32).
// The function return nil if the value is not an integer or an integer larger than what
// can fit in an int32
func attribToStatusCode(v *tracepb.AttributeValue) *int32 {
	i := v.GetIntValue()
	if i <= math.MaxInt32 {
		j := int32(i)
		return &j
	}
	return nil
}
