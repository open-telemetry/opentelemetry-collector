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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

// ValueType specifies the type of Value.
type ValueType int32

const (
	ValueTypeEmpty ValueType = iota
	ValueTypeStr
	ValueTypeInt
	ValueTypeDouble
	ValueTypeBool
	ValueTypeMap
	ValueTypeSlice
	ValueTypeBytes
)

// String returns the string representation of the ValueType.
func (avt ValueType) String() string {
	switch avt {
	case ValueTypeEmpty:
		return "Empty"
	case ValueTypeStr:
		return "Str"
	case ValueTypeBool:
		return "Bool"
	case ValueTypeInt:
		return "Int"
	case ValueTypeDouble:
		return "Double"
	case ValueTypeMap:
		return "Map"
	case ValueTypeSlice:
		return "Slice"
	case ValueTypeBytes:
		return "Bytes"
	}
	return ""
}

type commonValue interface {
	Type() ValueType
	Str() string
	Int() int64
	Double() float64
	Bool() bool
	CopyTo(dest MutableValue)
	AsString() string
	AsRaw() any
	getOrig() *otlpcommon.AnyValue
}

type Value interface {
	commonValue
	Map() Map
	Slice() Slice
	Bytes() ByteSlice
}

type MutableValue interface {
	commonValue
	Map() MutableMap
	Slice() MutableSlice
	Bytes() MutableByteSlice
	FromRaw(iv any) error
	SetStr(sv string)
	SetInt(iv int64)
	SetDouble(dv float64)
	SetBool(bv bool)
	SetEmptyBytes() MutableByteSlice
	SetEmptyMap() MutableMap
	SetEmptySlice() MutableSlice
}

type immutableValue struct {
	orig *otlpcommon.AnyValue
}

type mutableValue struct {
	immutableValue
}

func NewImmutableValue(orig *otlpcommon.AnyValue) Value {
	return immutableValue{orig}
}

func NewMutableValue(orig *otlpcommon.AnyValue) MutableValue {
	return mutableValue{immutableValue{orig}}
}

func (v immutableValue) getOrig() *otlpcommon.AnyValue {
	return v.orig
}

func FillTestValue(dest MutableValue) {
	dest.getOrig().Value = &otlpcommon.AnyValue_StringValue{StringValue: "v"}
}

func GenerateTestValue() MutableValue {
	var orig otlpcommon.AnyValue
	ms := NewMutableValue(&orig)
	FillTestValue(ms)
	return ms
}

// NewValueEmpty creates a new Value with an empty value.
func NewValueEmpty() MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{})
}

// NewValueStr creates a new Value with the given string value.
func NewValueStr(v string) MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: v}})
}

// NewValueInt creates a new Value with the given int64 value.
func NewValueInt(v int64) MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: v}})
}

// NewValueDouble creates a new Value with the given float64 value.
func NewValueDouble(v float64) MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: v}})
}

// NewValueBool creates a new Value with the given bool value.
func NewValueBool(v bool) MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BoolValue{BoolValue: v}})
}

// NewValueMap creates a new Value of map type.
func NewValueMap() MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.
		KeyValueList{}}})
}

// NewValueSlice creates a new Value of array type.
func NewValueSlice() MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.
		ArrayValue{}}})
}

// NewValueBytes creates a new empty Value of byte type.
func NewValueBytes() MutableValue {
	return NewMutableValue(&otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BytesValue{BytesValue: nil}})
}

func (v mutableValue) FromRaw(iv any) error {
	switch tv := iv.(type) {
	case nil:
		v.getOrig().Value = nil
	case string:
		v.SetStr(tv)
	case int:
		v.SetInt(int64(tv))
	case int8:
		v.SetInt(int64(tv))
	case int16:
		v.SetInt(int64(tv))
	case int32:
		v.SetInt(int64(tv))
	case int64:
		v.SetInt(tv)
	case uint:
		v.SetInt(int64(tv))
	case uint8:
		v.SetInt(int64(tv))
	case uint16:
		v.SetInt(int64(tv))
	case uint32:
		v.SetInt(int64(tv))
	case uint64:
		v.SetInt(int64(tv))
	case float32:
		v.SetDouble(float64(tv))
	case float64:
		v.SetDouble(tv)
	case bool:
		v.SetBool(tv)
	case []byte:
		v.SetEmptyBytes().FromRaw(tv)
	case map[string]any:
		return v.SetEmptyMap().FromRaw(tv)
	case []any:
		return v.SetEmptySlice().FromRaw(tv)
	default:
		return fmt.Errorf("<Invalid value type %T>", tv)
	}
	return nil
}

// Type returns the type of the value for this Value.
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Type() ValueType {
	switch v.getOrig().Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		return ValueTypeStr
	case *otlpcommon.AnyValue_BoolValue:
		return ValueTypeBool
	case *otlpcommon.AnyValue_IntValue:
		return ValueTypeInt
	case *otlpcommon.AnyValue_DoubleValue:
		return ValueTypeDouble
	case *otlpcommon.AnyValue_KvlistValue:
		return ValueTypeMap
	case *otlpcommon.AnyValue_ArrayValue:
		return ValueTypeSlice
	case *otlpcommon.AnyValue_BytesValue:
		return ValueTypeBytes
	}
	return ValueTypeEmpty
}

// Str returns the string value associated with this Value.
// The shorter name is used instead of String to avoid implementing fmt.Stringer interface.
// If the Type() is not ValueTypeStr then returns empty string.
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Str() string {
	return v.getOrig().GetStringValue()
}

// Int returns the int64 value associated with this Value.
// If the Type() is not ValueTypeInt then returns int64(0).
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Int() int64 {
	return v.getOrig().GetIntValue()
}

// Double returns the float64 value associated with this Value.
// If the Type() is not ValueTypeDouble then returns float64(0).
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Double() float64 {
	return v.getOrig().GetDoubleValue()
}

// Bool returns the bool value associated with this Value.
// If the Type() is not ValueTypeBool then returns false.
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Bool() bool {
	return v.getOrig().GetBoolValue()
}

// Map returns the map value associated with this Value.
// If the Type() is not ValueTypeMap then returns an invalid map. Note that using
// such map can cause panic.
//
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Map() Map {
	kvlist := v.getOrig().GetKvlistValue()
	if kvlist == nil {
		return immutableMap{}
	}
	return NewImmutableMap(&kvlist.Values)
}

func (v mutableValue) Map() MutableMap {
	return mutableMap{v.immutableValue.Map().(immutableMap)}
}

// Slice returns the slice value associated with this Value.
// If the Type() is not ValueTypeSlice then returns an invalid slice. Note that using
// such slice can cause panic.
//
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Slice() Slice {
	arr := v.getOrig().GetArrayValue()
	if arr == nil {
		return immutableSlice{}
	}
	return NewImmutableSlice(&arr.Values)
}

func (v mutableValue) Slice() MutableSlice {
	return mutableSlice{v.immutableValue.Slice().(immutableSlice)}
}

// Bytes returns the ByteSlice value associated with this Value.
// If the Type() is not ValueTypeBytes then returns an invalid ByteSlice object. Note that using
// such slice can cause panic.
//
// Calling this function on zero-initialized Value will cause a panic.
func (v immutableValue) Bytes() ByteSlice {
	bv, ok := v.getOrig().GetValue().(*otlpcommon.AnyValue_BytesValue)
	if !ok {
		return NewImmutableByteSlice(nil)
	}
	return NewImmutableByteSlice(&bv.BytesValue)
}

func (v mutableValue) Bytes() MutableByteSlice {
	return v.immutableValue.Bytes().(MutableByteSlice)
}

// SetStr replaces the string value associated with this Value,
// it also changes the type to be ValueTypeStr.
// The shorter name is used instead of SetString to avoid implementing
// fmt.Stringer interface by the corresponding getter method.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetStr(sv string) {
	v.getOrig().Value = &otlpcommon.AnyValue_StringValue{StringValue: sv}
}

// SetInt replaces the int64 value associated with this Value,
// it also changes the type to be ValueTypeInt.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetInt(iv int64) {
	v.getOrig().Value = &otlpcommon.AnyValue_IntValue{IntValue: iv}
}

// SetDouble replaces the float64 value associated with this Value,
// it also changes the type to be ValueTypeDouble.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetDouble(dv float64) {
	v.getOrig().Value = &otlpcommon.AnyValue_DoubleValue{DoubleValue: dv}
}

// SetBool replaces the bool value associated with this Value,
// it also changes the type to be ValueTypeBool.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetBool(bv bool) {
	v.getOrig().Value = &otlpcommon.AnyValue_BoolValue{BoolValue: bv}
}

// SetEmptyBytes sets value to an empty byte slice and returns it.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetEmptyBytes() MutableByteSlice {
	bv := otlpcommon.AnyValue_BytesValue{BytesValue: nil}
	v.getOrig().Value = &bv
	return NewMutableByteSlice(&bv.BytesValue)
}

// SetEmptyMap sets value to an empty map and returns it.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetEmptyMap() MutableMap {
	kv := &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}
	v.getOrig().Value = kv
	return NewMutableMap(&kv.KvlistValue.Values)
}

// SetEmptySlice sets value to an empty slice and returns it.
// Calling this function on zero-initialized Value will cause a panic.
func (v mutableValue) SetEmptySlice() MutableSlice {
	av := &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}
	v.getOrig().Value = av
	return NewMutableSlice(&av.ArrayValue.Values)
}

// CopyTo copies the Value instance overriding the destination.
func (v immutableValue) CopyTo(dest MutableValue) {
	destOrig := dest.getOrig()
	switch ov := v.getOrig().Value.(type) {
	case *otlpcommon.AnyValue_KvlistValue:
		kv, ok := destOrig.Value.(*otlpcommon.AnyValue_KvlistValue)
		if !ok {
			kv = &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}
			destOrig.Value = kv
		}
		if ov.KvlistValue == nil {
			kv.KvlistValue = nil
			return
		}
		// Deep copy to dest.
		NewImmutableMap(&ov.KvlistValue.Values).CopyTo(NewMutableMap(&kv.KvlistValue.Values))
	case *otlpcommon.AnyValue_ArrayValue:
		av, ok := destOrig.Value.(*otlpcommon.AnyValue_ArrayValue)
		if !ok {
			av = &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}
			destOrig.Value = av
		}
		if ov.ArrayValue == nil {
			av.ArrayValue = nil
			return
		}
		// Deep copy to dest.
		NewImmutableSlice(&ov.ArrayValue.Values).CopyTo(NewMutableSlice(&av.ArrayValue.Values))
	case *otlpcommon.AnyValue_BytesValue:
		bv, ok := destOrig.Value.(*otlpcommon.AnyValue_BytesValue)
		if !ok {
			bv = &otlpcommon.AnyValue_BytesValue{}
			destOrig.Value = bv
		}
		bv.BytesValue = make([]byte, len(ov.BytesValue))
		copy(bv.BytesValue, ov.BytesValue)
	default:
		// Primitive immutable type, no need for deep copy.
		destOrig.Value = ov
	}
}

// AsString converts an OTLP Value object of any type to its equivalent string
// representation. This differs from Str which only returns a non-empty value
// if the ValueType is ValueTypeStr.
func (v immutableValue) AsString() string {
	switch v.Type() {
	case ValueTypeEmpty:
		return ""

	case ValueTypeStr:
		return v.Str()

	case ValueTypeBool:
		return strconv.FormatBool(v.Bool())

	case ValueTypeDouble:
		return float64AsString(v.Double())

	case ValueTypeInt:
		return strconv.FormatInt(v.Int(), 10)

	case ValueTypeMap:
		jsonStr, _ := json.Marshal(v.Map().AsRaw())
		return string(jsonStr)

	case ValueTypeBytes:
		return base64.StdEncoding.EncodeToString(*v.Bytes().getOrig())

	case ValueTypeSlice:
		jsonStr, _ := json.Marshal(v.Slice().AsRaw())
		return string(jsonStr)

	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", v.Type())
	}
}

// See https://cs.opensource.google/go/go/+/refs/tags/go1.17.7:src/encoding/json/encode.go;l=585.
// This allows us to avoid using reflection.
func float64AsString(f float64) string {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return fmt.Sprintf("json: unsupported value: %s", strconv.FormatFloat(f, 'g', -1, 64))
	}

	// Convert as if by ES6 number to string conversion.
	// This matches most other JSON generators.
	// See golang.org/issue/6384 and golang.org/issue/14135.
	// Like fmt %g, but the exponent cutoffs are different
	// and exponents themselves are not padded to two digits.
	scratch := [64]byte{}
	b := scratch[:0]
	abs := math.Abs(f)
	fmt := byte('f')
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		fmt = 'e'
	}
	b = strconv.AppendFloat(b, f, fmt, -1, 64)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}
	return string(b)
}

func (v immutableValue) AsRaw() any {
	switch v.Type() {
	case ValueTypeEmpty:
		return nil
	case ValueTypeStr:
		return v.Str()
	case ValueTypeBool:
		return v.Bool()
	case ValueTypeDouble:
		return v.Double()
	case ValueTypeInt:
		return v.Int()
	case ValueTypeBytes:
		return v.Bytes().AsRaw()
	case ValueTypeMap:
		return v.Map().AsRaw()
	case ValueTypeSlice:
		return v.Slice().AsRaw()
	}
	return fmt.Sprintf("<Unknown OpenTelemetry value type %q>", v.Type())
}

func newKeyValueString(k string, v string) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := NewMutableValue(&orig.Value)
	akv.SetStr(v)
	return orig
}

func newKeyValueInt(k string, v int64) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := NewMutableValue(&orig.Value)
	akv.SetInt(v)
	return orig
}

func newKeyValueDouble(k string, v float64) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := NewMutableValue(&orig.Value)
	akv.SetDouble(v)
	return orig
}

func newKeyValueBool(k string, v bool) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := NewMutableValue(&orig.Value)
	akv.SetBool(v)
	return orig
}
