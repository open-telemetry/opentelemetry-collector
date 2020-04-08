// Copyright 2020 OpenTelemetry Authors
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

package data

// This file contains data structures that are common for all telemetry types,
// such as timestamps, attributes, etc.

import (
	"sort"

	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
)

// TimestampUnixNano is a time specified as UNIX Epoch time in nanoseconds since
// 00:00:00 UTC on 1 January 1970.
type TimestampUnixNano uint64

// AttributeValueType specifies the type of value. Numerically is equal to
// otlp.AttributeKeyValue_ValueType.
type AttributeValueType int32

const (
	AttributeValueSTRING = AttributeValueType(otlpcommon.AttributeKeyValue_STRING)
	AttributeValueINT    = AttributeValueType(otlpcommon.AttributeKeyValue_INT)
	AttributeValueDOUBLE = AttributeValueType(otlpcommon.AttributeKeyValue_DOUBLE)
	AttributeValueBOOL   = AttributeValueType(otlpcommon.AttributeKeyValue_BOOL)
)

// AttributeValue represents a value of an attribute. Typically used in AttributeMap.
// Must use one of NewAttributeValue* functions below to create new instances.
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
	orig *otlpcommon.AttributeKeyValue
}

// NewAttributeValueString creates a new AttributeValue with the given string value.
func NewAttributeValueString(v string) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_STRING, StringValue: v}}
}

// NewAttributeValueInt creates a new AttributeValue with the given int64 value.
func NewAttributeValueInt(v int64) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_INT, IntValue: v}}
}

// NewAttributeValueDouble creates a new AttributeValue with the given float64 value.
func NewAttributeValueDouble(v float64) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_DOUBLE, DoubleValue: v}}
}

// NewAttributeValueBool creates a new AttributeValue with the given bool value.
func NewAttributeValueBool(v bool) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_BOOL, BoolValue: v}}
}

// NewAttributeValueSlice creates a slice of attributes values that are correctly initialized.
func NewAttributeValueSlice(len int) []AttributeValue {
	// Allocate 2 slices, one for AttributeValues, another for underlying OTLP structs.
	// TODO: make one allocation for both slices.
	origs := make([]otlpcommon.AttributeKeyValue, len)
	wrappers := make([]AttributeValue, len)
	for i := range origs {
		wrappers[i].orig = &origs[i]
	}
	return wrappers
}

// Type returns the type of the value for this AttributeValue.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) Type() AttributeValueType {
	return AttributeValueType(a.orig.Type)
}

// Value returns the string value associated with this AttributeValue.
// If the Type() is not AttributeValueSTRING then return empty string.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) StringVal() string {
	return a.orig.StringValue
}

// Value returns the int64 value associated with this AttributeValue.
// If the Type() is not AttributeValueINT then return int64(0).
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) IntVal() int64 {
	return a.orig.IntValue
}

// Value returns the float64 value associated with this AttributeValue.
// If the Type() is not AttributeValueDOUBLE then return float64(0).
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) DoubleVal() float64 {
	return a.orig.DoubleValue
}

// Value returns the bool value associated with this AttributeValue.
// If the Type() is not AttributeValueBOOL then return false.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) BoolVal() bool {
	return a.orig.BoolValue
}

// SetStringVal replaces the string value associated with this AttributeValue,
// it also changes the type to be AttributeValueSTRING.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetStringVal(v string) {
	a.setTypeAndClear(otlpcommon.AttributeKeyValue_STRING)
	a.orig.StringValue = v
}

// SetIntVal replaces the int64 value associated with this AttributeValue,
// it also changes the type to be AttributeValueINT.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetIntVal(v int64) {
	a.setTypeAndClear(otlpcommon.AttributeKeyValue_INT)
	a.orig.IntValue = v
}

// SetDoubleVal replaces the float64 value associated with this AttributeValue,
// it also changes the type to be AttributeValueDOUBLE.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetDoubleVal(v float64) {
	a.setTypeAndClear(otlpcommon.AttributeKeyValue_DOUBLE)
	a.orig.DoubleValue = v
}

// SetBoolVal replaces the bool value associated with this AttributeValue,
// it also changes the type to be AttributeValueBOOL.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) SetBoolVal(v bool) {
	a.setTypeAndClear(otlpcommon.AttributeKeyValue_BOOL)
	a.orig.BoolValue = v
}

// CopyFrom copies all the fields from the given AttributeValue.
// Calling this function on zero-initialized AttributeValue will cause a panic.
func (a AttributeValue) CopyFrom(av AttributeValue) {
	akv := av.orig
	a.orig.Type = akv.Type
	a.orig.StringValue = akv.StringValue
	a.orig.IntValue = akv.IntValue
	a.orig.DoubleValue = akv.DoubleValue
	a.orig.BoolValue = akv.BoolValue
}

func (a AttributeValue) setTypeAndClear(ty otlpcommon.AttributeKeyValue_ValueType) {
	a.orig.Type = ty
	a.orig.StringValue = ""
	a.orig.IntValue = 0
	a.orig.DoubleValue = 0.0
	a.orig.BoolValue = false
}

func (a AttributeValue) Equal(av AttributeValue) bool {
	return a.orig.Type == av.orig.Type &&
		a.orig.StringValue == av.orig.StringValue &&
		a.orig.IntValue == av.orig.IntValue &&
		a.orig.DoubleValue == av.orig.DoubleValue &&
		a.orig.BoolValue == av.orig.BoolValue
}

func newAttributeKeyValueString(k string, v string) *otlpcommon.AttributeKeyValue {
	akv := AttributeValue{&otlpcommon.AttributeKeyValue{Key: k}}
	akv.SetStringVal(v)
	return akv.orig
}

func newAttributeKeyValueInt(k string, v int64) *otlpcommon.AttributeKeyValue {
	akv := AttributeValue{&otlpcommon.AttributeKeyValue{Key: k}}
	akv.SetIntVal(v)
	return akv.orig
}

func newAttributeKeyValueDouble(k string, v float64) *otlpcommon.AttributeKeyValue {
	akv := AttributeValue{&otlpcommon.AttributeKeyValue{Key: k}}
	akv.SetDoubleVal(v)
	return akv.orig
}

func newAttributeKeyValueBool(k string, v bool) *otlpcommon.AttributeKeyValue {
	akv := AttributeValue{&otlpcommon.AttributeKeyValue{Key: k}}
	akv.SetBoolVal(v)
	return akv.orig
}

func newAttributeKeyValue(k string, av AttributeValue) *otlpcommon.AttributeKeyValue {
	akv := AttributeValue{&otlpcommon.AttributeKeyValue{Key: k}}
	akv.CopyFrom(av)
	return akv.orig
}

// AttributeMap stores a map of attribute keys to values.
type AttributeMap struct {
	orig *[]*otlpcommon.AttributeKeyValue
}

// NewAttributeMap creates a AttributeMap with 0 elements.
func NewAttributeMap() AttributeMap {
	orig := []*otlpcommon.AttributeKeyValue(nil)
	return AttributeMap{&orig}
}

func newAttributeMap(orig *[]*otlpcommon.AttributeKeyValue) AttributeMap {
	return AttributeMap{orig}
}

// InitFromMap overwrites the entire AttributeMap and reconstructs the AttributeMap
// with values from the given map[string]string.
//
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, NewAttributeMap().InitFromMap(map[string]AttributeValue{...}), actual)
func (am AttributeMap) InitFromMap(attrMap map[string]AttributeValue) AttributeMap {
	if len(attrMap) == 0 {
		*am.orig = []*otlpcommon.AttributeKeyValue(nil)
		return am
	}
	origs := make([]otlpcommon.AttributeKeyValue, len(attrMap))
	wrappers := make([]*otlpcommon.AttributeKeyValue, len(attrMap))

	ix := 0
	for k, v := range attrMap {
		wrappers[ix] = &origs[ix]
		wrappers[ix].Key = k
		AttributeValue{wrappers[ix]}.CopyFrom(v)
		ix++
	}

	*am.orig = wrappers

	return am
}

// Get returns the AttributeKeyValue associated with the key and true,
// otherwise an invalid instance of the AttributeKeyValue and false.
// Calling any functions on the returned invalid instance will cause a panic.
func (am AttributeMap) Get(key string) (AttributeValue, bool) {
	for _, a := range *am.orig {
		if a != nil && a.Key == key {
			return AttributeValue{a}, true
		}
	}
	return AttributeValue{nil}, false
}

// Delete deletes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (am AttributeMap) Delete(key string) bool {
	for i, a := range *am.orig {
		if a != nil && a.Key == key {
			(*am.orig)[i] = (*am.orig)[len(*am.orig)-1]
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

// Insert adds the string Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertString(k string, v string) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueString(k, v))
	}
}

// Insert adds the int Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertInt(k string, v int64) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueInt(k, v))
	}
}

// Insert adds the double Value to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (am AttributeMap) InsertDouble(k string, v float64) {
	if _, existing := am.Get(k); !existing {
		*am.orig = append(*am.orig, newAttributeKeyValueDouble(k, v))
	}
}

// Insert adds the bool Value to the map when the key does not exist.
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
		av.CopyFrom(v)
	}
}

// Update updates an existing string Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateString(k string, v string) {
	if av, existing := am.Get(k); existing {
		av.SetStringVal(v)
	}
}

// Update updates an existing int Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateInt(k string, v int64) {
	if av, existing := am.Get(k); existing {
		av.SetIntVal(v)
	}
}

// Update updates an existing double Value with a value.
// No action is applied to the map where the key does not exist.
func (am AttributeMap) UpdateDouble(k string, v float64) {
	if av, existing := am.Get(k); existing {
		av.SetDoubleVal(v)
	}
}

// Update updates an existing bool Value with a value.
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
		av.CopyFrom(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValue(k, v))
	}
}

// Upsert performs the Insert or Update action. The AttributeValue is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertString(k string, v string) {
	if av, existing := am.Get(k); existing {
		av.SetStringVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueString(k, v))
	}
}

// Upsert performs the Insert or Update action. The int Value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertInt(k string, v int64) {
	if av, existing := am.Get(k); existing {
		av.SetIntVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueInt(k, v))
	}
}

// Upsert performs the Insert or Update action. The double Value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertDouble(k string, v float64) {
	if av, existing := am.Get(k); existing {
		av.SetDoubleVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueDouble(k, v))
	}
}

// Upsert performs the Insert or Update action. The bool Value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (am AttributeMap) UpsertBool(k string, v bool) {
	if av, existing := am.Get(k); existing {
		av.SetBoolVal(v)
	} else {
		*am.orig = append(*am.orig, newAttributeKeyValueBool(k, v))
	}
}

// Len returns the number of AttributeKeyValue in the map.
func (am AttributeMap) Len() int {
	return len(*am.orig)
}

// Sort sorts the entries in the AttributeMap so two instances can be compared.
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, expected.Sort(), actual.Sort())
func (am AttributeMap) Sort() AttributeMap {
	// Intention is to move the nil values at the end.
	sort.SliceStable(*am.orig, func(i, j int) bool {
		return ((*am.orig)[j] == nil) || ((*am.orig)[i] != nil && (*am.orig)[i].Key < (*am.orig)[j].Key)
	})
	return am
}

// GetAttribute returns the AttributeKeyValue associated with the given index.
//
// This function is used mostly for iterating over all the values in the map:
// for i := 1; i < am.Len(); i++ {
//     akv := am.GetAttribute(i)
//     ... // Do something with the attribute
// }
func (am AttributeMap) GetAttribute(ix int) (string, AttributeValue) {
	return (*am.orig)[ix].Key, AttributeValue{(*am.orig)[ix]}
}

// StringValue stores a string value.
//
// Intended to be passed by value since internally it is just a pointer to actual
// value representation. For the same reason passing by value and calling setters
// will modify the original, e.g.:
//
//   function f1(val StringValue) { val.SetValue("1234") }
//   function f2() {
//   	v := NewStringKeyValue("key", "a string")
//      f1(v)
//      _ := v.Value() // this will return "1234"
//   }
type StringValue struct {
	orig *otlpcommon.StringKeyValue
}

// Value returns the value associated with this StringValue.
// Calling this function on zero-initialized StringValue will cause a panic.
func (akv StringValue) Value() string {
	return akv.orig.Value
}

// SetValue replaces the value associated with this StringValue.
// Calling this function on zero-initialized StringValue will cause a panic.
func (akv StringValue) SetValue(v string) {
	akv.orig.Value = v
}

func newStringKeyValue(k, v string) *otlpcommon.StringKeyValue {
	return &otlpcommon.StringKeyValue{Key: k, Value: v}
}

// StringMap stores a map of attribute keys to values.
type StringMap struct {
	orig *[]*otlpcommon.StringKeyValue
}

// NewStringMap creates a StringMap with 0 elements.
func NewStringMap() StringMap {
	orig := []*otlpcommon.StringKeyValue(nil)
	return StringMap{&orig}
}

func newStringMap(orig *[]*otlpcommon.StringKeyValue) StringMap {
	return StringMap{orig}
}

// InitFromMap overwrites the entire StringMap and reconstructs the StringMap
// with values from the given map[string]string.
//
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, NewStringMap().InitFromMap(map[string]string{...}), actual)
func (sm StringMap) InitFromMap(attrMap map[string]string) StringMap {
	if len(attrMap) == 0 {
		*sm.orig = []*otlpcommon.StringKeyValue(nil)
		return sm
	}
	origs := make([]otlpcommon.StringKeyValue, len(attrMap))
	wrappers := make([]*otlpcommon.StringKeyValue, len(attrMap))

	ix := 0
	for k, v := range attrMap {
		wrappers[ix] = &origs[ix]
		wrappers[ix].Key = k
		wrappers[ix].Value = v
		ix++
	}
	*sm.orig = wrappers
	return sm
}

// Get returns the StringValue associated with the key and true,
// otherwise an invalid instance of the StringKeyValue and false.
// Calling any functions on the returned invalid instance will cause a panic.
func (sm StringMap) Get(k string) (StringValue, bool) {
	for _, a := range *sm.orig {
		if a != nil && a.Key == k {
			return StringValue{a}, true
		}
	}
	return StringValue{nil}, false
}

// Delete deletes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (sm StringMap) Delete(k string) bool {
	for i, a := range *sm.orig {
		if a != nil && a.Key == k {
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
	if av, existing := sm.Get(k); existing {
		av.SetValue(v)
	}
}

// Upsert performs the Insert or Update action. The string value is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (sm StringMap) Upsert(k, v string) {
	if av, existing := sm.Get(k); existing {
		av.SetValue(v)
	} else {
		*sm.orig = append(*sm.orig, newStringKeyValue(k, v))
	}
}

// Len returns the number of StringValue in the map.
func (sm StringMap) Len() int {
	return len(*sm.orig)
}

// GetStringKeyValue returns the StringValue associated with the given index.
//
// This function is used mostly for iterating over all the values in the map:
// for i := 0; i < am.Len(); i++ {
//     k, v := am.GetStringKeyValue(i)
//     ... // Do something with the attribute
// }
func (sm StringMap) GetStringKeyValue(ix int) (string, StringValue) {
	return (*sm.orig)[ix].Key, StringValue{(*sm.orig)[ix]}
}

// Sort sorts the entries in the StringMap so two instances can be compared.
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, expected.Sort(), actual.Sort())
func (sm StringMap) Sort() StringMap {
	sort.SliceStable(*sm.orig, func(i, j int) bool {
		// Intention is to move the nil values at the end.
		return ((*sm.orig)[j] == nil) || ((*sm.orig)[i] != nil && (*sm.orig)[i].Key < (*sm.orig)[j].Key)
	})
	return sm
}
