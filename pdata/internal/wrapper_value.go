// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"fmt"
	"math"
	"sync"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/proto"
)

type Value struct {
	orig  *otlpcommon.AnyValue
	state *State
}

var (
	protoPoolAnyValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue{}
		},
	}
	protoPoolAnyValueStringValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_StringValue{}
		},
	}
	protoPoolAnyValueBoolValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_BoolValue{}
		},
	}
	protoPoolAnyValueIntValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_IntValue{}
		},
	}
	protoPoolAnyValueDoubleValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_DoubleValue{}
		},
	}
	protoPoolAnyValueBytesValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_BytesValue{}
		},
	}
	protoPoolAnyValueArrayValue = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_ArrayValue{}
		},
	}
	protoPoolAnyValueKeyValueList = sync.Pool{
		New: func() any {
			return &otlpcommon.AnyValue_KvlistValue{}
		},
	}
)

func NewOrigAnyValue() *otlpcommon.AnyValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue{}
	}
	return protoPoolAnyValue.Get().(*otlpcommon.AnyValue)
}

func NewOrigAnyValueStringValue() *otlpcommon.AnyValue_StringValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_StringValue{}
	}
	return protoPoolAnyValueStringValue.Get().(*otlpcommon.AnyValue_StringValue)
}

func NewOrigAnyValueIntValue() *otlpcommon.AnyValue_IntValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_IntValue{}
	}
	return protoPoolAnyValueIntValue.Get().(*otlpcommon.AnyValue_IntValue)
}

func NewOrigAnyValueBoolValue() *otlpcommon.AnyValue_BoolValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_BoolValue{}
	}
	return protoPoolAnyValueBoolValue.Get().(*otlpcommon.AnyValue_BoolValue)
}

func NewOrigAnyValueDoubleValue() *otlpcommon.AnyValue_DoubleValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_DoubleValue{}
	}
	return protoPoolAnyValueDoubleValue.Get().(*otlpcommon.AnyValue_DoubleValue)
}

func NewOrigAnyValueBytesValue() *otlpcommon.AnyValue_BytesValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_BytesValue{}
	}
	return protoPoolAnyValueBytesValue.Get().(*otlpcommon.AnyValue_BytesValue)
}

func NewOrigAnyValueArrayValue() *otlpcommon.AnyValue_ArrayValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_ArrayValue{}
	}
	return protoPoolAnyValueArrayValue.Get().(*otlpcommon.AnyValue_ArrayValue)
}

func NewOrigAnyValueKvlistValue() *otlpcommon.AnyValue_KvlistValue {
	if !UseProtoPooling.IsEnabled() {
		return &otlpcommon.AnyValue_KvlistValue{}
	}
	return protoPoolAnyValueKeyValueList.Get().(*otlpcommon.AnyValue_KvlistValue)
}

func DeleteOrigAnyValue(orig *otlpcommon.AnyValue, nullable bool) {
	if orig == nil {
		return
	}

	if !UseProtoPooling.IsEnabled() {
		orig.Reset()
		return
	}

	switch v := orig.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		v.StringValue = ""
		protoPoolAnyValueStringValue.Put(v)
	case *otlpcommon.AnyValue_BoolValue:
		v.BoolValue = false
		protoPoolAnyValueBoolValue.Put(v)
	case *otlpcommon.AnyValue_IntValue:
		v.IntValue = 0
		protoPoolAnyValueIntValue.Put(v)
	case *otlpcommon.AnyValue_DoubleValue:
		v.DoubleValue = 0
		protoPoolAnyValueDoubleValue.Put(v)
	case *otlpcommon.AnyValue_BytesValue:
		v.BytesValue = nil
		protoPoolAnyValueBytesValue.Put(v)
	case *otlpcommon.AnyValue_ArrayValue:
		DeleteOrigArrayValue(v.ArrayValue, true)
		protoPoolAnyValueArrayValue.Put(v)
	case *otlpcommon.AnyValue_KvlistValue:
		DeleteOrigKeyValueList(v.KvlistValue, true)
		protoPoolAnyValueKeyValueList.Put(v)
	}

	orig.Reset()
	if nullable {
		protoPoolAnyValue.Put(orig)
	}
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

func CopyOrigAnyValue(dest, src *otlpcommon.AnyValue) {
	if src == dest {
		return
	}
	switch sv := src.Value.(type) {
	case nil:
		dest.Value = nil
	case *otlpcommon.AnyValue_StringValue:
		dv := NewOrigAnyValueStringValue()
		dv.StringValue = sv.StringValue
		dest.Value = dv
	case *otlpcommon.AnyValue_BoolValue:
		dv := NewOrigAnyValueBoolValue()
		dv.BoolValue = sv.BoolValue
		dest.Value = dv
	case *otlpcommon.AnyValue_IntValue:
		dv := NewOrigAnyValueIntValue()
		dv.IntValue = sv.IntValue
		dest.Value = dv
	case *otlpcommon.AnyValue_DoubleValue:
		dv := NewOrigAnyValueDoubleValue()
		dv.DoubleValue = sv.DoubleValue
		dest.Value = dv
	case *otlpcommon.AnyValue_BytesValue:
		dv := NewOrigAnyValueBytesValue()
		dest.Value = dv
		dv.BytesValue = make([]byte, len(sv.BytesValue))
		copy(dv.BytesValue, sv.BytesValue)
	case *otlpcommon.AnyValue_KvlistValue:
		dv := NewOrigAnyValueKvlistValue()
		dest.Value = dv
		if sv.KvlistValue == nil {
			return
		}
		dv.KvlistValue = NewOrigKeyValueList()
		dv.KvlistValue.Values = CopyOrigKeyValueSlice(dv.KvlistValue.Values, sv.KvlistValue.Values)
	case *otlpcommon.AnyValue_ArrayValue:
		dv := NewOrigAnyValueArrayValue()
		dest.Value = dv
		if sv.ArrayValue == nil {
			return
		}
		dv.ArrayValue = NewOrigArrayValue()
		dv.ArrayValue.Values = CopyOrigAnyValueSlice(dv.ArrayValue.Values, sv.ArrayValue.Values)
	}
}

// MarshalJSONOrigAnyValue marshals all properties from the current struct to the destination stream.
func MarshalJSONOrigAnyValue(orig *otlpcommon.AnyValue, dest *json.Stream) {
	dest.WriteObjectStart()
	switch v := orig.Value.(type) {
	case nil:
		// Do nothing, return an empty object.
	case *otlpcommon.AnyValue_StringValue:
		dest.WriteObjectField("stringValue")
		dest.WriteString(v.StringValue)
	case *otlpcommon.AnyValue_BoolValue:
		dest.WriteObjectField("boolValue")
		dest.WriteBool(v.BoolValue)
	case *otlpcommon.AnyValue_IntValue:
		dest.WriteObjectField("intValue")
		dest.WriteInt64(v.IntValue)
	case *otlpcommon.AnyValue_DoubleValue:
		dest.WriteObjectField("doubleValue")
		dest.WriteFloat64(v.DoubleValue)
	case *otlpcommon.AnyValue_BytesValue:
		dest.WriteObjectField("bytesValue")
		dest.WriteBytes(v.BytesValue)
	case *otlpcommon.AnyValue_ArrayValue:
		dest.WriteObjectField("arrayValue")
		MarshalJSONOrigArrayValue(v.ArrayValue, dest)
	case *otlpcommon.AnyValue_KvlistValue:
		dest.WriteObjectField("kvlistValue")
		MarshalJSONOrigKeyValueList(v.KvlistValue, dest)
	default:
		dest.ReportError(fmt.Errorf("invalid value type in the passed attribute value: %T", orig.Value))
	}
	dest.WriteObjectEnd()
}

// UnmarshalJSONOrigAnyValue Unmarshal JSON data and return otlpcommon.AnyValue
func UnmarshalJSONOrigAnyValue(orig *otlpcommon.AnyValue, iter *json.Iterator) {
	for f := iter.ReadObject(); f != ""; f = iter.ReadObject() {
		switch f {
		case "stringValue", "string_value":
			ov := NewOrigAnyValueStringValue()
			ov.StringValue = iter.ReadString()
			orig.Value = ov
		case "boolValue", "bool_value":
			ov := NewOrigAnyValueBoolValue()
			ov.BoolValue = iter.ReadBool()
			orig.Value = ov
		case "intValue", "int_value":
			ov := NewOrigAnyValueIntValue()
			ov.IntValue = iter.ReadInt64()
			orig.Value = ov
		case "doubleValue", "double_value":
			ov := NewOrigAnyValueDoubleValue()
			ov.DoubleValue = iter.ReadFloat64()
			orig.Value = ov
		case "bytesValue", "bytes_value":
			ov := NewOrigAnyValueBytesValue()
			ov.BytesValue = iter.ReadBytes()
			orig.Value = ov
		case "arrayValue", "array_value":
			ov := NewOrigAnyValueArrayValue()
			ov.ArrayValue = NewOrigArrayValue()
			UnmarshalJSONOrigArrayValue(ov.ArrayValue, iter)
			orig.Value = ov
		case "kvlistValue", "kvlist_value":
			ov := NewOrigAnyValueKvlistValue()
			ov.KvlistValue = NewOrigKeyValueList()
			UnmarshalJSONOrigKeyValueList(ov.KvlistValue, iter)
			orig.Value = ov
		default:
			iter.Skip()
		}
	}
}

func GenTestOrigAnyValue() *otlpcommon.AnyValue {
	ov := NewOrigAnyValueStringValue()
	ov.StringValue = "v"
	orig := NewOrigAnyValue()
	orig.Value = ov
	return orig
}

func SizeProtoOrigAnyValue(orig *otlpcommon.AnyValue) int {
	return orig.Size()
}

func MarshalProtoOrigAnyValue(orig *otlpcommon.AnyValue, buf []byte) int {
	size, _ := orig.MarshalToSizedBuffer(buf)
	return size
}

func UnmarshalProtoOrigAnyValue(orig *otlpcommon.AnyValue, buf []byte) error {
	var err error
	var fieldNum int32
	var wireType proto.WireType

	l := len(buf)
	pos := 0
	for pos < l {
		// If in a group parsing, move to the next tag.
		fieldNum, wireType, pos, err = proto.ConsumeTag(buf, pos)
		if err != nil {
			return err
		}
		switch fieldNum {
		case 1:
			if wireType != proto.WireTypeLen {
				return fmt.Errorf("proto: wrong wireType = %d for field StringValue", wireType)
			}
			var length int
			length, pos, err = proto.ConsumeLen(buf, pos)
			if err != nil {
				return err
			}
			startPos := pos - length
			ov := NewOrigAnyValueStringValue()
			ov.StringValue = string(buf[startPos:pos])
			orig.Value = ov

		case 2:
			if wireType != proto.WireTypeVarint {
				return fmt.Errorf("proto: wrong wireType = %d for field BoolValue", wireType)
			}
			var num uint64
			num, pos, err = proto.ConsumeVarint(buf, pos)
			if err != nil {
				return err
			}
			ov := NewOrigAnyValueBoolValue()
			ov.BoolValue = num != 0
			orig.Value = ov

		case 3:
			if wireType != proto.WireTypeVarint {
				return fmt.Errorf("proto: wrong wireType = %d for field IntValue", wireType)
			}
			var num uint64
			num, pos, err = proto.ConsumeVarint(buf, pos)
			if err != nil {
				return err
			}
			ov := NewOrigAnyValueIntValue()
			ov.IntValue = int64(num) //nolint:gosec // G115
			orig.Value = ov

		case 4:
			if wireType != proto.WireTypeI64 {
				return fmt.Errorf("proto: wrong wireType = %d for field DoubleValue", wireType)
			}
			var num uint64
			num, pos, err = proto.ConsumeI64(buf, pos)
			if err != nil {
				return err
			}
			ov := NewOrigAnyValueDoubleValue()
			ov.DoubleValue = math.Float64frombits(num)
			orig.Value = ov

		case 5:
			if wireType != proto.WireTypeLen {
				return fmt.Errorf("proto: wrong wireType = %d for field ArrayValue", wireType)
			}
			var length int
			length, pos, err = proto.ConsumeLen(buf, pos)
			if err != nil {
				return err
			}
			startPos := pos - length
			ov := NewOrigAnyValueArrayValue()
			ov.ArrayValue = NewOrigArrayValue()
			err = UnmarshalProtoOrigArrayValue(ov.ArrayValue, buf[startPos:pos])
			if err != nil {
				return err
			}
			orig.Value = ov

		case 6:
			if wireType != proto.WireTypeLen {
				return fmt.Errorf("proto: wrong wireType = %d for field KvlistValue", wireType)
			}
			var length int
			length, pos, err = proto.ConsumeLen(buf, pos)
			if err != nil {
				return err
			}
			startPos := pos - length
			ov := NewOrigAnyValueKvlistValue()
			ov.KvlistValue = NewOrigKeyValueList()
			err = UnmarshalProtoOrigKeyValueList(ov.KvlistValue, buf[startPos:pos])
			if err != nil {
				return err
			}
			orig.Value = ov

		case 7:
			if wireType != proto.WireTypeLen {
				return fmt.Errorf("proto: wrong wireType = %d for field BytesValue", wireType)
			}
			var length int
			length, pos, err = proto.ConsumeLen(buf, pos)
			if err != nil {
				return err
			}
			startPos := pos - length
			ov := NewOrigAnyValueBytesValue()
			ov.BytesValue = make([]byte, length)
			copy(ov.BytesValue, buf[startPos:pos])
			orig.Value = ov

		default:
			pos, err = proto.ConsumeUnknown(buf, pos, wireType)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
