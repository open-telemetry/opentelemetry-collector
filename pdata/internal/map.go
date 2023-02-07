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
	"go.uber.org/multierr"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

type commonMap struct {
	orig *[]otlpcommon.KeyValue
}

type Map struct {
	commonMap
}

type MutableMap struct {
	commonMap
	preventConversion struct{} // nolint:unused
}

func (m Map) asMutable() MutableMap {
	return MutableMap{commonMap: commonMap{orig: m.orig}}
}

func (m MutableMap) AsImmutable() Map {
	return Map{commonMap{orig: m.orig}}
}

func NewMapFromOrig(orig *[]otlpcommon.KeyValue) Map {
	return Map{commonMap{orig}}
}

func NewMutableMapFromOrig(orig *[]otlpcommon.KeyValue) MutableMap {
	return MutableMap{commonMap: commonMap{orig}}
}

// NewMap creates a Map with 0 elements.
func NewMutableMap() MutableMap {
	orig := []otlpcommon.KeyValue(nil)
	return NewMutableMapFromOrig(&orig)
}

func GenerateTestMap() MutableMap {
	var orig []otlpcommon.KeyValue
	ms := NewMutableMapFromOrig(&orig)
	FillTestMap(ms)
	return ms
}

func FillTestMap(dest MutableMap) {
	*dest.orig = nil
	*dest.orig = append(*dest.orig, otlpcommon.KeyValue{Key: "k",
		Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v"}}})
}

// Clear erases any existing entries in this Map instance.
func (m MutableMap) Clear() {
	*m.orig = nil
}

// EnsureCapacity increases the capacity of this Map instance, if necessary,
// to ensure that it can hold at least the number of elements specified by the capacity argument.
func (m MutableMap) EnsureCapacity(capacity int) {
	if capacity <= cap(*m.orig) {
		return
	}
	oldOrig := *m.orig
	*m.orig = make([]otlpcommon.KeyValue, 0, capacity)
	copy(*m.orig, oldOrig)
}

// Get returns the Value associated with the key and true. Returned
// Value is not a copy, it is a reference to the value stored in this map.
// It is allowed to modify the returned value using Value.Set* functions.
// Such modification will be applied to the value stored in this map.
//
// If the key does not exist returns an invalid instance of the KeyValue and false.
// Calling any functions on the returned invalid instance will cause a panic.
func (m Map) Get(key string) (Value, bool) {
	for i := range *m.orig {
		akv := &(*m.orig)[i]
		if akv.Key == key {
			return NewValueFromOrig(&akv.Value), true
		}
	}
	return NewValueFromOrig(nil), false
}

func (m MutableMap) Get(key string) (MutableValue, bool) {
	v, ok := m.AsImmutable().Get(key)
	return v.asMutable(), ok
}

// Remove removes the entry associated with the key and returns true if the key
// was present in the map, otherwise returns false.
func (m MutableMap) Remove(key string) bool {
	for i := range *m.orig {
		akv := &(*m.orig)[i]
		if akv.Key == key {
			*akv = (*m.orig)[len(*m.orig)-1]
			*m.orig = (*m.orig)[:len(*m.orig)-1]
			return true
		}
	}
	return false
}

// RemoveIf removes the entries for which the function in question returns true
func (m MutableMap) RemoveIf(f func(string, MutableValue) bool) {
	newLen := 0
	for i := 0; i < len(*m.orig); i++ {
		akv := &(*m.orig)[i]
		if f(akv.Key, NewMutableValueFromOrig(&akv.Value)) {
			continue
		}
		if newLen == i {
			// Nothing to move, element is at the right place.
			newLen++
			continue
		}
		(*m.orig)[newLen] = (*m.orig)[i]
		newLen++
	}
	*m.orig = (*m.orig)[:newLen]
}

// PutEmpty inserts or updates an empty value to the map under given key
// and return the updated/inserted value.
func (m MutableMap) PutEmpty(k string) MutableValue {
	if av, existing := m.Get(k); existing {
		av.orig.Value = nil
		return NewMutableValueFromOrig(av.orig)
	}
	*m.orig = append(*m.orig, otlpcommon.KeyValue{Key: k})
	return NewMutableValueFromOrig(&(*m.orig)[len(*m.orig)-1].Value)
}

// PutStr performs the Insert or Update action. The Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (m MutableMap) PutStr(k string, v string) {
	if av, existing := m.Get(k); existing {
		av.SetStr(v)
	} else {
		*m.orig = append(*m.orig, newKeyValueString(k, v))
	}
}

// PutInt performs the Insert or Update action. The int Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (m MutableMap) PutInt(k string, v int64) {
	if av, existing := m.Get(k); existing {
		av.SetInt(v)
	} else {
		*m.orig = append(*m.orig, newKeyValueInt(k, v))
	}
}

// PutDouble performs the Insert or Update action. The double Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (m MutableMap) PutDouble(k string, v float64) {
	if av, existing := m.Get(k); existing {
		av.SetDouble(v)
	} else {
		*m.orig = append(*m.orig, newKeyValueDouble(k, v))
	}
}

// PutBool performs the Insert or Update action. The bool Value is
// inserted to the map that did not originally have the key. The key/value is
// updated to the map where the key already existed.
func (m MutableMap) PutBool(k string, v bool) {
	if av, existing := m.Get(k); existing {
		av.SetBool(v)
	} else {
		*m.orig = append(*m.orig, newKeyValueBool(k, v))
	}
}

// PutEmptyBytes inserts or updates an empty byte slice under given key and returns it.
func (m MutableMap) PutEmptyBytes(k string) MutableByteSlice {
	bv := otlpcommon.AnyValue_BytesValue{}
	if av, existing := m.Get(k); existing {
		av.orig.Value = &bv
	} else {
		*m.orig = append(*m.orig, otlpcommon.KeyValue{Key: k, Value: otlpcommon.AnyValue{Value: &bv}})
	}
	return NewMutableByteSliceFromOrig(&bv.BytesValue)
}

// PutEmptyMap inserts or updates an empty map under given key and returns it.
func (m MutableMap) PutEmptyMap(k string) MutableMap {
	kvl := otlpcommon.AnyValue_KvlistValue{KvlistValue: &otlpcommon.KeyValueList{Values: []otlpcommon.KeyValue(nil)}}
	if av, existing := m.Get(k); existing {
		av.orig.Value = &kvl
	} else {
		*m.orig = append(*m.orig, otlpcommon.KeyValue{Key: k, Value: otlpcommon.AnyValue{Value: &kvl}})
	}
	return NewMutableMapFromOrig(&kvl.KvlistValue.Values)
}

// PutEmptySlice inserts or updates an empty slice under given key and returns it.
func (m MutableMap) PutEmptySlice(k string) MutableSlice {
	vl := otlpcommon.AnyValue_ArrayValue{ArrayValue: &otlpcommon.ArrayValue{Values: []otlpcommon.AnyValue(nil)}}
	if av, existing := m.Get(k); existing {
		av.orig.Value = &vl
	} else {
		*m.orig = append(*m.orig, otlpcommon.KeyValue{Key: k, Value: otlpcommon.AnyValue{Value: &vl}})
	}
	return NewMutableSliceFromOrig(&vl.ArrayValue.Values)
}

// Len returns the length of this map.
//
// Because the Map is represented internally by a slice of pointers, and the data are comping from the wire,
// it is possible that when iterating using "Range" to get access to fewer elements because nil elements are skipped.
func (m commonMap) Len() int {
	return len(*m.orig)
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
//
// Example:
//
//	sm.Range(func(k string, v Value) bool {
//	    ...
//	})
func (m Map) Range(f func(k string, v Value) bool) {
	for i := range *m.orig {
		kv := &(*m.orig)[i]
		if !f(kv.Key, NewValueFromOrig(&kv.Value)) {
			break
		}
	}
}

func (m MutableMap) Range(f func(k string, v MutableValue) bool) {
	m.AsImmutable().Range(func(k string, v Value) bool {
		return f(k, v.asMutable())
	})
}

// CopyTo copies all elements from the current map overriding the destination.
func (m commonMap) CopyTo(dest MutableMap) {
	newLen := len(*m.orig)
	oldCap := cap(*dest.orig)
	if newLen <= oldCap {
		// New slice fits in existing slice, no need to reallocate.
		*dest.orig = (*dest.orig)[:newLen:oldCap]
		for i := range *m.orig {
			akv := &(*m.orig)[i]
			destAkv := &(*dest.orig)[i]
			destAkv.Key = akv.Key
			NewValueFromOrig(&akv.Value).CopyTo(NewMutableValueFromOrig(&destAkv.Value))
		}
		return
	}

	// New slice is bigger than exist slice. Allocate new space.
	origs := make([]otlpcommon.KeyValue, len(*m.orig))
	for i := range *m.orig {
		akv := &(*m.orig)[i]
		origs[i].Key = akv.Key
		NewValueFromOrig(&akv.Value).CopyTo(NewMutableValueFromOrig(&origs[i].Value))
	}
	*dest.orig = origs
}

// AsRaw returns a standard go map representation of this Map.
func (m commonMap) AsRaw() map[string]any {
	rawMap := make(map[string]any)
	Map{m}.Range(func(k string, v Value) bool {
		rawMap[k] = v.AsRaw()
		return true
	})
	return rawMap
}

// FromRaw overrides this Map instance from a standard go map.
func (m MutableMap) FromRaw(rawMap map[string]any) error {
	if len(rawMap) == 0 {
		*m.orig = nil
		return nil
	}

	var errs error
	origs := make([]otlpcommon.KeyValue, len(rawMap))
	ix := 0
	for k, iv := range rawMap {
		origs[ix].Key = k
		errs = multierr.Append(errs, NewMutableValueFromOrig(&origs[ix].Value).FromRaw(iv))
		ix++
	}
	*m.orig = origs
	return errs
}
