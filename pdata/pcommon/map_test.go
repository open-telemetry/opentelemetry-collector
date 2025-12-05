// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/internal"
)

func TestMap(t *testing.T) {
	assert.Equal(t, 0, NewMap().Len())

	val, exist := NewMap().Get("test_key")
	assert.False(t, exist)
	assert.Equal(t, newValue(nil, internal.NewState()), val)

	putString := NewMap()
	putString.PutStr("k", "v")
	assert.Equal(t, generateTestStringMap(t), putString)

	putInt := NewMap()
	putInt.PutInt("k", 123)
	assert.Equal(t, generateTestIntMap(t), putInt)

	putDouble := NewMap()
	putDouble.PutDouble("k", 12.3)
	assert.Equal(t, generateTestDoubleMap(t), putDouble)

	putBool := NewMap()
	putBool.PutBool("k", true)
	assert.Equal(t, generateTestBoolMap(t), putBool)

	putBytes := NewMap()
	putBytes.PutEmptyBytes("k").FromRaw([]byte{1, 2, 3, 4, 5})
	assert.Equal(t, generateTestBytesMap(t), putBytes)

	putMap := NewMap()
	putMap.PutEmptyMap("k")
	assert.Equal(t, generateTestEmptyMap(t), putMap)

	putSlice := NewMap()
	putSlice.PutEmptySlice("k")
	assert.Equal(t, generateTestEmptySlice(t), putSlice)

	removeMap := NewMap()
	assert.False(t, removeMap.Remove("k"))
	assert.Equal(t, NewMap(), removeMap)
}

func TestMapReadOnly(t *testing.T) {
	state := internal.NewState()
	state.MarkReadOnly()
	m := newMap(&[]internal.KeyValue{
		{Key: "k1", Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "v1"}}},
	}, state)

	assert.Equal(t, 1, m.Len())

	v, ok := m.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, "v1", v.Str())

	m.Range(func(k string, v Value) bool {
		assert.Equal(t, "k1", k)
		assert.Equal(t, "v1", v.Str())
		return true
	})

	assert.Panics(t, func() { m.PutStr("k2", "v2") })
	assert.Panics(t, func() { m.PutInt("k2", 123) })
	assert.Panics(t, func() { m.PutDouble("k2", 1.23) })
	assert.Panics(t, func() { m.PutBool("k2", true) })
	assert.Panics(t, func() { m.PutEmpty("foo") })
	assert.Panics(t, func() { m.GetOrPutEmpty("foo") })
	assert.Panics(t, func() { m.PutEmptyBytes("k2") })
	assert.Panics(t, func() { m.PutEmptyMap("k2") })
	assert.Panics(t, func() { m.PutEmptySlice("k2") })
	assert.Panics(t, func() { m.Remove("k1") })
	assert.Panics(t, func() { m.RemoveIf(func(string, Value) bool { return true }) })
	assert.Panics(t, func() { m.EnsureCapacity(2) })

	m2 := NewMap()
	m.CopyTo(m2)
	assert.Equal(t, m2.AsRaw(), m.AsRaw())
	assert.Panics(t, func() { NewMap().CopyTo(m) })

	assert.Equal(t, map[string]any{"k1": "v1"}, m.AsRaw())
	assert.Panics(t, func() { _ = m.FromRaw(map[string]any{"k1": "v1"}) })
}

func TestMapPutEmpty(t *testing.T) {
	m := NewMap()
	v := m.PutEmpty("k1")
	assert.Equal(t, map[string]any{
		"k1": nil,
	}, m.AsRaw())

	v.SetBool(true)
	assert.Equal(t, map[string]any{
		"k1": true,
	}, m.AsRaw())

	v = m.PutEmpty("k1")
	v.SetInt(1)
	v2, ok := m.Get("k1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), v2.Int())
}

func TestMapGetOrPutEmpty(t *testing.T) {
	m := NewMap()
	v := m.PutEmpty("k1")
	v.SetStr("test")
	assert.Equal(t, map[string]any{
		"k1": "test",
	}, m.AsRaw())

	v, found := m.GetOrPutEmpty("k1")
	assert.True(t, found)
	require.Equal(t, ValueTypeStr, v.Type())
	assert.Equal(t, "test", v.Str())

	v, found = m.GetOrPutEmpty("k2")
	assert.False(t, found)
	require.Equal(t, ValueTypeEmpty, v.Type())
}

func TestMapPutEmptyMap(t *testing.T) {
	m := NewMap()
	childMap := m.PutEmptyMap("k1")
	assert.Equal(t, map[string]any{
		"k1": map[string]any{},
	}, m.AsRaw())
	childMap.PutEmptySlice("k2").AppendEmpty().SetStr("val")
	assert.Equal(t, map[string]any{
		"k1": map[string]any{
			"k2": []any{"val"},
		},
	}, m.AsRaw())

	childMap.PutEmptyMap("k2").PutInt("k3", 1)
	assert.Equal(t, map[string]any{
		"k1": map[string]any{
			"k2": map[string]any{"k3": int64(1)},
		},
	}, m.AsRaw())
}

func TestMapPutEmptySlice(t *testing.T) {
	m := NewMap()
	childSlice := m.PutEmptySlice("k")
	assert.Equal(t, map[string]any{
		"k": []any{},
	}, m.AsRaw())
	childSlice.AppendEmpty().SetDouble(1.1)
	assert.Equal(t, map[string]any{
		"k": []any{1.1},
	}, m.AsRaw())

	m.PutEmptySlice("k")
	assert.Equal(t, map[string]any{
		"k": []any{},
	}, m.AsRaw())
	childSliceVal, ok := m.Get("k")
	assert.True(t, ok)
	childSliceVal.Slice().AppendEmpty().SetEmptySlice().AppendEmpty().SetStr("val")
	assert.Equal(t, map[string]any{
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
	origWithNil := []internal.KeyValue{
		{},
		{
			Key:   "test_key",
			Value: internal.AnyValue{Value: &internal.AnyValue_StringValue{StringValue: "test_value"}},
		},
		{
			Key:   "test_key2",
			Value: internal.AnyValue{Value: nil},
		},
	}
	sm := newMap(&origWithNil, internal.NewState())
	val, exist := sm.Get("test_key")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeStr, val.Type())
	assert.Equal(t, "test_value", val.Str())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeEmpty, val.Type())
	assert.Empty(t, val.Str())

	sm.PutStr("other_key_string", "other_value")
	val, exist = sm.Get("other_key_string")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeStr, val.Type())
	assert.Equal(t, "other_value", val.Str())

	sm.PutInt("other_key_int", 123)
	val, exist = sm.Get("other_key_int")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 123, val.Int())

	sm.PutDouble("other_key_double", 1.23)
	val, exist = sm.Get("other_key_double")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeDouble, val.Type())
	assert.InDelta(t, 1.23, val.Double(), 0.01)

	sm.PutBool("other_key_bool", true)
	val, exist = sm.Get("other_key_bool")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeBool, val.Type())
	assert.True(t, val.Bool())

	sm.PutEmptyBytes("other_key_bytes").FromRaw([]byte{7, 8, 9})
	val, exist = sm.Get("other_key_bytes")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeBytes, val.Type())
	assert.Equal(t, []byte{7, 8, 9}, val.Bytes().AsRaw())

	sm.PutStr("another_key_string", "another_value")
	val, exist = sm.Get("another_key_string")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeStr, val.Type())
	assert.Equal(t, "another_value", val.Str())

	sm.PutInt("another_key_int", 456)
	val, exist = sm.Get("another_key_int")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeInt, val.Type())
	assert.EqualValues(t, 456, val.Int())

	sm.PutDouble("another_key_double", 4.56)
	val, exist = sm.Get("another_key_double")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeDouble, val.Type())
	assert.InDelta(t, 4.56, val.Double(), 0.01)

	sm.PutBool("another_key_bool", false)
	val, exist = sm.Get("another_key_bool")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeBool, val.Type())
	assert.False(t, val.Bool())

	sm.PutEmptyBytes("another_key_bytes").FromRaw([]byte{1})
	val, exist = sm.Get("another_key_bytes")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeBytes, val.Type())
	assert.Equal(t, []byte{1}, val.Bytes().AsRaw())

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
	assert.Equal(t, ValueTypeStr, val.Type())
	assert.Equal(t, "test_value", val.Str())

	val, exist = sm.Get("test_key2")
	assert.True(t, exist)
	assert.Equal(t, ValueTypeEmpty, val.Type())
	assert.Empty(t, val.Str())

	_, exist = sm.Get("test_key3")
	assert.False(t, exist)
}

func TestMapIterationNil(t *testing.T) {
	NewMap().Range(func(string, Value) bool {
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
	require.NoError(t, am.FromRaw(rawMap))
	assert.Equal(t, 5, am.Len())

	calls := 0
	am.Range(func(string, Value) bool {
		calls++
		return false
	})
	assert.Equal(t, 1, calls)

	am.Range(func(k string, v Value) bool {
		assert.Equal(t, rawMap[k], v.AsRaw())
		delete(rawMap, k)
		return true
	})
	assert.Empty(t, rawMap)
}

func TestMap_All(t *testing.T) {
	rawMap := map[string]any{
		"k_string": "123",
		"k_int":    int64(123),
		"k_double": float64(1.23),
		"k_bool":   true,
		"k_empty":  nil,
	}
	am := NewMap()
	require.NoError(t, am.FromRaw(rawMap))
	assert.Equal(t, 5, am.Len())

	calls := 0
	for range am.All() {
		calls++
	}
	assert.Equal(t, am.Len(), calls)

	for k, v := range am.All() {
		assert.Equal(t, rawMap[k], v.AsRaw())
		delete(rawMap, k)
	}
	assert.Empty(t, rawMap)
}

func TestMap_FromRaw(t *testing.T) {
	am := NewMap()
	require.NoError(t, am.FromRaw(map[string]any{}))
	assert.Equal(t, 0, am.Len())
	am.PutEmpty("k")
	assert.Equal(t, 1, am.Len())

	require.NoError(t, am.FromRaw(nil))
	assert.Equal(t, 0, am.Len())
	am.PutEmpty("k")
	assert.Equal(t, 1, am.Len())

	require.NoError(t, am.FromRaw(map[string]any{
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
	assert.InDelta(t, 1.23, v.Double(), 0.01)
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

func TestMap_MoveTo(t *testing.T) {
	dest := NewMap()
	// Test MoveTo to empty
	NewMap().MoveTo(dest)
	assert.Equal(t, 0, dest.Len())

	// Test MoveTo larger slice
	src := Map(internal.GenTestMapWrapper())
	src.MoveTo(dest)
	assert.Equal(t, Map(internal.GenTestMapWrapper()), dest)
	assert.Equal(t, 0, src.Len())

	// Test MoveTo from empty to non-empty
	NewMap().MoveTo(dest)
	assert.Equal(t, 0, dest.Len())

	dest.PutStr("k", "v")
	dest.MoveTo(dest)
	assert.Equal(t, 1, dest.Len())
	assert.Equal(t, map[string]any{"k": "v"}, dest.AsRaw())
}

func TestMap_CopyTo(t *testing.T) {
	dest := NewMap()
	// Test CopyTo to empty
	NewMap().CopyTo(dest)
	assert.Equal(t, 0, dest.Len())

	// Test CopyTo larger slice
	Map(internal.GenTestMapWrapper()).CopyTo(dest)
	assert.Equal(t, Map(internal.GenTestMapWrapper()), dest)

	// Test CopyTo same size slice
	Map(internal.GenTestMapWrapper()).CopyTo(dest)
	assert.Equal(t, Map(internal.GenTestMapWrapper()), dest)

	// Test CopyTo with an empty Value in the destination
	(*dest.getOrig())[0].Value = internal.AnyValue{}
	Map(internal.GenTestMapWrapper()).CopyTo(dest)
	assert.Equal(t, Map(internal.GenTestMapWrapper()), dest)

	// Test CopyTo same size slice
	dest.CopyTo(dest)
	assert.Equal(t, Map(internal.GenTestMapWrapper()), dest)
}

func TestMap_CopyToAndEnsureCapacity(t *testing.T) {
	dest := NewMap()
	src := Map(internal.GenTestMapWrapper())
	dest.EnsureCapacity(src.Len())
	src.CopyTo(dest)
	assert.Equal(t, Map(internal.GenTestMapWrapper()), dest)
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

func TestMap_EnsureCapacity_Existing(t *testing.T) {
	am := NewMap()
	am.PutStr("foo", "bar")

	assert.Equal(t, 1, am.Len())

	// Add more capacity.
	am.EnsureCapacity(5)

	// Ensure previously existing element is still there.
	assert.Equal(t, 1, am.Len())
	v, ok := am.Get("foo")
	assert.Equal(t, "bar", v.Str())
	assert.True(t, ok)

	assert.Equal(t, 5, cap(*am.getOrig()))

	// Add one more element.
	am.PutStr("abc", "xyz")

	// Verify that both elements are there.
	assert.Equal(t, 2, am.Len())

	v, ok = am.Get("foo")
	assert.Equal(t, "bar", v.Str())
	assert.True(t, ok)

	v, ok = am.Get("abc")
	assert.Equal(t, "xyz", v.Str())
	assert.True(t, ok)
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

func TestMap_RemoveIfAll(t *testing.T) {
	am := Map(internal.GenTestMapWrapper())
	assert.Equal(t, 5, am.Len())
	am.RemoveIf(func(string, Value) bool {
		return true
	})
	assert.Equal(t, 0, am.Len())
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

func generateTestStringMap(t *testing.T) Map {
	m := NewMap()
	assert.NoError(t, m.FromRaw(map[string]any{"k": "v"}))
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

func TestInvalidMap(t *testing.T) {
	v := Map{}

	testFunc := func(string, Value) bool {
		return true
	}

	assert.Panics(t, func() { v.Clear() })
	assert.Panics(t, func() { v.EnsureCapacity(1) })
	assert.Panics(t, func() { v.Get("foo") })
	assert.Panics(t, func() { v.Remove("foo") })
	assert.Panics(t, func() { v.RemoveIf(testFunc) })
	assert.Panics(t, func() { v.PutEmpty("foo") })
	assert.Panics(t, func() { v.GetOrPutEmpty("foo") })
	assert.Panics(t, func() { v.PutStr("foo", "bar") })
	assert.Panics(t, func() { v.PutInt("foo", 1) })
	assert.Panics(t, func() { v.PutDouble("foo", 1.1) })
	assert.Panics(t, func() { v.PutBool("foo", true) })
	assert.Panics(t, func() { v.PutEmptyBytes("foo") })
	assert.Panics(t, func() { v.PutEmptyMap("foo") })
	assert.Panics(t, func() { v.PutEmptySlice("foo") })
	assert.Panics(t, func() { v.Len() })
	assert.Panics(t, func() { v.Range(testFunc) })
	assert.Panics(t, func() { v.CopyTo(NewMap()) })
	assert.Panics(t, func() { v.AsRaw() })
	assert.Panics(t, func() { _ = v.FromRaw(map[string]any{"foo": "bar"}) })
}

func TestMapEqual(t *testing.T) {
	for _, tt := range []struct {
		name       string
		val        Map
		comparison Map
		expected   bool
	}{
		{
			name:       "with two empty maps",
			val:        NewMap(),
			comparison: NewMap(),
			expected:   true,
		},
		{
			name: "with two equal values",
			val: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				return m
			}(),
			comparison: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				return m
			}(),
			expected: true,
		},
		{
			name: "with multiple equal values",
			val: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				m.PutStr("bonjour", "monde")
				return m
			}(),
			comparison: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				m.PutStr("bonjour", "monde")
				return m
			}(),
			expected: true,
		},
		{
			name: "with two different values",
			val: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				return m
			}(),
			comparison: func() Map {
				m := NewMap()
				m.PutStr("bonjour", "monde")
				return m
			}(),
			expected: false,
		},
		{
			name: "with the same key and different values",
			val: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				return m
			}(),
			comparison: func() Map {
				m := NewMap()
				m.PutStr("hello", "monde")
				return m
			}(),
			expected: false,
		},
		{
			name: "with multiple different values",
			val: func() Map {
				m := NewMap()
				m.PutStr("hello", "world")
				m.PutStr("bonjour", "monde")
				return m
			}(),
			comparison: func() Map {
				m := NewMap()
				m.PutStr("question", "unknown")
				m.PutStr("answer", "42")
				return m
			}(),
			expected: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.val.Equal(tt.comparison))
		})
	}
}

func BenchmarkMapEqual(b *testing.B) {
	testutil.SkipMemoryBench(b)
	m := NewMap()
	m.PutStr("hello", "world")
	cmp := NewMap()
	cmp.PutStr("hello", "world")

	b.ReportAllocs()

	for b.Loop() {
		_ = m.Equal(cmp)
	}
}
