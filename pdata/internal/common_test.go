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

package internal

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestAttributeValue(t *testing.T) {
	v := NewValueString("abc")
	assert.EqualValues(t, ValueTypeString, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v = NewValueInt(123)
	assert.EqualValues(t, ValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v = NewValueDouble(3.4)
	assert.EqualValues(t, ValueTypeDouble, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v = NewValueBool(true)
	assert.EqualValues(t, ValueTypeBool, v.Type())
	assert.True(t, v.BoolVal())

	v = NewValueEmpty()
	assert.EqualValues(t, ValueTypeEmpty, v.Type())

	v.SetStringVal("abc")
	assert.EqualValues(t, ValueTypeString, v.Type())
	assert.EqualValues(t, "abc", v.StringVal())

	v.SetIntVal(123)
	assert.EqualValues(t, ValueTypeInt, v.Type())
	assert.EqualValues(t, 123, v.IntVal())

	v.SetDoubleVal(3.4)
	assert.EqualValues(t, ValueTypeDouble, v.Type())
	assert.EqualValues(t, 3.4, v.DoubleVal())

	v.SetBoolVal(true)
	assert.EqualValues(t, ValueTypeBool, v.Type())
	assert.True(t, v.BoolVal())

	bytesValue := NewImmutableByteSlice([]byte{1, 2, 3, 4})
	v = NewValueBytes(bytesValue)
	assert.EqualValues(t, ValueTypeBytes, v.Type())
	assert.EqualValues(t, bytesValue, v.BytesVal())
}

func TestAttributeValueType(t *testing.T) {
	assert.EqualValues(t, "EMPTY", ValueTypeEmpty.String())
	assert.EqualValues(t, "STRING", ValueTypeString.String())
	assert.EqualValues(t, "BOOL", ValueTypeBool.String())
	assert.EqualValues(t, "INT", ValueTypeInt.String())
	assert.EqualValues(t, "DOUBLE", ValueTypeDouble.String())
	assert.EqualValues(t, "MAP", ValueTypeMap.String())
	assert.EqualValues(t, "SLICE", ValueTypeSlice.String())
	assert.EqualValues(t, "BYTES", ValueTypeBytes.String())
}

func TestAttributeValueMap(t *testing.T) {
	m1 := NewValueMap()
	assert.Equal(t, ValueTypeMap, m1.Type())
	assert.Equal(t, NewMap(), m1.MapVal())
	assert.Equal(t, 0, m1.MapVal().Len())

	m1.MapVal().InsertDouble("double_key", 123)
	assert.Equal(t, 1, m1.MapVal().Len())
	got, exists := m1.MapVal().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)

	// Create a second map.
	m2 := NewValueMap()
	assert.Equal(t, 0, m2.MapVal().Len())

	// Modify the source map that was inserted.
	m2.MapVal().UpsertString("key_in_child", "somestr")
	assert.Equal(t, 1, m2.MapVal().Len())
	got, exists = m2.MapVal().Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewValueString("somestr"), got)

	// Insert the second map as a child. This should perform a deep copy.
	m1.MapVal().Insert("child_map", m2)
	assert.EqualValues(t, 2, m1.MapVal().Len())
	got, exists = m1.MapVal().Get("double_key")
	assert.True(t, exists)
	assert.Equal(t, NewValueDouble(123), got)
	got, exists = m1.MapVal().Get("child_map")
	assert.True(t, exists)
	assert.Equal(t, m2, got)

	// Modify the source map m2 that was inserted into m1.
	m2.MapVal().UpdateString("key_in_child", "somestr2")
	assert.EqualValues(t, 1, m2.MapVal().Len())
	got, exists = m2.MapVal().Get("key_in_child")
	assert.True(t, exists)
	assert.Equal(t, NewValueString("somestr2"), got)

	// The child map inside m1 should not be modified.
	childMap, childMapExists := m1.MapVal().Get("child_map")
	require.True(t, childMapExists)
	got, exists = childMap.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueString("somestr"), got)

	// Now modify the inserted map (not the source)
	childMap.MapVal().UpdateString("key_in_child", "somestr3")
	assert.EqualValues(t, 1, childMap.MapVal().Len())
	got, exists = childMap.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueString("somestr3"), got)

	// The source child map should not be modified.
	got, exists = m2.MapVal().Get("key_in_child")
	require.True(t, exists)
	assert.Equal(t, NewValueString("somestr2"), got)

	removed := m1.MapVal().Remove("double_key")
	assert.True(t, removed)
	assert.EqualValues(t, 1, m1.MapVal().Len())
	_, exists = m1.MapVal().Get("double_key")
	assert.False(t, exists)

	removed = m1.MapVal().Remove("child_map")
	assert.True(t, removed)
	assert.EqualValues(t, 0, m1.MapVal().Len())
	_, exists = m1.MapVal().Get("child_map")
	assert.False(t, exists)

	// Test nil KvlistValue case for MapVal() func.
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	m1 = Value{orig: orig}
	assert.EqualValues(t, Map{}, m1.MapVal())
}

func TestNilOrigSetAttributeValue(t *testing.T) {
	av := NewValueEmpty()
	av.SetStringVal("abc")
	assert.EqualValues(t, "abc", av.StringVal())

	av = NewValueEmpty()
	av.SetIntVal(123)
	assert.EqualValues(t, 123, av.IntVal())

	av = NewValueEmpty()
	av.SetBoolVal(true)
	assert.True(t, av.BoolVal())

	av = NewValueEmpty()
	av.SetDoubleVal(1.23)
	assert.EqualValues(t, 1.23, av.DoubleVal())

	av = NewValueEmpty()
	av.SetBytesVal(NewImmutableByteSlice([]byte{1, 2, 3}))
	assert.Equal(t, []byte{1, 2, 3}, av.BytesVal().AsRaw())
}

func TestAttributeValueEqual(t *testing.T) {
	av1 := NewValueEmpty()
	av2 := NewValueEmpty()
	assert.True(t, av1.Equal(av2))

	av2 = NewValueString("abc")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueString("abc")
	assert.True(t, av1.Equal(av2))

	av2 = NewValueString("edf")
	assert.False(t, av1.Equal(av2))

	av2 = NewValueInt(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueInt(234)
	assert.False(t, av1.Equal(av2))

	av1 = NewValueInt(123)
	assert.True(t, av1.Equal(av2))

	av2 = NewValueDouble(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueDouble(234)
	assert.False(t, av1.Equal(av2))

	av1 = NewValueDouble(123)
	assert.True(t, av1.Equal(av2))

	av2 = NewValueBool(false)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueBool(true)
	assert.False(t, av1.Equal(av2))

	av1 = NewValueBool(false)
	assert.True(t, av1.Equal(av2))

	av2 = NewValueBytes(NewImmutableByteSlice([]byte{1, 2, 3}))
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av1 = NewValueBytes(NewImmutableByteSlice([]byte{1, 2, 4}))
	assert.False(t, av1.Equal(av2))

	av1 = NewValueBytes(NewImmutableByteSlice([]byte{1, 2, 3}))
	assert.True(t, av1.Equal(av2))

	av1 = NewValueSlice()
	av1.SliceVal().AppendEmpty().SetIntVal(123)
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewValueSlice()
	av2.SliceVal().AppendEmpty().SetDoubleVal(123)
	assert.False(t, av1.Equal(av2))

	NewValueInt(123).CopyTo(av2.SliceVal().At(0))
	assert.True(t, av1.Equal(av2))

	av1.CopyTo(av2.SliceVal().AppendEmpty())
	assert.False(t, av1.Equal(av2))

	av1 = NewValueMap()
	av1.MapVal().UpsertString("foo", "bar")
	assert.False(t, av1.Equal(av2))
	assert.False(t, av2.Equal(av1))

	av2 = NewValueMap()
	av2.MapVal().UpsertString("foo", "bar")
	assert.True(t, av1.Equal(av2))

	fooVal, ok := av2.MapVal().Get("foo")
	if !ok {
		assert.Fail(t, "expected to find value with key foo")
	}
	fooVal.SetStringVal("not-bar")
	assert.False(t, av1.Equal(av2))
}

func TestNilMap(t *testing.T) {
	assert.EqualValues(t, 0, NewMap().Len())

	val, exist := NewMap().Get("test_key")
	assert.False(t, exist)
	assert.EqualValues(t, Value{nil}, val)

	insertMap := NewMap()
	insertMap.Insert("k", NewValueString("v"))
	assert.EqualValues(t, generateTestMap(), insertMap)

	insertMapString := NewMap()
	insertMapString.InsertString("k", "v")
	assert.EqualValues(t, generateTestMap(), insertMapString)

	insertMapNull := NewMap()
	insertMapNull.InsertNull("k")
	assert.EqualValues(t, generateTestEmptyMap(), insertMapNull)

	insertMapInt := NewMap()
	insertMapInt.InsertInt("k", 123)
	assert.EqualValues(t, generateTestIntMap(), insertMapInt)

	insertMapDouble := NewMap()
	insertMapDouble.InsertDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleMap(), insertMapDouble)

	insertMapBool := NewMap()
	insertMapBool.InsertBool("k", true)
	assert.EqualValues(t, generateTestBoolMap(), insertMapBool)

	insertMapBytes := NewMap()
	insertMapBytes.InsertBytes("k", NewImmutableByteSlice([]byte{1, 2, 3, 4, 5}))
	assert.EqualValues(t, generateTestBytesMap(), insertMapBytes)

	updateMap := NewMap()
	updateMap.Update("k", NewValueString("v"))
	assert.EqualValues(t, NewMap(), updateMap)

	updateMapString := NewMap()
	updateMapString.UpdateString("k", "v")
	assert.EqualValues(t, NewMap(), updateMapString)

	updateMapInt := NewMap()
	updateMapInt.UpdateInt("k", 123)
	assert.EqualValues(t, NewMap(), updateMapInt)

	updateMapDouble := NewMap()
	updateMapDouble.UpdateDouble("k", 12.3)
	assert.EqualValues(t, NewMap(), updateMapDouble)

	updateMapBool := NewMap()
	updateMapBool.UpdateBool("k", true)
	assert.EqualValues(t, NewMap(), updateMapBool)

	updateMapBytes := NewMap()
	updateMapBytes.UpdateBytes("k", NewImmutableByteSlice([]byte{1, 2, 3}))
	assert.EqualValues(t, NewMap(), updateMapBytes)

	upsertMap := NewMap()
	upsertMap.Upsert("k", NewValueString("v"))
	assert.EqualValues(t, generateTestMap(), upsertMap)

	upsertMapString := NewMap()
	upsertMapString.UpsertString("k", "v")
	assert.EqualValues(t, generateTestMap(), upsertMapString)

	upsertMapInt := NewMap()
	upsertMapInt.UpsertInt("k", 123)
	assert.EqualValues(t, generateTestIntMap(), upsertMapInt)

	upsertMapDouble := NewMap()
	upsertMapDouble.UpsertDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleMap(), upsertMapDouble)

	upsertMapBool := NewMap()
	upsertMapBool.UpsertBool("k", true)
	assert.EqualValues(t, generateTestBoolMap(), upsertMapBool)

	upsertMapBytes := NewMap()
	upsertMapBytes.UpsertBytes("k", NewImmutableByteSlice([]byte{1, 2, 3, 4, 5}))
	assert.EqualValues(t, generateTestBytesMap(), upsertMapBytes)

	removeMap := NewMap()
	assert.False(t, removeMap.Remove("k"))
	assert.EqualValues(t, NewMap(), removeMap)

	// Test Sort
	assert.EqualValues(t, NewMap(), NewMap().Sort())
}

func TestMapWithEmpty(t *testing.T) {
	origWithNil := []otlpcommon.KeyValue{
		{},
		{
			Key:   "test_key",
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
		},
		{
			Key:   "test_key2",
			Value: otlpcommon.AnyValue{Value: nil},
		},
	}
	sm := Map{
		orig: &origWithNil,
	}
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	sm.Insert("other_key", NewValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.InsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.IntVal())

	sm.InsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.InsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.True(t, val.BoolVal())

	sm.InsertBytes("other_key_bytes", NewImmutableByteSlice([]byte{1, 2, 3}))
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1, 2, 3}, val.BytesVal().AsRaw())

	sm.Update("other_key", NewValueString("yet_another_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateString("other_key_string", "yet_another_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpdateInt("other_key_int", 456)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpdateDouble("other_key_double", 4.56)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpdateBool("other_key_bool", false)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.False(t, val.BoolVal())

	sm.UpdateBytes("other_key_bytes", NewImmutableByteSlice([]byte{4, 5, 6}))
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{4, 5, 6}, val.BytesVal().AsRaw())

	sm.Upsert("other_key", NewValueString("other_value"))
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.UpsertString("other_key_string", "other_value")
	val, exist = sm.Get("other_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())

	sm.UpsertInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.IntVal())

	sm.UpsertDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.DoubleVal())

	sm.UpsertBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.True(t, val.BoolVal())

	sm.UpsertBytes("other_key_bytes", NewImmutableByteSlice([]byte{7, 8, 9}))
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{7, 8, 9}, val.BytesVal().AsRaw())

	sm.Upsert("yet_another_key", NewValueString("yet_another_value"))
	val, exist = sm.Get("yet_another_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpsertString("yet_another_key_string", "yet_another_value")
	val, exist = sm.Get("yet_another_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "yet_another_value", val.StringVal())

	sm.UpsertInt("yet_another_key_int", 456)
	val, exist = sm.Get("yet_another_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.IntVal())

	sm.UpsertDouble("yet_another_key_double", 4.56)
	val, exist = sm.Get("yet_another_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.DoubleVal())

	sm.UpsertBool("yet_another_key_bool", false)
	val, exist = sm.Get("yet_another_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.False(t, val.BoolVal())

	sm.UpsertBytes("yet_another_key_bytes", NewImmutableByteSlice([]byte{1}))
	val, exist = sm.Get("yet_another_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1}, val.BytesVal().AsRaw())

	assert.True(t, sm.Remove("other_key"))
	assert.True(t, sm.Remove("other_key_string"))
	assert.True(t, sm.Remove("other_key_int"))
	assert.True(t, sm.Remove("other_key_double"))
	assert.True(t, sm.Remove("other_key_bool"))
	assert.True(t, sm.Remove("other_key_bytes"))
	assert.True(t, sm.Remove("yet_another_key"))
	assert.True(t, sm.Remove("yet_another_key_string"))
	assert.True(t, sm.Remove("yet_another_key_int"))
	assert.True(t, sm.Remove("yet_another_key_double"))
	assert.True(t, sm.Remove("yet_another_key_bool"))
	assert.True(t, sm.Remove("yet_another_key_bytes"))
	assert.False(t, sm.Remove("other_key"))
	assert.False(t, sm.Remove("yet_another_key"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	_, exist = sm.Get("test_key3")
	assert.False(t, exist)

	// Test Sort
	assert.EqualValues(t, Map{orig: &origWithNil}, sm.Sort())
}

func TestMapIterationNil(t *testing.T) {
	NewMap().Range(func(k string, v Value) bool {
		// Fail if any element is returned
		t.Fail()
		return true
	})
}

func TestMap_Range(t *testing.T) {
	rawMap := map[string]interface{}{
		"k_string": "123",
		"k_int":    int64(123),
		"k_double": float64(1.23),
		"k_bool":   true,
		"k_empty":  nil,
	}
	am := NewMapFromRaw(rawMap)
	assert.Equal(t, 5, am.Len())

	calls := 0
	am.Range(func(k string, v Value) bool {
		calls++
		return false
	})
	assert.Equal(t, 1, calls)

	am.Range(func(k string, v Value) bool {
		assert.Equal(t, rawMap[k], v.asRaw())
		delete(rawMap, k)
		return true
	})
	assert.EqualValues(t, 0, len(rawMap))
}

func TestMap_InitFromRaw(t *testing.T) {
	am := NewMapFromRaw(map[string]interface{}(nil))
	assert.EqualValues(t, NewMap(), am)

	rawMap := map[string]interface{}{
		"k_string": "123",
		"k_int":    123,
		"k_double": 1.23,
		"k_bool":   true,
		"k_null":   nil,
		"k_bytes":  []byte{1, 2, 3},
	}
	rawOrig := []otlpcommon.KeyValue{
		newAttributeKeyValueString("k_string", "123"),
		newAttributeKeyValueInt("k_int", 123),
		newAttributeKeyValueDouble("k_double", 1.23),
		newAttributeKeyValueBool("k_bool", true),
		newAttributeKeyValueNull("k_null"),
		newAttributeKeyValueBytes("k_bytes", NewImmutableByteSlice([]byte{1, 2, 3})),
	}
	am = NewMapFromRaw(rawMap)
	assert.EqualValues(t, Map{orig: &rawOrig}.Sort(), am.Sort())
}

func TestAttributeValue_CopyTo(t *testing.T) {
	// Test nil KvlistValue case for MapVal() func.
	dest := NewValueEmpty()
	orig := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_KvlistValue{KvlistValue: nil}}
	Value{orig: orig}.CopyTo(dest)
	assert.Nil(t, dest.orig.Value.(*otlpcommon.AnyValue_KvlistValue).KvlistValue)

	// Test nil ArrayValue case for SliceVal() func.
	dest = NewValueEmpty()
	orig = &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}}
	Value{orig: orig}.CopyTo(dest)
	assert.Nil(t, dest.orig.Value.(*otlpcommon.AnyValue_ArrayValue).ArrayValue)

	// Test copy empty value.
	Value{orig: &otlpcommon.AnyValue{}}.CopyTo(dest)
	assert.Nil(t, dest.orig.Value)
}

func TestMap_CopyTo(t *testing.T) {
	dest := NewMap()
	// Test CopyTo to empty
	NewMap().CopyTo(dest)
	assert.EqualValues(t, 0, dest.Len())

	// Test CopyTo larger slice
	generateTestMap().CopyTo(dest)
	assert.EqualValues(t, generateTestMap(), dest)

	// Test CopyTo same size slice
	generateTestMap().CopyTo(dest)
	assert.EqualValues(t, generateTestMap(), dest)

	// Test CopyTo with an empty Value in the destination
	(*dest.orig)[0].Value = otlpcommon.AnyValue{}
	generateTestMap().CopyTo(dest)
	assert.EqualValues(t, generateTestMap(), dest)
}

func TestAttributeValue_copyTo(t *testing.T) {
	av := NewValueEmpty()
	destVal := otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{}}
	av.copyTo(&destVal)
	assert.EqualValues(t, nil, destVal.Value)
}

func TestValueBytes_CopyTo_Deprecated(t *testing.T) {
	orig := NewValueMBytes([]byte{1, 2, 3})
	dest := NewValueEmpty()
	orig.CopyTo(dest)
	assert.Equal(t, orig, dest)

	orig.MBytesVal()[0] = 10
	assert.NotEqual(t, orig, dest)
	assert.Equal(t, []byte{1, 2, 3}, dest.MBytesVal())
	assert.Equal(t, []byte{10, 2, 3}, orig.MBytesVal())
}

func TestMap_Update(t *testing.T) {
	origWithNil := []otlpcommon.KeyValue{
		{
			Key:   "test_key",
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
		},
		{
			Key:   "test_key2",
			Value: otlpcommon.AnyValue{Value: nil},
		},
	}
	sm := Map{
		orig: &origWithNil,
	}

	av, exists := sm.Get("test_key")
	assert.True(t, exists)
	assert.EqualValues(t, ValueTypeString, av.Type())
	assert.EqualValues(t, "test_value", av.StringVal())
	av.SetIntVal(123)

	av2, exists := sm.Get("test_key")
	assert.True(t, exists)
	assert.EqualValues(t, ValueTypeInt, av2.Type())
	assert.EqualValues(t, 123, av2.IntVal())

	av, exists = sm.Get("test_key2")
	assert.True(t, exists)
	assert.EqualValues(t, ValueTypeEmpty, av.Type())
	assert.EqualValues(t, "", av.StringVal())
	av.SetIntVal(123)

	av2, exists = sm.Get("test_key2")
	assert.True(t, exists)
	assert.EqualValues(t, ValueTypeInt, av2.Type())
	assert.EqualValues(t, 123, av2.IntVal())
}

func TestMap_EnsureCapacity_Zero(t *testing.T) {
	am := NewMap()
	am.EnsureCapacity(0)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 0, cap(*am.orig))
}

func TestMap_EnsureCapacity(t *testing.T) {
	am := NewMap()
	am.EnsureCapacity(5)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.orig))
	am.EnsureCapacity(3)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.orig))
	am.EnsureCapacity(8)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 8, cap(*am.orig))
}

func TestMap_Clear(t *testing.T) {
	am := NewMap()
	assert.Nil(t, *am.orig)
	am.Clear()
	assert.Nil(t, *am.orig)
	am.EnsureCapacity(5)
	assert.NotNil(t, *am.orig)
	am.Clear()
	assert.Nil(t, *am.orig)
}

func TestMap_RemoveIf(t *testing.T) {
	rawMap := map[string]interface{}{
		"k_string": "123",
		"k_int":    int64(123),
		"k_double": float64(1.23),
		"k_bool":   true,
		"k_empty":  nil,
		"k_bytes":  []byte{},
	}
	am := NewMapFromRaw(rawMap)
	assert.Equal(t, 6, am.Len())

	am.RemoveIf(func(key string, val Value) bool {
		return key == "k_int" || val.Type() == ValueTypeBool
	})
	assert.Equal(t, 4, am.Len())
	_, exists := am.Get("k_string")
	assert.True(t, exists)
	_, exists = am.Get("k_bool")
	assert.False(t, exists)
	_, exists = am.Get("k_int")
	assert.False(t, exists)
}

func BenchmarkAttributeValue_CopyTo(b *testing.B) {
	av := NewValueString("k")
	c := NewValueInt(123)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.copyTo(av.orig)
	}
	if av.IntVal() != 123 {
		b.Fail()
	}
}

func BenchmarkAttributeValue_SetIntVal(b *testing.B) {
	av := NewValueString("k")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		av.SetIntVal(int64(n))
	}
	if av.IntVal() != int64(b.N-1) {
		b.Fail()
	}
}

func BenchmarkAttributeValueFloat_AsString(b *testing.B) {
	av := NewValueDouble(2359871345.583429543)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		av.AsString()
	}
}

func BenchmarkMap_Range(b *testing.B) {
	const numElements = 20
	rawOrig := make([]otlpcommon.KeyValue, numElements)
	for i := 0; i < numElements; i++ {
		rawOrig[i] = otlpcommon.KeyValue{
			Key:   "k" + strconv.Itoa(i),
			Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "v" + strconv.Itoa(i)}},
		}
	}
	am := Map{
		orig: &rawOrig,
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		numEls := 0
		am.Range(func(k string, v Value) bool {
			numEls++
			return true
		})
		if numEls != numElements {
			b.Fail()
		}
	}
}

func BenchmarkMap_RangeOverMap(b *testing.B) {
	const numElements = 20
	rawOrig := make(map[string]Value, numElements)
	for i := 0; i < numElements; i++ {
		key := "k" + strconv.Itoa(i)
		rawOrig[key] = NewValueString("v" + strconv.Itoa(i))
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		numEls := 0
		for _, v := range rawOrig {
			if v.orig == nil {
				continue
			}
			numEls++
		}
		if numEls != numElements {
			b.Fail()
		}
	}
}

func BenchmarkMap_Remove(b *testing.B) {
	b.StopTimer()
	// Remove all of the even keys
	keysToRemove := map[string]struct{}{}
	for j := 0; j < 50; j++ {
		keysToRemove[fmt.Sprintf("%d", j*2)] = struct{}{}
	}
	for i := 0; i < b.N; i++ {
		m := NewMap()
		for j := 0; j < 100; j++ {
			m.InsertString(fmt.Sprintf("%d", j), "string value")
		}
		b.StartTimer()
		for k := range keysToRemove {
			m.Remove(k)
		}
		b.StopTimer()
	}
}

func BenchmarkMap_RemoveIf(b *testing.B) {
	b.StopTimer()
	// Remove all of the even keys
	keysToRemove := map[string]struct{}{}
	for j := 0; j < 50; j++ {
		keysToRemove[fmt.Sprintf("%d", j*2)] = struct{}{}
	}
	for i := 0; i < b.N; i++ {
		m := NewMap()
		for j := 0; j < 100; j++ {
			m.InsertString(fmt.Sprintf("%d", j), "string value")
		}
		b.StartTimer()
		m.RemoveIf(func(key string, _ Value) bool {
			_, remove := keysToRemove[key]
			return remove
		})
		b.StopTimer()
	}
}

func BenchmarkStringMap_RangeOverMap(b *testing.B) {
	const numElements = 20
	rawOrig := make(map[string]string, numElements)
	for i := 0; i < numElements; i++ {
		key := "k" + strconv.Itoa(i)
		rawOrig[key] = "v" + strconv.Itoa(i)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		numEls := 0
		for _, v := range rawOrig {
			if v == "" {
				continue
			}
			numEls++
		}
		if numEls != numElements {
			b.Fail()
		}
	}
}

func fillTestValue(dest Value) {
	dest.SetStringVal("v")
}

func generateTestValue() Value {
	av := NewValueEmpty()
	fillTestValue(av)
	return av
}

func generateTestMap() Map {
	am := NewMap()
	fillTestMap(am)
	return am
}

func fillTestMap(dest Map) {
	NewMapFromRaw(map[string]interface{}{
		"k": "v",
	}).CopyTo(dest)
}

func generateTestEmptyMap() Map {
	return NewMapFromRaw(map[string]interface{}{
		"k": nil,
	})
}
func generateTestIntMap() Map {
	return NewMapFromRaw(map[string]interface{}{
		"k": 123,
	})
}

func generateTestDoubleMap() Map {
	return NewMapFromRaw(map[string]interface{}{
		"k": 12.3,
	})
}

func generateTestBoolMap() Map {
	return NewMapFromRaw(map[string]interface{}{
		"k": true,
	})
}

func generateTestBytesMap() Map {
	return NewMapFromRaw(map[string]interface{}{
		"k": []byte{1, 2, 3, 4, 5},
	})
}

func TestAttributeValueSlice(t *testing.T) {
	a1 := NewValueSlice()
	assert.EqualValues(t, ValueTypeSlice, a1.Type())
	assert.EqualValues(t, NewSlice(), a1.SliceVal())
	assert.EqualValues(t, 0, a1.SliceVal().Len())

	a1.SliceVal().AppendEmpty().SetDoubleVal(123)
	assert.EqualValues(t, 1, a1.SliceVal().Len())
	assert.EqualValues(t, NewValueDouble(123), a1.SliceVal().At(0))
	// Create a second array.
	a2 := NewValueSlice()
	assert.EqualValues(t, 0, a2.SliceVal().Len())

	a2.SliceVal().AppendEmpty().SetStringVal("somestr")
	assert.EqualValues(t, 1, a2.SliceVal().Len())
	assert.EqualValues(t, NewValueString("somestr"), a2.SliceVal().At(0))

	// Insert the second array as a child.
	a2.CopyTo(a1.SliceVal().AppendEmpty())
	assert.EqualValues(t, 2, a1.SliceVal().Len())
	assert.EqualValues(t, NewValueDouble(123), a1.SliceVal().At(0))
	assert.EqualValues(t, a2, a1.SliceVal().At(1))

	// Check that the array was correctly inserted.
	childArray := a1.SliceVal().At(1)
	assert.EqualValues(t, ValueTypeSlice, childArray.Type())
	assert.EqualValues(t, 1, childArray.SliceVal().Len())

	v := childArray.SliceVal().At(0)
	assert.EqualValues(t, ValueTypeString, v.Type())
	assert.EqualValues(t, "somestr", v.StringVal())

	// Test nil values case for SliceVal() func.
	a1 = Value{orig: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_ArrayValue{ArrayValue: nil}}}
	assert.EqualValues(t, Slice{}, a1.SliceVal())
}

func TestAttributeSliceWithNilValues(t *testing.T) {
	origWithNil := []otlpcommon.AnyValue{
		{},
		{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}},
	}
	sm := Slice{
		orig: &origWithNil,
	}

	val := sm.At(0)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.StringVal())

	val = sm.At(1)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "test_value", val.StringVal())

	sm.AppendEmpty().SetStringVal("other_value")
	val = sm.At(2)
	assert.EqualValues(t, ValueTypeString, val.Type())
	assert.EqualValues(t, "other_value", val.StringVal())
}

func TestAsString(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected string
	}{
		{
			name:     "string",
			input:    NewValueString("string value"),
			expected: "string value",
		},
		{
			name:     "int64",
			input:    NewValueInt(42),
			expected: "42",
		},
		{
			name:     "float64",
			input:    NewValueDouble(1.61803399),
			expected: "1.61803399",
		},
		{
			name:     "small float64",
			input:    NewValueDouble(.000000009),
			expected: "9e-9",
		},
		{
			name:     "bad float64",
			input:    NewValueDouble(math.Inf(1)),
			expected: "json: unsupported value: +Inf",
		},
		{
			name:     "boolean",
			input:    NewValueBool(true),
			expected: "true",
		},
		{
			name:     "empty_map",
			input:    NewValueMap(),
			expected: "{}",
		},
		{
			name:     "simple_map",
			input:    simpleValueMap(),
			expected: "{\"arrKey\":[\"strOne\",\"strTwo\"],\"boolKey\":false,\"floatKey\":18.6,\"intKey\":7,\"mapKey\":{\"keyOne\":\"valOne\",\"keyTwo\":\"valTwo\"},\"nullKey\":null,\"strKey\":\"strVal\"}",
		},
		{
			name:     "empty_array",
			input:    NewValueSlice(),
			expected: "[]",
		},
		{
			name:     "simple_array",
			input:    simpleValueSlice(),
			expected: "[\"strVal\",7,18.6,false,null]",
		},
		{
			name:     "empty",
			input:    NewValueEmpty(),
			expected: "",
		},
		{
			name:     "bytes",
			input:    NewValueBytes(NewImmutableByteSlice([]byte("String bytes"))),
			expected: base64.StdEncoding.EncodeToString([]byte("String bytes")),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.AsString()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestValueAsRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    Value
		expected interface{}
	}{
		{
			name:     "string",
			input:    NewValueString("value"),
			expected: "value",
		},
		{
			name:     "int",
			input:    NewValueInt(11),
			expected: int64(11),
		},
		{
			name:     "double",
			input:    NewValueDouble(1.2),
			expected: 1.2,
		},
		{
			name:     "bytes",
			input:    NewValueBytes(NewImmutableByteSlice([]byte("bytes"))),
			expected: []byte("bytes"),
		},
		{
			name:     "bytes",
			input:    NewValueBytes(NewImmutableByteSlice([]byte("bytes"))),
			expected: []byte("bytes"),
		},
		{
			name:     "empty",
			input:    NewValueEmpty(),
			expected: nil,
		},
		{
			name:     "slice",
			input:    simpleValueSlice(),
			expected: []interface{}{"strVal", int64(7), 18.6, false, nil},
		},
		{
			name:  "map",
			input: simpleValueMap(),
			expected: map[string]interface{}{
				"mapKey":   map[string]interface{}{"keyOne": "valOne", "keyTwo": "valTwo"},
				"nullKey":  nil,
				"strKey":   "strVal",
				"arrKey":   []interface{}{"strOne", "strTwo"},
				"boolKey":  false,
				"floatKey": 18.6,
				"intKey":   int64(7),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.asRaw()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestMapAsRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    Map
		expected map[string]interface{}
	}{
		{
			name: "asraw",
			input: NewMapFromRaw(
				map[string]interface{}{
					"array":  []interface{}{false, []byte("test"), 12.9, int64(91), "another string"},
					"bool":   true,
					"bytes":  []byte("bytes value"),
					"double": 1.2,
					"empty":  nil,
					"int":    int64(900),
					"map":    map[string]interface{}{},
					"string": "string value",
				},
			),
			expected: map[string]interface{}{
				"array":  []interface{}{false, []byte("test"), 12.9, int64(91), "another string"},
				"bool":   true,
				"bytes":  []byte("bytes value"),
				"double": 1.2,
				"empty":  nil,
				"int":    int64(900),
				"map":    map[string]interface{}{},
				"string": "string value",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.input.AsRaw()
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestNewValueFromRaw(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected Value
	}{
		{
			name:     "nil",
			input:    nil,
			expected: NewValueEmpty(),
		},
		{
			name:     "string",
			input:    "text",
			expected: NewValueString("text"),
		},
		{
			name:     "int",
			input:    123,
			expected: NewValueInt(int64(123)),
		},
		{
			name:     "int8",
			input:    int8(12),
			expected: NewValueInt(int64(12)),
		},
		{
			name:     "int16",
			input:    int16(23),
			expected: NewValueInt(int64(23)),
		},
		{
			name:     "int32",
			input:    int32(34),
			expected: NewValueInt(int64(34)),
		},
		{
			name:     "int64",
			input:    int64(45),
			expected: NewValueInt(45),
		},
		{
			name:     "uint",
			input:    uint(56),
			expected: NewValueInt(int64(56)),
		},
		{
			name:     "uint8",
			input:    uint8(67),
			expected: NewValueInt(int64(67)),
		},
		{
			name:     "uint16",
			input:    uint16(78),
			expected: NewValueInt(int64(78)),
		},
		{
			name:     "uint32",
			input:    uint32(89),
			expected: NewValueInt(int64(89)),
		},
		{
			name:     "uint64",
			input:    uint64(90),
			expected: NewValueInt(int64(90)),
		},
		{
			name:     "float32",
			input:    float32(1.234),
			expected: NewValueDouble(float64(float32(1.234))),
		},
		{
			name:     "float64",
			input:    float64(2.345),
			expected: NewValueDouble(float64(2.345)),
		},
		{
			name:     "bool",
			input:    true,
			expected: NewValueBool(true),
		},
		{
			name:     "bytes",
			input:    []byte{1, 2, 3},
			expected: NewValueBytes(NewImmutableByteSlice([]byte{1, 2, 3})),
		},
		{
			name: "map",
			input: map[string]interface{}{
				"k": "v",
			},
			expected: func() Value {
				m := NewValueMap()
				NewMapFromRaw(map[string]interface{}{
					"k": "v",
				}).CopyTo(m.MapVal())
				return m
			}(),
		},
		{
			name:  "slice",
			input: []interface{}{"v1", "v2"},
			expected: (func() Value {
				s := NewValueSlice()
				NewSliceFromRaw([]interface{}{"v1", "v2"}).CopyTo(s.SliceVal())
				return s
			})(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := newValueFromRaw(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func simpleValueMap() Value {
	ret := NewValueMap()
	attrMap := ret.MapVal()
	attrMap.UpsertString("strKey", "strVal")
	attrMap.UpsertInt("intKey", 7)
	attrMap.UpsertDouble("floatKey", 18.6)
	attrMap.UpsertBool("boolKey", false)
	attrMap.Upsert("nullKey", NewValueEmpty())
	attrMap.Upsert("mapKey", constructTestAttributeSubmap())
	attrMap.Upsert("arrKey", constructTestAttributeSubarray())
	return ret
}

func simpleValueSlice() Value {
	ret := NewValueSlice()
	attrArr := ret.SliceVal()
	attrArr.AppendEmpty().SetStringVal("strVal")
	attrArr.AppendEmpty().SetIntVal(7)
	attrArr.AppendEmpty().SetDoubleVal(18.6)
	attrArr.AppendEmpty().SetBoolVal(false)
	attrArr.AppendEmpty()
	return ret
}

func constructTestAttributeSubmap() Value {
	value := NewValueMap()
	value.MapVal().UpsertString("keyOne", "valOne")
	value.MapVal().UpsertString("keyTwo", "valTwo")
	return value
}

func constructTestAttributeSubarray() Value {
	value := NewValueSlice()
	value.SliceVal().AppendEmpty().SetStringVal("strOne")
	value.SliceVal().AppendEmpty().SetStringVal("strTwo")
	return value
}

func BenchmarkAttributeValueMapAccessor(b *testing.B) {
	val := simpleValueMap()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		val.MapVal()
	}
}

func BenchmarkAttributeValueSliceAccessor(b *testing.B) {
	val := simpleValueSlice()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		val.SliceVal()
	}
}
