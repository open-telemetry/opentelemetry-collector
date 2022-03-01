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

package pdata // import "go.opentelemetry.io/collector/model/internal/pdata"

// This file contains data structures that are common for all telemetry types,
// such as timestamps, attributes, etc.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	otlpcommon "go.opentelemetry.io/collector/model/internal/data/protogen/common/v1"
)

// ValueType specifies the type of Value.
type ValueType int32

const (
	ValueTypeEmpty ValueType = iota
	ValueTypeString
	ValueTypeInt
	ValueTypeDouble
	ValueTypeBool
	ValueTypeMap
	ValueTypeArray
	ValueTypeBytes
)

// String returns the string representation of the ValueType.
func (avt ValueType) String() string {
	switch avt {
	case ValueTypeEmpty:
		return "EMPTY"
	case ValueTypeString:
		return "STRING"
	case ValueTypeBool:
		return "BOOL"
	case ValueTypeInt:
		return "INT"
	case ValueTypeDouble:
		return "DOUBLE"
	case ValueTypeMap:
		return "MAP"
	case ValueTypeArray:
		return "ARRAY"
	case ValueTypeBytes:
		return "BYTES"
	}
	return ""
}

// Value is a mutable cell containing the value of an attribute. Typically used in AttributeMap.
// Must use one of NewValue+ functions below to create new instances.
//
// Intended to be passed by value since internally it is just a pointer to actual
// value representation. For the same reason passing by value and calling setters
// will modify the original, e.g.:
//
//   func f1(val Value) { val.SetIntVal(234) }
//   func f2() {
//       v := NewValueString("a string")
//       f1(v)
//       _ := v.Type() // this will return ValueTypeInt
//   }
//
// Important: zero-initialized instance is not valid for use. All Value functions below must
// be called only on instances that are created via NewValue+ functions.
type Value struct {
	orig *otlpcommon.AnyValue
}

func newValue(orig *otlpcommon.AnyValue) Value {
	return Value{orig}
}

// NewValueEmpty creates a new Value with an empty value.
func NewValueEmpty() Value {
	return Value{orig: &otlpcommon.AnyValue{}}
}

// NewValueString creates a new Value with the given string value.
func NewValueString(v string) Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: v}}}
}

// NewValueInt creates a new Value with the given int64 value.
func NewValueInt(v int64) Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: v}}}
}

// NewValueDouble creates a new Value with the given float64 value.
func NewValueDouble(v float64) Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: v}}}
}

// NewValueBool creates a new Value with the given bool value.
func NewValueBool(v bool) Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BoolValue{BoolValue: v}}}
}

// NewValueMap creates a new Value of map type.
func NewValueMap() Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}}}
}

// NewValueArray creates a new Value of array type.
func NewValueArray() Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}}}
}

// NewValueBytes creates a new Value with the given []byte value.
// The caller must ensure the []byte passed in is not modified after the call is made, sharing the data
// across multiple attributes is forbidden.
func NewValueBytes(v []byte) Value {
	return Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BytesValue{BytesValue: v}}}
}

// Type returns the type of the value for this Value.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) Type() ValueType {
	if v.orig.Value == nil {
		return ValueTypeEmpty
	}
	switch v.orig.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		return ValueTypeString
	case *otlpcommon.AnyValue_BoolValue:
		return ValueTypeBool
	case *otlpcommon.AnyValue_IntValue:
		return ValueTypeInt
	case *otlpcommon.AnyValue_DoubleValue:
		return ValueTypeDouble
	case *otlpcommon.AnyValue_KvlistValue:
		return ValueTypeMap
	case *otlpcommon.AnyValue_ArrayValue:
		return ValueTypeArray
	case *otlpcommon.AnyValue_BytesValue:
		return ValueTypeBytes
	}
	return ValueTypeEmpty
}

// StringVal returns the string value associated with this Value.
// If the Type() is not ValueTypeString then returns empty string.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) StringVal() string {
	return v.orig.GetStringValue()
}

// IntVal returns the int64 value associated with this Value.
// If the Type() is not ValueTypeInt then returns int64(0).
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) IntVal() int64 {
	return v.orig.GetIntValue()
}

// DoubleVal returns the float64 value associated with this Value.
// If the Type() is not ValueTypeDouble then returns float64(0).
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) DoubleVal() float64 {
	return v.orig.GetDoubleValue()
}

// BoolVal returns the bool value associated with this Value.
// If the Type() is not ValueTypeBool then returns false.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) BoolVal() bool {
	return v.orig.GetBoolValue()
}

// MapVal returns the map value associated with this Value.
// If the Type() is not ValueTypeMap then returns an empty map. Note that modifying
// such empty map has no effect on this Value.
//
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) MapVal() AttributeMap {
	kvlist := v.orig.GetKvlistValue()
	if kvlist == nil {
		return NewAttributeMap()
	}
	return newAttributeMap(&kvlist.Values)
}

// SliceVal returns the slice value associated with this Value.
// If the Type() is not ValueTypeArray then returns an empty slice. Note that modifying
// such empty slice has no effect on this Value.
//
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) SliceVal() AttributeValueSlice {
	arr := v.orig.GetArrayValue()
	if arr == nil {
		return NewAttributeValueSlice()
	}
	return newAttributeValueSlice(&arr.Values)
}

// BytesVal returns the []byte value associated with this Value.
// If the Type() is not ValueTypeBytes then returns false.
// Calling this function on zero-initialized Value will cause a panic.
// Modifying the returned []byte in-place is forbidden.
func (v Value) BytesVal() []byte {
	return v.orig.GetBytesValue()
}

// SetStringVal replaces the string value associated with this Value,
// it also changes the type to be ValueTypeString.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) SetStringVal(sv string) {
	v.orig.Value = &otlpcommon.AnyValue_StringValue{StringValue: sv}
}

// SetIntVal replaces the int64 value associated with this Value,
// it also changes the type to be ValueTypeInt.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) SetIntVal(iv int64) {
	v.orig.Value = &otlpcommon.AnyValue_IntValue{IntValue: iv}
}

// SetDoubleVal replaces the float64 value associated with this Value,
// it also changes the type to be ValueTypeDouble.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) SetDoubleVal(dv float64) {
	v.orig.Value = &otlpcommon.AnyValue_DoubleValue{DoubleValue: dv}
}

// SetBoolVal replaces the bool value associated with this Value,
// it also changes the type to be ValueTypeBool.
// Calling this function on zero-initialized Value will cause a panic.
func (v Value) SetBoolVal(bv bool) {
	v.orig.Value = &otlpcommon.AnyValue_BoolValue{BoolValue: bv}
}

// SetBytesVal replaces the []byte value associated with this Value,
// it also changes the type to be ValueTypeBytes.
// Calling this function on zero-initialized Value will cause a panic.
// The caller must ensure the []byte passed in is not modified after the call is made, sharing the data
// across multiple attributes is forbidden.
func (v Value) SetBytesVal(bv []byte) {
	v.orig.Value = &otlpcommon.AnyValue_BytesValue{BytesValue: bv}
}

// copyTo copies the value to Value. Will panic if dest is nil.
func (v Value) copyTo(dest *otlpcommon.AnyValue) {
	switch ov := v.orig.Value.(type) {
	case *otlpcommon.AnyValue_KvlistValue:
		kv, ok := dest.Value.(*otlpcommon.AnyValue_KvlistValue)
		if !ok {
			kv = &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}
			dest.Value = kv
		}
		if ov.KvlistValue == nil {
			kv.KvlistValue = nil
			return
		}
		// Deep copy to dest.
		newAttributeMap(&ov.KvlistValue.Values).CopyTo(newAttributeMap(&kv.KvlistValue.Values))
	case *otlpcommon.AnyValue_ArrayValue:
		av, ok := dest.Value.(*otlpcommon.AnyValue_ArrayValue)
		if !ok {
			av = &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}
			dest.Value = av
		}
		if ov.ArrayValue == nil {
			av.ArrayValue = nil
			return
		}
		// Deep copy to dest.
		newAttributeValueSlice(&ov.ArrayValue.Values).CopyTo(newAttributeValueSlice(&av.ArrayValue.Values))
	default:
		// Primitive immutable type, no need for deep copy.
		dest.Value = v.orig.Value
	}
}

// CopyTo copies the attribute to a destination.
func (v Value) CopyTo(dest Value) {
	v.copyTo(dest.orig)
}

// Equal checks for equality, it returns true if the objects are equal otherwise false.
func (v Value) Equal(av Value) bool {
	if v.orig == av.orig {
		return true
	}

	if v.orig.Value == nil || av.orig.Value == nil {
		return v.orig.Value == av.orig.Value
	}

	if v.Type() != av.Type() {
		return false
	}

	switch v := v.orig.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		return v.StringValue == av.orig.GetStringValue()
	case *otlpcommon.AnyValue_BoolValue:
		return v.BoolValue == av.orig.GetBoolValue()
	case *otlpcommon.AnyValue_IntValue:
		return v.IntValue == av.orig.GetIntValue()
	case *otlpcommon.AnyValue_DoubleValue:
		return v.DoubleValue == av.orig.GetDoubleValue()
	case *otlpcommon.AnyValue_ArrayValue:
		vv := v.ArrayValue.GetValues()
		avv := av.orig.GetArrayValue().GetValues()
		if len(vv) != len(avv) {
			return false
		}

		for i, val := range avv {
			val := val
			newAv := newValue(&vv[i])

			// According to the specification, array values must be scalar.
			if avType := newAv.Type(); avType == ValueTypeArray || avType == ValueTypeMap {
				return false
			}

			if !newAv.Equal(newValue(&val)) {
				return false
			}
		}
		return true
	case *otlpcommon.AnyValue_KvlistValue:
		cc := v.KvlistValue.GetValues()
		avv := av.orig.GetKvlistValue().GetValues()
		if len(cc) != len(avv) {
			return false
		}

		am := newAttributeMap(&avv)

		for _, val := range cc {
			newAv, ok := am.Get(val.Key)
			if !ok {
				return false
			}

			if !newAv.Equal(newValue(&val.Value)) {
				return false
			}
		}
		return true
	case *otlpcommon.AnyValue_BytesValue:
		return bytes.Equal(v.BytesValue, av.orig.GetBytesValue())
	}

	return false
}

// AsString converts an OTLP Value object of any type to its equivalent string
// representation. This differs from StringVal which only returns a non-empty value
// if the ValueType is ValueTypeString.
func (v Value) AsString() string {
	switch v.Type() {
	case ValueTypeEmpty:
		return ""

	case ValueTypeString:
		return v.StringVal()

	case ValueTypeBool:
		return strconv.FormatBool(v.BoolVal())

	case ValueTypeDouble:
		jsonStr, _ := json.Marshal(v.DoubleVal())
		return string(jsonStr)

	case ValueTypeInt:
		return strconv.FormatInt(v.IntVal(), 10)

	case ValueTypeMap:
		jsonStr, _ := json.Marshal(v.MapVal().AsRaw())
		return string(jsonStr)

	case ValueTypeBytes:
		return base64.StdEncoding.EncodeToString(v.BytesVal())

	case ValueTypeArray:
		jsonStr, _ := json.Marshal(v.SliceVal().asRaw())
		return string(jsonStr)

	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", v.Type())
	}
}

func newAttributeKeyValueString(k string, v string) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := Value{&orig.Value}
	akv.SetStringVal(v)
	return orig
}

func newAttributeKeyValueInt(k string, v int64) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := Value{&orig.Value}
	akv.SetIntVal(v)
	return orig
}

func newAttributeKeyValueDouble(k string, v float64) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := Value{&orig.Value}
	akv.SetDoubleVal(v)
	return orig
}

func newAttributeKeyValueBool(k string, v bool) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := Value{&orig.Value}
	akv.SetBoolVal(v)
	return orig
}

func newAttributeKeyValueNull(k string) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	return orig
}

func newAttributeKeyValue(k string, av Value) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	av.copyTo(&orig.Value)
	return orig
}

func newAttributeKeyValueBytes(k string, v []byte) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := Value{&orig.Value}
	akv.SetBytesVal(v)
	return orig
}

// AttributeMap stores a map of attribute keys to values.
type AttributeMap struct {
	orig *[]otlpcommon.KeyValue
}

// NewAttributeMap creates a AttributeMap with 0 elements.
func NewAttributeMap() AttributeMap {
	orig := []otlpcommon.KeyValue(nil)
	return AttributeMap{&orig}
}

// NewAttributeMapFromMap creates a AttributeMap with values
// from the given map[string]Value.
func NewAttributeMapFromMap(attrMap map[string]Value) AttributeMap {
	if len(attrMap) == 0 {
		kv := []otlpcommon.KeyValue(nil)
		return AttributeMap{&kv}
	}
	origs := make([]otlpcommon.KeyValue, len(attrMap))
	ix := 0
	for k, v := range attrMap {
		origs[ix].Key = k
		v.copyTo(&origs[ix].Value)
		ix++
	}
	return AttributeMap{&origs}
}

func newAttributeMap(orig *[]otlpcommon.KeyValue) AttributeMap {
	return AttributeMap{orig}
}

// Clear erases any existing entries in this AttributeMap instance.
func (am AttributeMap) Clear() {
	*am.orig = nil
}

// EnsureCapacity increases the capacity of this AttributeMap instance, if necessary,
// to ensure that it can hold at least the number of elements specified by the capacity argument.
func (am AttributeMap) EnsureCapacity(capacity int) {
	if capacity <= cap(*am.orig) {
		return
	}
	oldOrig := *am.orig
	*am.orig = make([]otlpcommon.KeyValue, 0, capacity)
	copy(*am.orig, oldOrig)
}

// Get returns the Value associated with the key and true. Returned
// Value is not a copy, it is a reference to the value stored in this map.
// It is allowed to modify the returned value using Value.Set* functions.
// Such modification will be applied to the value stored in this map.
//
// If the key does not exist returns an invalid instance of the KeyValue and false.
// Calling any functions on the returned invalid instance will cause a panic.
func (am AttributeMap) Get(key string) (Value, bool) {
	for i := range *am.orig {
		akv := &(*am.orig)[i]
		if akv.Key == key {
			return Value{&akv.Value}, true
		}
	}
	return Value{nil}, false
}

// Deprecated: [v0.47.0] Use Remove instead.
func (am AttributeMap) Delete(key string) bool {
	return am.Remove(key)
}

// Remove removes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (am AttributeMap) Remove(key string) bool {
	for i := range *am.orig {
		akv := &(*am.orig)[i]
		if akv.Key == key {
			*akv = (*am.orig)[len(*am.orig)-1]
			*am.orig = (*am.orig)[:len(*am.orig)-1]
			return true
		}
	}
	return false
}

// RemoveIf removes the entries for which the function in question returns true
func (am AttributeMap) RemoveIf(f func(string, Value) bool) {
	newLen := 0
	for i := 0; i < len(*am.orig); i++ {
		akv := &(*am.orig)[i]
		if f(akv.Key, Value{&akv.Value}) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*am.orig)[newLen] = (*am.orig)[i]
		newLen++
	}
	*am.orig = (*am.orig)[:newLen]
}

// Insert adds the Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
//
// Calling this function with a zero-initialized Value struct will cause a panic.
//
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Insert(k string, v Value) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValue(k, v))
	}
}

// InsertNull adds a null Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertNull(k string) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueNull(k))
	}
}

// InsertString adds the string Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertString(k string, v string) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueString(k, v))
	}
}

// InsertInt adds the int Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertInt(k string, v int64) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueInt(k, v))
	}
}

// InsertDouble adds the double Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertDouble(k string, v float64) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueDouble(k, v))
	}
}

// InsertBool adds the bool Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertBool(k string, v bool) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueBool(k, v))
	}
}

// InsertBytes adds the []byte Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
// The caller must ensure the []byte passed in is not modified after the call is made, sharing the data
// across multiple attributes is forbidden.
func (am AttributeMap) InsertBytes(k string, v []byte) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueBytes(k, v))
	}
}

// Update updates an existing Value with a value.
// No action is applied to the map where the key does not exist.
//
// Calling this function with a zero-initialized Value struct will cause a panic.
//
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Update(k string, v Value) {
	if av, existing := am.Get(k); existing {
		v.copyTo(av.orig)
	}
}

// UpdateString updates an existing string Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateString(k string, v string) {
	if av, existing := am.Get(k); existing {
		av.SetStringVal(v)
	}
}

// UpdateInt updates an existing int Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateInt(k string, v int64) {
	if av, existing := am.Get(k); existing {
		av.SetIntVal(v)
	}
}

// UpdateDouble updates an existing double Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateDouble(k string, v float64) {
	if av, existing := am.Get(k); existing {
		av.SetDoubleVal(v)
	}
}

// UpdateBool updates an existing bool Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateBool(k string, v bool) {
	if av, existing := am.Get(k); existing {
		av.SetBoolVal(v)
	}
}

// UpdateBytes updates an existing []byte Value with a value.
// No action is applied to the map where the key does not exist.
// The caller must ensure the []byte passed in is not modified after the call is made, sharing the data
// across multiple attributes is forbidden.
func (am AttributeMap) UpdateBytes(k string, v []byte) {
	if av, existing := am.Get(k); existing {
		av.SetBytesVal(v)
	}
}

// Upsert performs the Insert or Update action. The Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
//
// Calling this function with a zero-initialized Value struct will cause a panic.
//
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Upsert(k string, v Value) {
	if av, existing := am.Get(k); existing {
		v.copyTo(av.orig)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValue(k, v))
	}
}

// UpsertString performs the Insert or Update action. The Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertString(k string, v string) {
	if av, existing := am.Get(k); existing {
		av.SetStringVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueString(k, v))
	}
}

// UpsertInt performs the Insert or Update action. The int Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertInt(k string, v int64) {
	if av, existing := am.Get(k); existing {
		av.SetIntVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueInt(k, v))
	}
}

// UpsertDouble performs the Insert or Update action. The double Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertDouble(k string, v float64) {
	if av, existing := am.Get(k); existing {
		av.SetDoubleVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueDouble(k, v))
	}
}

// UpsertBool performs the Insert or Update action. The bool Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertBool(k string, v bool) {
	if av, existing := am.Get(k); existing {
		av.SetBoolVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueBool(k, v))
	}
}

// UpsertBytes performs the Insert or Update action. The []byte Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
// The caller must ensure the []byte passed in is not modified after the call is made, sharing the data
// across multiple attributes is forbidden.
func (am AttributeMap) UpsertBytes(k string, v []byte) {
	if av, existing := am.Get(k); existing {
		av.SetBytesVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueBytes(k, v))
	}
}

// Sort sorts the entries in the AttributeMap so two instances can be compared.
// Returns the same instance to allow nicer code like:
//   assert.EqualValues(t, expected.Sort(), actual.Sort())
func (am AttributeMap) Sort() AttributeMap {
	// Intention is to move the nil values at the end.
	sort.SliceStable(*am.orig, func(i, j int) bool {
		return (*am.orig)[i].Key < (*am.orig)[j].Key
	})
	return am
}

// Len returns the length of this map.
//
// Because the AttributeMap is represented internally by a slice of pointers, and the data are comping from the wire,
// it is possible that when iterating using "Range" to get access to fewer elements because nil elements are skipped.
func (am AttributeMap) Len() int {
	return len(*am.orig)
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
//
// Example:
//
//   sm.Range(func(k string, v Value) bool {
//       ...
//   })
func (am AttributeMap) Range(f func(k string, v Value) bool) {
	for i := range *am.orig {
		kv := &(*am.orig)[i]
		if !f(kv.Key, Value{&kv.Value}) {
			break
		}
	}
}

// CopyTo copies all elements from the current map to the dest.
func (am AttributeMap) CopyTo(dest AttributeMap) {
	newLen := len(*am.orig)
	oldCap := cap(*dest.orig)
	if newLen <= oldCap {
		// New slice fits in existing slice, no need to reallocate.
		*dest.orig = (*dest.orig)[:newLen:oldCap]
		for i := range *am.orig {
			akv := &(*am.orig)[i]
			destAkv := &(*dest.orig)[i]
			destAkv.Key = akv.Key
			Value{&akv.Value}.copyTo(&destAkv.Value)
		}
		return
	}

	// New slice is bigger than exist slice. Allocate new space.
	origs := make([]otlpcommon.KeyValue, len(*am.orig))
	for i := range *am.orig {
		akv := &(*am.orig)[i]
		origs[i].Key = akv.Key
		Value{&akv.Value}.copyTo(&origs[i].Value)
	}
	*dest.orig = origs
}

// AsRaw converts an OTLP AttributeMap to a standard go map
func (am AttributeMap) AsRaw() map[string]interface{} {
	rawMap := make(map[string]interface{})
	am.Range(func(k string, v Value) bool {
		switch v.Type() {
		case ValueTypeString:
			rawMap[k] = v.StringVal()
		case ValueTypeInt:
			rawMap[k] = v.IntVal()
		case ValueTypeDouble:
			rawMap[k] = v.DoubleVal()
		case ValueTypeBool:
			rawMap[k] = v.BoolVal()
		case ValueTypeBytes:
			rawMap[k] = v.BytesVal()
		case ValueTypeEmpty:
			rawMap[k] = nil
		case ValueTypeMap:
			rawMap[k] = v.MapVal().AsRaw()
		case ValueTypeArray:
			rawMap[k] = v.SliceVal().asRaw()
		}
		return true
	})
	return rawMap
}

// asRaw creates a slice out of a AttributeValueSlice.
func (es AttributeValueSlice) asRaw() []interface{} {
	rawSlice := make([]interface{}, 0, es.Len())
	for i := 0; i < es.Len(); i++ {
		v := es.At(i)
		switch v.Type() {
		case ValueTypeString:
			rawSlice = append(rawSlice, v.StringVal())
		case ValueTypeInt:
			rawSlice = append(rawSlice, v.IntVal())
		case ValueTypeDouble:
			rawSlice = append(rawSlice, v.DoubleVal())
		case ValueTypeBool:
			rawSlice = append(rawSlice, v.BoolVal())
		case ValueTypeBytes:
			rawSlice = append(rawSlice, v.BytesVal())
		case ValueTypeEmpty:
			rawSlice = append(rawSlice, nil)
		default:
			rawSlice = append(rawSlice, "<Invalid array value>")
		}
	}
	return rawSlice
}
