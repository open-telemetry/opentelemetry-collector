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

package pdata

// This file contains data structures that are common for all telemetry types,
// such as timestamps, attributes, etc.

import (
	"sort"
	"time"

	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
)

// TimestampUnixNano is a time specified as UNIX Epoch time in nanoseconds since
// 00:00:00 UTC on 1 January 1970.
type TimestampUnixNano uint64

func (ts TimestampUnixNano) String() string {
	return time.Unix(0, int64(ts)).String()
}

// AttributeValueType specifies the type of AttributeValue.
type AttributeValueType int

const (
	AttributeValueNULL AttributeValueType = iota
	AttributeValueSTRING
	AttributeValueINT
	AttributeValueDOUBLE
	AttributeValueBOOL
	AttributeValueMAP
	AttributeValueARRAY
)

func (avt AttributeValueType) String() string {
	switch avt {
	case AttributeValueNULL:
		return "NULL"
	case AttributeValueSTRING:
		return "STRING"
	case AttributeValueBOOL:
		return "BOOL"
	case AttributeValueINT:
		return "INT"
	case AttributeValueDOUBLE:
		return "DOUBLE"
	case AttributeValueMAP:
		return "MAP"
	case AttributeValueARRAY:
		return "ARRAY"
	}
	return ""
}

// AttributeValue represents a value of an attribute. Typically used in AttributeMap.
// Must use one of NewAttributeValue+ functions below to create new instances.
//
// Intended to be passed by value since internally it is just a pointer to actual
// value representation. For the same reason passing by value and calling setters
// will modify the original, e.g.:
//
//   function f1(val AttributeValue) { val.SetIntVal(234) }
//   function f2() {
//   	v := NewAttributeValueString("a string")
//      f1(v)
//      _ := v.Type() // this will return AttributeValueINT
//   }
//
// Important: zero-initialized instance is not valid for use. All AttributeValue functions bellow must
// be called only on instances that are created via NewAttributeValue+ functions.
type AttributeValue struct {
	orig *otlpcommon.AnyValue
}

func newAttributeValue(orig *otlpcommon.AnyValue) AttributeValue {
	return AttributeValue{orig}
}

// NewAttributeValueNull creates a new AttributeValue with a null value.
func NewAttributeValueNull() AttributeValue {
	orig := &otlpcommon.AnyValue{}
	return AttributeValue{orig: orig}
}

// Deprecated: Use NewAttributeValueNull()
func NewAttributeValue() AttributeValue {
	return NewAttributeValueNull()
}

// NewAttributeValueString creates a new AttributeValue with the given string value.
func NewAttributeValueString(v string) AttributeValue {
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: v}}
	return AttributeValue{orig: orig}
}

// NewAttributeValueInt creates a new AttributeValue with the given int64 value.
func NewAttributeValueInt(v int64) AttributeValue {
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: v}}
	return AttributeValue{orig: orig}
}

// NewAttributeValueDouble creates a new AttributeValue with the given float64 value.
func NewAttributeValueDouble(v float64) AttributeValue {
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_DoubleValue{DoubleValue: v}}
	return AttributeValue{orig: orig}
}

// NewAttributeValueBool creates a new AttributeValue with the given bool value.
func NewAttributeValueBool(v bool) AttributeValue {
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BoolValue{BoolValue: v}}
	return AttributeValue{orig: orig}
}

// NewAttributeValueMap creates a new AttributeValue of map type.
func NewAttributeValueMap() AttributeValue {
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}}
	return AttributeValue{orig: orig}
}

// NewAttributeValueArray creates a new AttributeValue of array type.
func NewAttributeValueArray() AttributeValue {
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}}
	return AttributeValue{orig: orig}
}

// Type returns the type of the value for this AttributeValue.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) Type() AttributeValueType {
	if a.orig.Value == nil {
		return AttributeValueNULL
	}
	switch a.orig.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		return AttributeValueSTRING
	case *otlpcommon.AnyValue_BoolValue:
		return AttributeValueBOOL
	case *otlpcommon.AnyValue_IntValue:
		return AttributeValueINT
	case *otlpcommon.AnyValue_DoubleValue:
		return AttributeValueDOUBLE
	case *otlpcommon.AnyValue_KvlistValue:
		return AttributeValueMAP
	case *otlpcommon.AnyValue_ArrayValue:
		return AttributeValueARRAY
	}
	return AttributeValueNULL
}

// StringVal returns the string value associated with this AttributeValue.
// If the Type() is not AttributeValueSTRING then returns empty string.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) StringVal() string {
	return a.orig.GetStringValue()
}

// IntVal returns the int64 value associated with this AttributeValue.
// If the Type() is not AttributeValueINT then returns int64(0).
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) IntVal() int64 {
	return a.orig.GetIntValue()
}

// DoubleVal returns the float64 value associated with this AttributeValue.
// If the Type() is not AttributeValueDOUBLE then returns float64(0).
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) DoubleVal() float64 {
	return a.orig.GetDoubleValue()
}

// BoolVal returns the bool value associated with this AttributeValue.
// If the Type() is not AttributeValueBOOL then returns false.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) BoolVal() bool {
	return a.orig.GetBoolValue()
}

// MapVal returns the map value associated with this AttributeValue.
// If the Type() is not AttributeValueMAP then returns an empty map. Note that modifying
// such empty map has no effect on this AttributeValue.
//
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) MapVal() AttributeMap {
	kvlist := a.orig.GetKvlistValue()
	if kvlist == nil {
		return NewAttributeMap()
	}
	return newAttributeMap(&kvlist.Values)
}

// ArrayVal returns the array value associated with this AttributeValue.
// If the Type() is not AttributeValueARRAY then returns an empty array. Note that modifying
// such empty array has no effect on this AttributeValue.
//
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) ArrayVal() AnyValueArray {
	arr := a.orig.GetArrayValue()
	if arr == nil {
		return NewAnyValueArray()
	}
	return newAnyValueArray(&arr.Values)
}

// SetStringVal replaces the string value associated with this AttributeValue,
// it also changes the type to be AttributeValueSTRING.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetStringVal(v string) {
	a.orig.Value = &otlpcommon.AnyValue_StringValue{StringValue: v}
}

// SetIntVal replaces the int64 value associated with this AttributeValue,
// it also changes the type to be AttributeValueINT.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetIntVal(v int64) {
	a.orig.Value = &otlpcommon.AnyValue_IntValue{IntValue: v}
}

// SetDoubleVal replaces the float64 value associated with this AttributeValue,
// it also changes the type to be AttributeValueDOUBLE.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetDoubleVal(v float64) {
	a.orig.Value = &otlpcommon.AnyValue_DoubleValue{DoubleValue: v}
}

// SetBoolVal replaces the bool value associated with this AttributeValue,
// it also changes the type to be AttributeValueBOOL.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetBoolVal(v bool) {
	a.orig.Value = &otlpcommon.AnyValue_BoolValue{BoolValue: v}
}

// copyTo copies the value to AnyValue. Will panic if dest is nil.
func (a AttributeValue) copyTo(dest *otlpcommon.AnyValue) {
	switch v := a.orig.Value.(type) {
	case *otlpcommon.AnyValue_KvlistValue:
		kv, ok := dest.Value.(*otlpcommon.AnyValue_KvlistValue)
		if !ok {
			kv = &otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{}}
			dest.Value = kv
		}
		if v.KvlistValue == nil {
			kv.KvlistValue = nil
			return
		}
		// Deep copy to dest.
		newAttributeMap(&v.KvlistValue.Values).CopyTo(newAttributeMap(&kv.KvlistValue.Values))
	case *otlpcommon.AnyValue_ArrayValue:
		av, ok := dest.Value.(*otlpcommon.AnyValue_ArrayValue)
		if !ok {
			av = &otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{}}
			dest.Value = av
		}
		if v.ArrayValue == nil {
			av.ArrayValue = nil
			return
		}
		// Deep copy to dest.
		newAnyValueArray(&v.ArrayValue.Values).CopyTo(newAnyValueArray(&av.ArrayValue.Values))
	default:
		// Primitive immutable type, no need for deep copy.
		dest.Value = a.orig.Value
	}
}

func (a AttributeValue) CopyTo(dest AttributeValue) {
	a.copyTo(dest.orig)
}

// Equal checks for equality, it returns true if the objects are equal otherwise false.
func (a AttributeValue) Equal(av AttributeValue) bool {
	if a.orig.Value == nil || av.orig.Value == nil {
		return a.orig.Value == av.orig.Value
	}

	switch v := a.orig.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		return v.StringValue == av.orig.GetStringValue()
	case *otlpcommon.AnyValue_BoolValue:
		return v.BoolValue == av.orig.GetBoolValue()
	case *otlpcommon.AnyValue_IntValue:
		return v.IntValue == av.orig.GetIntValue()
	case *otlpcommon.AnyValue_DoubleValue:
		return v.DoubleValue == av.orig.GetDoubleValue()
	}
	// TODO: handle MAP and ARRAY data types.
	return false
}

func newAttributeKeyValueString(k string, v string) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := AttributeValue{&orig.Value}
	akv.SetStringVal(v)
	return orig
}

func newAttributeKeyValueInt(k string, v int64) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := AttributeValue{&orig.Value}
	akv.SetIntVal(v)
	return orig
}

func newAttributeKeyValueDouble(k string, v float64) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := AttributeValue{&orig.Value}
	akv.SetDoubleVal(v)
	return orig
}

func newAttributeKeyValueBool(k string, v bool) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	akv := AttributeValue{&orig.Value}
	akv.SetBoolVal(v)
	return orig
}

func newAttributeKeyValueNull(k string) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	return orig
}

func newAttributeKeyValue(k string, av AttributeValue) otlpcommon.KeyValue {
	orig := otlpcommon.KeyValue{Key: k}
	av.copyTo(&orig.Value)
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

func newAttributeMap(orig *[]otlpcommon.KeyValue) AttributeMap {
	return AttributeMap{orig}
}

// InitFromMap overwrites the entire AttributeMap and reconstructs the AttributeMap
// with values from the given map[string]string.
//
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, NewAttributeMap().InitFromMap(map[string]AttributeValue{...}), actual)
func (am AttributeMap) InitFromMap(attrMap map[string]AttributeValue) AttributeMap {
	if len(attrMap) == 0 {
		*am.orig = []otlpcommon.KeyValue(nil)
		return am
	}
	origs := make([]otlpcommon.KeyValue, len(attrMap))
	ix := 0
	for k, v := range attrMap {
		origs[ix].Key = k
		v.copyTo(&origs[ix].Value)
		ix++
	}
	*am.orig = origs
	return am
}

// InitEmptyWithCapacity constructs an empty AttributeMap with predefined slice capacity.
func (am AttributeMap) InitEmptyWithCapacity(cap int) {
	if cap == 0 {
		*am.orig = []otlpcommon.KeyValue(nil)
		return
	}
	*am.orig = make([]otlpcommon.KeyValue, 0, cap)
}

// Get returns the AttributeValue associated with the key and true. Returned
// AttributeValue is not a copy, it is a reference to the value stored in this map.
// It is allowed to modify the returned value using AttributeValue.Set* functions.
// Such modification will be applied to the value stored in this map.
//
// If the key does not exist returns an invalid instance of the KeyValue and false.
// Calling any functions on the returned invalid instance will cause a panic.
func (am AttributeMap) Get(key string) (AttributeValue, bool) {
	for i := range *am.orig {
		akv := &(*am.orig)[i]
		if akv.Key == key {
			return AttributeValue{&akv.Value}, true
		}
	}
	return AttributeValue{nil}, false
}

// Delete deletes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (am AttributeMap) Delete(key string) bool {
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

// Insert adds the AttributeValue to the map when the key does not exist.
// No action is applied to the map where the key already exists.
//
// Calling this function with a zero-initialized AttributeValue struct will cause a panic.
//
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Insert(k string, v AttributeValue) {
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

// Update updates an existing AttributeValue with a value.
// No action is applied to the map where the key does not exist.
//
// Calling this function with a zero-initialized AttributeValue struct will cause a panic.
//
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Update(k string, v AttributeValue) {
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

// Upsert performs the Insert or Update action. The AttributeValue is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
//
// Calling this function with a zero-initialized AttributeValue struct will cause a panic.
//
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Upsert(k string, v AttributeValue) {
	if av, existing := am.Get(k); existing {
		v.copyTo(av.orig)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValue(k, v))
	}
}

// UpsertString performs the Insert or Update action. The AttributeValue is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertString(k string, v string) {
	if av, existing := am.Get(k); existing {
		av.SetStringVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueString(k, v))
	}
}

// UpsertInt performs the Insert or Update action. The int Value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertInt(k string, v int64) {
	if av, existing := am.Get(k); existing {
		av.SetIntVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueInt(k, v))
	}
}

// UpsertDouble performs the Insert or Update action. The double Value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertDouble(k string, v float64) {
	if av, existing := am.Get(k); existing {
		av.SetDoubleVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueDouble(k, v))
	}
}

// UpsertBool performs the Insert or Update action. The bool Value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertBool(k string, v bool) {
	if av, existing := am.Get(k); existing {
		av.SetBoolVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueBool(k, v))
	}
}

// Sort sorts the entries in the AttributeMap so two instances can be compared.
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, expected.Sort(), actual.Sort())
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
// it is possible that when iterating using "ForEach" to get access to fewer elements because nil elements are skipped.
func (am AttributeMap) Len() int {
	return len(*am.orig)
}

// ForEach iterates over the every elements in the map by calling the provided func.
//
// Example:
//
// it := sm.ForEach(func(k string, v AttributeValue) {
//   ...
// })
func (am AttributeMap) ForEach(f func(k string, v AttributeValue)) {
	for i := range *am.orig {
		kv := &(*am.orig)[i]
		f(kv.Key, AttributeValue{&kv.Value})
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
			AttributeValue{&akv.Value}.copyTo(&destAkv.Value)
		}
		return
	}

	// New slice is bigger than exist slice. Allocate new space.
	origs := make([]otlpcommon.KeyValue, len(*am.orig))
	for i := range *am.orig {
		akv := &(*am.orig)[i]
		origs[i].Key = akv.Key
		AttributeValue{&akv.Value}.copyTo(&origs[i].Value)
	}
	*dest.orig = origs
}

// StringMap stores a map of attribute keys to values.
type StringMap struct {
	orig *[]otlpcommon.StringKeyValue
}

// NewStringMap creates a StringMap with 0 elements.
func NewStringMap() StringMap {
	orig := []otlpcommon.StringKeyValue(nil)
	return StringMap{&orig}
}

func newStringMap(orig *[]otlpcommon.StringKeyValue) StringMap {
	return StringMap{orig}
}

// InitFromMap overwrites the entire StringMap and reconstructs the StringMap
// with values from the given map[string]string.
//
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{...}), actual)
func (sm StringMap) InitFromMap(attrMap map[string]string) StringMap {
	if len(attrMap) == 0 {
		*sm.orig = []otlpcommon.StringKeyValue(nil)
		return sm
	}
	origs := make([]otlpcommon.StringKeyValue, len(attrMap))
	ix := 0
	for k, v := range attrMap {
		origs[ix].Key = k
		origs[ix].Value = v
		ix++
	}
	*sm.orig = origs
	return sm
}

// InitEmptyWithCapacity constructs an empty StringMap with predefined slice capacity.
func (sm StringMap) InitEmptyWithCapacity(cap int) {
	if cap == 0 {
		*sm.orig = []otlpcommon.StringKeyValue(nil)
		return
	}
	*sm.orig = make([]otlpcommon.StringKeyValue, 0, cap)
}

// Get returns the StringValue associated with the key and true,
// otherwise an invalid instance of the StringKeyValue and false.
// Calling any functions on the returned invalid instance will cause a panic.
func (sm StringMap) Get(k string) (string, bool) {
	skv, found := sm.get(k)
	// GetValue handles the case where skv is nil.
	return skv.GetValue(), found
}

// Delete deletes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (sm StringMap) Delete(k string) bool {
	for i := range *sm.orig {
		skv := &(*sm.orig)[i]
		if skv.Key == k {
			(*sm.orig)[i] = (*sm.orig)[len(*sm.orig)-1]
			*sm.orig = (*sm.orig)[:len(*sm.orig)-1]
			return true
		}
	}
	return false
}

// Insert adds the string value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (sm StringMap) Insert(k, v string) {
	if _, existing := sm.Get(k); !existing {
		*sm.orig = append(*sm.orig, newStringKeyValue(k, v))
	}
}

// Update updates an existing string value with a value.
// No action is applied to the map where the key does not exist.
func (sm StringMap) Update(k, v string) {
	if skv, existing := sm.get(k); existing {
		skv.Value = v
	}
}

// Upsert performs the Insert or Update action. The string value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (sm StringMap) Upsert(k, v string) {
	if skv, existing := sm.get(k); existing {
		skv.Value = v
	} else {
		*sm.orig = append(*sm.orig, newStringKeyValue(k, v))
	}
}

// Len returns the length of this map.
//
// Because the AttributeMap is represented internally by a slice of pointers, and the data are comping from the wire,
// it is possible that when iterating using "ForEach" to get access to fewer elements because nil elements are skipped.
func (sm StringMap) Len() int {
	return len(*sm.orig)
}

// ForEach iterates over the every elements in the map by calling the provided func.
//
// Example:
//
// it := sm.ForEach(func(k string, v StringValue) {
//   ...
// })
func (sm StringMap) ForEach(f func(k string, v string)) {
	for i := range *sm.orig {
		skv := &(*sm.orig)[i]
		f(skv.Key, skv.Value)
	}
}

// CopyTo copies all elements from the current map to the dest.
func (sm StringMap) CopyTo(dest StringMap) {
	newLen := len(*sm.orig)
	oldCap := cap(*dest.orig)
	if newLen <= oldCap {
		*dest.orig = (*dest.orig)[:newLen:oldCap]
	} else {
		*dest.orig = make([]otlpcommon.StringKeyValue, newLen)
	}

	for i := range *sm.orig {
		skv := &(*sm.orig)[i]
		(*dest.orig)[i].Key = skv.Key
		(*dest.orig)[i].Value = skv.Value
	}
}

func (sm StringMap) get(k string) (*otlpcommon.StringKeyValue, bool) {
	for i := range *sm.orig {
		skv := &(*sm.orig)[i]
		if skv.Key == k {
			return skv, true
		}
	}
	return nil, false
}

// Sort sorts the entries in the StringMap so two instances can be compared.
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, expected.Sort(), actual.Sort())
func (sm StringMap) Sort() StringMap {
	sort.SliceStable(*sm.orig, func(i, j int) bool {
		// Intention is to move the nil values at the end.
		return (*sm.orig)[i].Key < (*sm.orig)[j].Key
	})
	return sm
}

func newStringKeyValue(k, v string) otlpcommon.StringKeyValue {
	return otlpcommon.StringKeyValue{Key: k, Value: v}
}
