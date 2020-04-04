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

var emptyAttributeKeyValue = &otlpcommon.AttributeKeyValue{}

// AttributeValue represents a value of an attribute. Typically used in an Attributes map.
// Must use one of NewAttributeValue* functions below to create new instances.
// Important: zero-initialized instance is not valid for use.
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
type AttributeValue struct {
	orig *otlpcommon.AttributeKeyValue
}

func NewAttributeValueString(v string) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_STRING, StringValue: v}}
}

func NewAttributeValueInt(v int64) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_INT, IntValue: v}}
}

func NewAttributeValueDouble(v float64) AttributeValue {
	return AttributeValue{orig: &otlpcommon.AttributeKeyValue{Type: otlpcommon.AttributeKeyValue_DOUBLE, DoubleValue: v}}
}

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

// All AttributeValue functions bellow must be called only on instances that are created
// via NewAttributeValue* functions. Calling these functions on zero-initialized
// AttributeValue struct will cause a panic.

func (a AttributeValue) Type() AttributeValueType {
	return AttributeValueType(a.orig.Type)
}

func (a AttributeValue) StringVal() string {
	return a.orig.StringValue
}

func (a AttributeValue) IntVal() int64 {
	return a.orig.IntValue
}

func (a AttributeValue) DoubleVal() float64 {
	return a.orig.DoubleValue
}

func (a AttributeValue) BoolVal() bool {
	return a.orig.BoolValue
}

func (a AttributeValue) SetStringVal(v string) {
	a.copyValues(emptyAttributeKeyValue)
	a.orig.Type = otlpcommon.AttributeKeyValue_STRING
	a.orig.StringValue = v
}

func (a AttributeValue) SetIntVal(v int64) {
	a.copyValues(emptyAttributeKeyValue)
	a.orig.Type = otlpcommon.AttributeKeyValue_INT
	a.orig.IntValue = v
}

func (a AttributeValue) SetDoubleVal(v float64) {
	a.copyValues(emptyAttributeKeyValue)
	a.orig.Type = otlpcommon.AttributeKeyValue_DOUBLE
	a.orig.DoubleValue = v
}

func (a AttributeValue) SetBoolVal(v bool) {
	a.copyValues(emptyAttributeKeyValue)
	a.orig.Type = otlpcommon.AttributeKeyValue_BOOL
	a.orig.BoolValue = v
}

func (a AttributeValue) SetValue(av AttributeValue) {
	a.copyValues(av.orig)
}

func (a AttributeValue) copyValues(akv *otlpcommon.AttributeKeyValue) {
	a.orig.Type = akv.Type
	a.orig.StringValue = akv.StringValue
	a.orig.IntValue = akv.IntValue
	a.orig.DoubleValue = akv.DoubleValue
	a.orig.BoolValue = akv.BoolValue
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
	akv.SetValue(av)
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
		AttributeValue{wrappers[ix]}.SetValue(v)
		ix++
	}

	*am.orig = wrappers

	return am
}

// Get returns the AttributeKeyValue associated with the key and true,
// otherwise an invalid instance of the AttributeKeyValue and false.
func (am AttributeMap) Get(key string) (AttributeValue, bool) {
	for _, a := range *am.orig {
		if a.Key == key {
			return AttributeValue{a}, true
		}
	}
	return AttributeValue{nil}, false
}

// Delete deletes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (am AttributeMap) Delete(key string) bool {
	for i, a := range *am.orig {
		if a.Key == key {
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
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Update(k string, v AttributeValue) {
	if av, existing := am.Get(k); existing {
		av.SetValue(v)
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
// Important: this function should not be used if the caller has access to
// the raw value to avoid an extra allocation.
func (am AttributeMap) Upsert(k string, v AttributeValue) {
	if av, existing := am.Get(k); existing {
		av.SetValue(v)
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
	sort.SliceStable(*am.orig, func(i, j int) bool { return (*am.orig)[i].Key < (*am.orig)[j].Key })
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

// StringKeyValue stores a key and value pair.
type StringKeyValue struct {
	orig *otlpcommon.StringKeyValue
}

// NewStringKeyValue creates a new StringKeyValue with the given key and value.
func NewStringKeyValue(k string, v string) StringKeyValue {
	return StringKeyValue{&otlpcommon.StringKeyValue{Key: k, Value: v}}
}

// Key returns the key associated with this StringKeyValue.
func (akv StringKeyValue) Key() string {
	return akv.orig.Key
}

// Value returns the value associated with this StringKeyValue.
func (akv StringKeyValue) Value() string {
	return akv.orig.Value
}

// SetValue replaces the value associated with this StringKeyValue.
func (akv StringKeyValue) SetValue(v string) {
	akv.orig.Value = v
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

// Get returns the StringKeyValue associated with the key and true,
// otherwise an invalid instance of the StringKeyValue and false.
func (sm StringMap) Get(k string) (StringKeyValue, bool) {
	for _, a := range *sm.orig {
		if a.Key == k {
			return StringKeyValue{a}, true
		}
	}
	return StringKeyValue{nil}, false
}

// Delete deletes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (sm StringMap) Delete(k string) bool {
	for i, a := range *sm.orig {
		if a.Key == k {
			(*sm.orig)[i] = (*sm.orig)[len(*sm.orig)-1]
			*sm.orig = (*sm.orig)[:len(*sm.orig)-1]
			return true
		}
	}
	return false
}

// Insert adds the StringKeyValue to the map when the key does not exist.
// No action is applied to the map where the key already exists.
func (sm StringMap) Insert(k, v string) {
	if _, existing := sm.Get(k); !existing {
		*sm.orig = append(*sm.orig, NewStringKeyValue(k, v).orig)
	}
}

// Update updates an existing StringKeyValue with a value.
// No action is applied to the map where the key does not exist.
func (sm StringMap) Update(k, v string) {
	if av, existing := sm.Get(k); existing {
		av.SetValue(v)
	}
}

// Upsert performs the Insert or Update action. The StringKeyValue is
// insert to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (sm StringMap) Upsert(k, v string) {
	if av, existing := sm.Get(k); existing {
		av.SetValue(v)
	} else {
		*sm.orig = append(*sm.orig, NewStringKeyValue(k, v).orig)
	}
}

// Len returns the number of StringKeyValue in the map.
func (sm StringMap) Len() int {
	return len(*sm.orig)
}

// GetStringKeyValue returns the StringKeyValue associated with the given index.
//
// This function is used mostly for iterating over all the values in the map:
// for i := 0; i < am.Len(); i++ {
//     akv := am.GetStringKeyValue(i)
//     ... // Do something with the attribute
// }
func (sm StringMap) GetStringKeyValue(ix int) StringKeyValue {
	return StringKeyValue{(*sm.orig)[ix]}
}

// Sort sorts the entries in the StringMap so two instances can be compared.
// Returns the same instance to allow nicer code like:
// assert.EqualValues(t, expected.Sort(), actual.Sort())
func (sm StringMap) Sort() StringMap {
	sort.SliceStable(*sm.orig, func(i, j int) bool { return (*sm.orig)[i].Key < (*sm.orig)[j].Key })
	return sm
}
