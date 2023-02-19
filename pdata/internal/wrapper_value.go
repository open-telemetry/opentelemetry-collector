// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type Value struct {
	orig *otlpcommon.AnyValue
}

func GetOrigValue(ms Value) *otlpcommon.AnyValue {
	return ms.orig
}

func NewValue(orig *otlpcommon.AnyValue) Value {
	return Value{orig: orig}
}

func CopyOrigValue(dst, src *otlpcommon.AnyValue) {
	switch ov := src.Value.(type) {
	case *otlpcommon.AnyValue_KvlistValue:
		kv, ok := dst.Value.(*otlpcommon.AnyValue_KvlistValue)
		if !ok {
			kv = &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}
			dst.Value = kv
		}
		if ov.KvlistValue == nil {
			kv.KvlistValue = nil
			return
		}
		// Deep copy to dest.
		CopyOrigMap(&kv.KvlistValue.Values, &ov.KvlistValue.Values)
	case *otlpcommon.AnyValue_ArrayValue:
		av, ok := dst.Value.(*otlpcommon.AnyValue_ArrayValue)
		if !ok {
			av = &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}
			dst.Value = av
		}
		if ov.ArrayValue == nil {
			av.ArrayValue = nil
			return
		}
		// Deep copy to dest.
		CopyOrigSlice(&av.ArrayValue.Values, &ov.ArrayValue.Values)
	case *otlpcommon.AnyValue_BytesValue:
		bv, ok := dst.Value.(*otlpcommon.AnyValue_BytesValue)
		if !ok {
			bv = &otlpcommon.AnyValue_BytesValue{}
			dst.Value = bv
		}
		bv.BytesValue = make([]byte, len(ov.BytesValue))
		copy(bv.BytesValue, ov.BytesValue)
	default:
		// Primitive immutable type, no need for deep copy.
		dst.Value = ov
	}
}

func FillTestValue(dest Value) {
	dest.orig.Value = &otlpcommon.AnyValue_StringValue{StringValue: "v"}
}

func GenerateTestValue() Value {
	var orig otlpcommon.AnyValue
	ms := NewValue(&orig)
	FillTestValue(ms)
	return ms
}
