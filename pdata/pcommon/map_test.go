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

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestMap(t *testing.T) {
	assert.EqualValues(t, 0, NewMap().Len())

	val, exist := NewMap().Get("test_key")
	assert.False(t, exist)
	assert.EqualValues(t, newValue(nil), val)

	putString := NewMap()
	putString.PutStr("k", "v")
	assert.EqualValues(t, Map(internal.GenerateTestMap()), putString)

	putInt := NewMap()
	putInt.PutInt("k", 123)
	assert.EqualValues(t, generateTestIntMap(t), putInt)

	putDouble := NewMap()
	putDouble.PutDouble("k", 12.3)
	assert.EqualValues(t, generateTestDoubleMap(t), putDouble)

	putBool := NewMap()
	putBool.PutBool("k", true)
	assert.EqualValues(t, generateTestBoolMap(t), putBool)

	putBytes := NewMap()
	putBytes.PutEmptyBytes("k").FromRaw([]byte{1, 2, 3, 4, 5})
	assert.EqualValues(t, generateTestBytesMap(t), putBytes)

	putMap := NewMap()
	putMap.PutEmptyMap("k")
	assert.EqualValues(t, generateTestEmptyMap(t), putMap)

	putSlice := NewMap()
	putSlice.PutEmptySlice("k")
	assert.EqualValues(t, generateTestEmptySlice(t), putSlice)

	removeMap := NewMap()
	assert.False(t, removeMap.Remove("k"))
	assert.EqualValues(t, NewMap(), removeMap)
}

func TestMapPutEmpty(t *testing.T) {
	m := NewMap()
	v := m.PutEmpty("k1")
	assert.EqualValues(t, map[string]any{
		"k1": nil,
	}, m.AsRaw())

	v.SetBool(true)
	assert.EqualValues(t, map[string]any{
		"k1": true,
	}, m.AsRaw())

	v = m.PutEmpty("k1")
	v.SetInt(1)
	v2, ok := m.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), v2.Int())
}

func TestMapPutEmptyMap(t *testing.T) {
	m := NewMap()
	childMap := m.PutEmptyMap("k1")
	assert.EqualValues(t, map[string]any{
		"k1": map[string]any{},
	}, m.AsRaw())
	childMap.PutEmptySlice("k2").AppendEmpty().SetStr("val")
	assert.EqualValues(t, map[string]any{
		"k1": map[string]any{
			"k2": []any{"val"},
		},
	}, m.AsRaw())

	childMap.PutEmptyMap("k2").PutInt("k3", 1)
	assert.EqualValues(t, map[string]any{
		"k1": map[string]any{
			"k2": map[string]any{"k3": int64(1)},
		},
	}, m.AsRaw())
}

func TestMapPutEmptySlice(t *testing.T) {
	m := NewMap()
	childSlice := m.PutEmptySlice("k")
	assert.EqualValues(t, map[string]any{
		"k": []any{},
	}, m.AsRaw())
	childSlice.AppendEmpty().SetDouble(1.1)
	assert.EqualValues(t, map[string]any{
		"k": []any{1.1},
	}, m.AsRaw())

	m.PutEmptySlice("k")
	assert.EqualValues(t, map[string]any{
		"k": []any{},
	}, m.AsRaw())
	childSliceVal, ok := m.Get("k")
	assert.True(t, ok)
	childSliceVal.Slice().AppendEmpty().SetEmptySlice().AppendEmpty().SetStr("val")
	assert.EqualValues(t, map[string]any{
		"k": []any{[]any{"val"}},
	}, m.AsRaw())
}

func TestMapPutEmptyBytes(t *testing.T) {
	m := NewMap()
	b := m.PutEmptyBytes("k")
	bv, ok := m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte(nil), bv.Bytes().AsRaw())
	b.FromRaw([]byte{1, 2, 3})
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, bv.Bytes().AsRaw())

	m.PutEmptyBytes("k")
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte(nil), bv.Bytes().AsRaw())
	bv.Bytes().FromRaw([]byte{3, 2, 1})
	bv, ok = m.Get("k")
	assert.True(t, ok)
	assert.Equal(t, []byte{3, 2, 1}, bv.Bytes().AsRaw())
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
	sm := newMap(&origWithNil)
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "test_value", val.Str())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.Str())

	sm.PutStr("other_key_string", "other_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "other_value", val.Str())

	sm.PutInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.Int())

	sm.PutDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 1.23, val.Double())

	sm.PutBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.True(t, val.Bool())

	sm.PutEmptyBytes("other_key_bytes").FromRaw([]byte{7, 8, 9})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{7, 8, 9}, val.Bytes().AsRaw())

	sm.PutStr("another_key_string", "another_value")
	val, exist = sm.Get("another_key_string")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "another_value", val.Str())

	sm.PutInt("another_key_int", 456)
	val, exist = sm.Get("another_key_int")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.Int())

	sm.PutDouble("another_key_double", 4.56)
	val, exist = sm.Get("another_key_double")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeDouble, val.Type())
	assert.EqualValues(t, 4.56, val.Double())

	sm.PutBool("another_key_bool", false)
	val, exist = sm.Get("another_key_bool")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBool, val.Type())
	assert.False(t, val.Bool())

	sm.PutEmptyBytes("another_key_bytes").FromRaw([]byte{1})
	val, exist = sm.Get("another_key_bytes")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeBytes, val.Type())
	assert.EqualValues(t, []byte{1}, val.Bytes().AsRaw())

	assert.True(t, sm.Remove("other_key_string"))
	assert.True(t, sm.Remove("other_key_int"))
	assert.True(t, sm.Remove("other_key_double"))
	assert.True(t, sm.Remove("other_key_bool"))
	assert.True(t, sm.Remove("other_key_bytes"))
	assert.True(t, sm.Remove("another_key_string"))
	assert.True(t, sm.Remove("another_key_int"))
	assert.True(t, sm.Remove("another_key_double"))
	assert.True(t, sm.Remove("another_key_bool"))
	assert.True(t, sm.Remove("another_key_bytes"))

	assert.False(t, sm.Remove("other_key_string"))
	assert.False(t, sm.Remove("another_key_string"))

	// Test that the initial key is still there.
	val, exist = sm.Get("test_key")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeStr, val.Type())
	assert.EqualValues(t, "test_value", val.Str())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.EqualValues(t, ValueTypeEmpty, val.Type())
	assert.EqualValues(t, "", val.Str())

	_, exist = sm.Get("test_key3")
	assert.False(t, exist)
}

func TestMapIterationNil(t *testing.T) {
	NewMap().Range(func(k string, v Value) bool {
		// Fail if any element is returned
		t.Fail()
		return true
	})
}

func TestMap_Range(t *testing.T) {
	rawMap := map[string]any{
		"k_string": "123",
		"k_int":    int64(123),
		"k_double": float64(1.23),
		"k_bool":   true,
		"k_empty":  nil,
	}
	am := NewMap()
	assert.NoError(t, am.FromRaw(rawMap))
	assert.Equal(t, 5, am.Len())

	calls := 0
	am.Range(func(k string, v Value) bool {
		calls++
		return false
	})
	assert.Equal(t, 1, calls)

	am.Range(func(k string, v Value) bool {
		assert.Equal(t, rawMap[k], v.AsRaw())
		delete(rawMap, k)
		return true
	})
	assert.EqualValues(t, 0, len(rawMap))
}

func TestMap_FromRaw(t *testing.T) {
	am := NewMap()
	assert.NoError(t, am.FromRaw(map[string]any{}))
	assert.Equal(t, 0, am.Len())
	am.PutEmpty("k")
	assert.Equal(t, 1, am.Len())

	assert.NoError(t, am.FromRaw(nil))
	assert.Equal(t, 0, am.Len())
	am.PutEmpty("k")
	assert.Equal(t, 1, am.Len())

	assert.NoError(t, am.FromRaw(map[string]any{
		"k_string": "123",
		"k_int":    123,
		"k_double": 1.23,
		"k_bool":   true,
		"k_null":   nil,
		"k_bytes":  []byte{1, 2, 3},
		"k_slice":  []any{1, 2.1, "val"},
		"k_map": map[string]any{
			"k_int":    1,
			"k_string": "val",
		},
	}))
	assert.Equal(t, 8, am.Len())
	v, ok := am.Get("k_string")
	assert.True(t, ok)
	assert.Equal(t, "123", v.Str())
	v, ok = am.Get("k_int")
	assert.True(t, ok)
	assert.Equal(t, int64(123), v.Int())
	v, ok = am.Get("k_double")
	assert.True(t, ok)
	assert.Equal(t, 1.23, v.Double())
	v, ok = am.Get("k_null")
	assert.True(t, ok)
	assert.Equal(t, ValueTypeEmpty, v.Type())
	v, ok = am.Get("k_bytes")
	assert.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, v.Bytes().AsRaw())
	v, ok = am.Get("k_slice")
	assert.True(t, ok)
	assert.Equal(t, []any{int64(1), 2.1, "val"}, v.Slice().AsRaw())
	v, ok = am.Get("k_map")
	assert.True(t, ok)
	assert.Equal(t, map[string]any{
		"k_int":    int64(1),
		"k_string": "val",
	}, v.Map().AsRaw())
}

func TestMap_CopyTo(t *testing.T) {
	dest := NewMap()
	// Test CopyTo to empty
	NewMap().CopyTo(dest)
	assert.EqualValues(t, 0, dest.Len())

	// Test CopyTo larger slice
	Map(internal.GenerateTestMap()).CopyTo(dest)
	assert.EqualValues(t, Map(internal.GenerateTestMap()), dest)

	// Test CopyTo same size slice
	Map(internal.GenerateTestMap()).CopyTo(dest)
	assert.EqualValues(t, Map(internal.GenerateTestMap()), dest)

	// Test CopyTo with an empty Value in the destination
	(*dest.getOrig())[0].Value = otlpcommon.AnyValue{}
	Map(internal.GenerateTestMap()).CopyTo(dest)
	assert.EqualValues(t, Map(internal.GenerateTestMap()), dest)
}

func TestMap_EnsureCapacity_Zero(t *testing.T) {
	am := NewMap()
	am.EnsureCapacity(0)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 0, cap(*am.getOrig()))
}

func TestMap_EnsureCapacity(t *testing.T) {
	am := NewMap()
	am.EnsureCapacity(5)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.getOrig()))
	am.EnsureCapacity(3)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 5, cap(*am.getOrig()))
	am.EnsureCapacity(8)
	assert.Equal(t, 0, am.Len())
	assert.Equal(t, 8, cap(*am.getOrig()))
}

func TestMap_Clear(t *testing.T) {
	am := NewMap()
	assert.Nil(t, *am.getOrig())
	am.Clear()
	assert.Nil(t, *am.getOrig())
	am.EnsureCapacity(5)
	assert.NotNil(t, *am.getOrig())
	am.Clear()
	assert.Nil(t, *am.getOrig())
}

func TestMap_RemoveIf(t *testing.T) {
	am := NewMap()
	am.PutStr("k_string", "123")
	am.PutInt("k_int", int64(123))
	am.PutDouble("k_double", float64(1.23))
	am.PutBool("k_bool", true)
	am.PutEmpty("k_empty")

	assert.Equal(t, 5, am.Len())

	am.RemoveIf(func(key string, val Value) bool {
		return key == "k_int" || val.Type() == ValueTypeBool
	})
	assert.Equal(t, 3, am.Len())
	_, exists := am.Get("k_string")
	assert.True(t, exists)
	_, exists = am.Get("k_int")
	assert.False(t, exists)
	_, exists = am.Get("k_double")
	assert.True(t, exists)
	_, exists = am.Get("k_bool")
	assert.False(t, exists)
	_, exists = am.Get("k_empty")
	assert.True(t, exists)
}

func generateTestEmptyMap(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": map[string]any(nil)}))
	return m
}

func generateTestEmptySlice(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": []any(nil)}))
	return m
}

func generateTestIntMap(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": 123}))
	return m
}

func generateTestDoubleMap(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": 12.3}))
	return m
}

func generateTestBoolMap(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": true}))
	return m
}

func generateTestBytesMap(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": []byte{1, 2, 3, 4, 5}}))
	return m
}
