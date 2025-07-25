// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	jsoniter "github.com/json-iterator/go"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
)

type Value struct {
	orig  *otlpcommon.AnyValue
	state *State
}

func GetOrigValue(ms Value) *otlpcommon.AnyValue {
	return ms.orig
}

func GetValueState(ms Value) *State {
	return ms.state
}

func NewValue(orig *otlpcommon.AnyValue, state *State) Value {
	return Value{orig: orig, state: state}
}

func CopyOrigValue(dest, src *otlpcommon.AnyValue) {
	switch sv := src.Value.(type) {
	case *otlpcommon.AnyValue_KvlistValue:
		dv, ok := dest.Value.(*otlpcommon.AnyValue_KvlistValue)
		if !ok {
			dv = &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}
			dest.Value = dv
		}
		if sv.KvlistValue == nil {
			dv.KvlistValue = nil
			return
		}
		dv.KvlistValue.Values = CopyOrigMap(dv.KvlistValue.Values, sv.KvlistValue.Values)
	case *otlpcommon.AnyValue_ArrayValue:
		dv, ok := dest.Value.(*otlpcommon.AnyValue_ArrayValue)
		if !ok {
			dv = &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}
			dest.Value = dv
		}
		if sv.ArrayValue == nil {
			dv.ArrayValue = nil
			return
		}
		dv.ArrayValue.Values = CopyOrigSlice(dv.ArrayValue.Values, sv.ArrayValue.Values)
	case *otlpcommon.AnyValue_BytesValue:
		bv, ok := dest.Value.(*otlpcommon.AnyValue_BytesValue)
		if !ok {
			bv = &otlpcommon.AnyValue_BytesValue{}
			dest.Value = bv
		}
		bv.BytesValue = make([]byte, len(sv.BytesValue))
		copy(bv.BytesValue, sv.BytesValue)
	default:
		// Primitive immutable type, no need for deep copy.
		dest.Value = sv
	}
}

func FillTestValue(dest Value) {
	dest.orig.Value = &otlpcommon.AnyValue_StringValue{StringValue: "v"}
}

func GenerateTestValue() Value {
	var orig otlpcommon.AnyValue
	state := StateMutable
	ms := NewValue(&orig, &state)
	FillTestValue(ms)
	return ms
}

// UnmarshalJSONIterValue Unmarshal JSON data and return otlpcommon.AnyValue
func UnmarshalJSONIterValue(val Value, iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "stringValue", "string_value":
			val.orig.Value = &otlpcommon.AnyValue_StringValue{
				StringValue: iter.ReadString(),
			}
		case "boolValue", "bool_value":
			val.orig.Value = &otlpcommon.AnyValue_BoolValue{
				BoolValue: iter.ReadBool(),
			}
		case "intValue", "int_value":
			val.orig.Value = &otlpcommon.AnyValue_IntValue{
				IntValue: json.ReadInt64(iter),
			}
		case "doubleValue", "double_value":
			val.orig.Value = &otlpcommon.AnyValue_DoubleValue{
				DoubleValue: json.ReadFloat64(iter),
			}
		case "bytesValue", "bytes_value":
			val.orig.Value = &otlpcommon.AnyValue_BytesValue{}
			UnmarshalJSONIterByteSlice(NewByteSlice(&val.orig.Value.(*otlpcommon.AnyValue_BytesValue).BytesValue, val.state), iter)
		case "arrayValue", "array_value":
			val.orig.Value = &otlpcommon.AnyValue_ArrayValue{
				ArrayValue: readArray(iter),
			}
		case "kvlistValue", "kvlist_value":
			val.orig.Value = &otlpcommon.AnyValue_KvlistValue{
				KvlistValue: readKvlistValue(iter),
			}
		default:
			iter.Skip()
		}
		return true
	})
}

func readArray(iter *jsoniter.Iterator) *otlpcommon.ArrayValue {
	v := &otlpcommon.ArrayValue{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "values":
			UnmarshalJSONIterSlice(NewSlice(&v.Values, nil), iter)
		default:
			iter.Skip()
		}
		return true
	})
	return v
}

func readKvlistValue(iter *jsoniter.Iterator) *otlpcommon.KeyValueList {
	v := &otlpcommon.KeyValueList{}
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "values":
			UnmarshalJSONIterMap(NewMap(&v.Values, nil), iter)
		default:
			iter.Skip()
		}
		return true
	})
	return v
}
