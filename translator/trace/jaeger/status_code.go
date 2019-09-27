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
	"math"
	"strconv"

	model "github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"

	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func statusCodeFromHTTPTag(tag *jaeger.Tag) *int32 {
	code := statusCodeFromTag(tag)
	if code != nil {
		*code = tracetranslator.OCStatusCodeFromHTTP(*code)
	}
	return code
}

// statusCodeFromTag maps an integer attribute value to a status code (int32).
// The function return nil if the value is not an integer or an integer larger than what
// can fit in an int32
func statusCodeFromTag(tag *jaeger.Tag) *int32 {
	toInt32 := func(i int) *int32 {
		if i <= math.MaxInt32 {
			j := int32(i)
			return &j
		}
		return nil
	}

	switch tag.GetVType() {
	case jaeger.TagType_LONG:
		return toInt32(int(tag.GetVLong()))
	case jaeger.TagType_STRING:
		if i, err := strconv.Atoi(tag.GetVStr()); err == nil {
			return toInt32(i)
		}
	}
	return nil
}

// statusCodeFromTag maps an integer attribute value to a status code (int32).
// The function return nil if the value is not an integer or an integer larger than what
// can fit in an int32
func statusCodeFromProtoTag(tag model.KeyValue) *int32 {
	toInt32 := func(i int) *int32 {
		if i <= math.MaxInt32 {
			j := int32(i)
			return &j
		}
		return nil
	}

	switch tag.GetVType() {
	case model.ValueType_INT64:
		return toInt32(int(tag.GetVInt64()))
	case model.ValueType_STRING:
		if i, err := strconv.Atoi(tag.GetVStr()); err == nil {
			return toInt32(i)
		}
	}
	return nil
}
