// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xhash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestMapHash(t *testing.T) {
	tests := []struct {
		name  string
		maps  []pcommon.Map
		equal bool
	}{
		{
			name: "different_maps",
			maps: func() []pcommon.Map {
				m := make([]pcommon.Map, 29)
				for i := range m {
					m[i] = pcommon.NewMap()
				}
				m[1].PutStr("k", "")
				m[2].PutStr("k", "v")
				m[3].PutStr("k1", "v1")
				m[4].PutBool("k", false)
				m[5].PutBool("k", true)
				m[6].PutInt("k", 0)
				m[7].PutInt("k", 1)
				m[8].PutDouble("k", 0)
				m[9].PutDouble("k", 1)
				m[10].PutEmpty("k")

				m[11].PutStr("k1", "val")
				m[11].PutStr("k2", "val")
				m[12].PutStr("k1", "va")
				m[12].PutStr("lk2", "val")

				m[13].PutEmptySlice("k")
				m[14].PutEmptySlice("k").AppendEmpty()
				m[15].PutEmptySlice("k").AppendEmpty().SetStr("")
				m[16].PutEmptySlice("k").AppendEmpty().SetStr("v")
				sl1 := m[17].PutEmptySlice("k")
				sl1.AppendEmpty().SetStr("v1")
				sl1.AppendEmpty().SetStr("v2")
				sl2 := m[18].PutEmptySlice("k")
				sl2.AppendEmpty().SetStr("v2")
				sl2.AppendEmpty().SetStr("v1")

				m[19].PutEmptyBytes("k")
				m[20].PutEmptyBytes("k").FromRaw([]byte{0})
				m[21].PutEmptyBytes("k").FromRaw([]byte{1})

				m[22].PutEmptyMap("k")
				m[23].PutEmptyMap("k").PutStr("k", "")
				m[24].PutEmptyMap("k").PutBool("k", false)
				m[25].PutEmptyMap("k").PutEmptyMap("")
				m[26].PutEmptyMap("k").PutEmptyMap("k")

				m[27].PutStr("k1", "v1")
				m[27].PutStr("k2", "v2")
				m[28].PutEmptyMap("k0").PutStr("k1", "v1")
				m[28].PutStr("k2", "v2")

				return m
			}(),
			equal: false,
		},
		{
			name:  "empty_maps",
			maps:  []pcommon.Map{pcommon.NewMap(), pcommon.NewMap()},
			equal: true,
		},
		{
			name: "same_maps_different_order",
			maps: func() []pcommon.Map {
				m := []pcommon.Map{pcommon.NewMap(), pcommon.NewMap()}
				m[0].PutStr("k1", "v1")
				m[0].PutInt("k2", 1)
				m[0].PutDouble("k3", 1)
				m[0].PutBool("k4", true)
				m[0].PutEmptyBytes("k5").FromRaw([]byte("abc"))
				sl := m[0].PutEmptySlice("k6")
				sl.AppendEmpty().SetStr("str")
				sl.AppendEmpty().SetBool(true)
				m0 := m[0].PutEmptyMap("k")
				m0.PutInt("k1", 1)
				m0.PutDouble("k2", 10)

				m1 := m[1].PutEmptyMap("k")
				m1.PutDouble("k2", 10)
				m1.PutInt("k1", 1)
				m[1].PutEmptyBytes("k5").FromRaw([]byte("abc"))
				m[1].PutBool("k4", true)
				sl = m[1].PutEmptySlice("k6")
				sl.AppendEmpty().SetStr("str")
				sl.AppendEmpty().SetBool(true)
				m[1].PutInt("k2", 1)
				m[1].PutStr("k1", "v1")
				m[1].PutDouble("k3", 1)

				return m
			}(),
			equal: true,
		},
		{
			// Specific test to ensure panic described in https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/18910 is fixed.
			name: "nested_maps_different_order",
			maps: func() []pcommon.Map {
				m := []pcommon.Map{pcommon.NewMap(), pcommon.NewMap()}
				m[0].PutStr("k1", "v1")
				m0 := m[0].PutEmptyMap("k2")
				m[0].PutDouble("k3", 1)
				m[0].PutBool("k4", true)
				m0.PutInt("k21", 1)
				m0.PutInt("k22", 1)
				m0.PutInt("k23", 1)

				m1 := m[1].PutEmptyMap("k2")
				m1.PutInt("k22", 1)
				m1.PutInt("k21", 1)
				m1.PutInt("k23", 1)
				m[1].PutDouble("k3", 1)
				m[1].PutStr("k1", "v1")
				m[1].PutBool("k4", true)

				return m
			}(),
			equal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < len(tt.maps); i++ {
				for j := i + 1; j < len(tt.maps); j++ {
					if tt.equal {
						assert.Equal(t, MapHash(tt.maps[i]), MapHash(tt.maps[j]),
							"maps %d %v and %d %v must have the same hash", i, tt.maps[i].AsRaw(), j, tt.maps[j].AsRaw())
					} else {
						assert.NotEqual(t, MapHash(tt.maps[i]), MapHash(tt.maps[j]),
							"maps %d %v and %d %v must have different hashes", i, tt.maps[i].AsRaw(), j, tt.maps[j].AsRaw())
					}
				}
			}
		})
	}
}

func TestValueHash(t *testing.T) {
	tests := []struct {
		name   string
		values []pcommon.Value
		equal  bool
	}{
		{
			name: "different_values",
			values: func() []pcommon.Value {
				m := make([]pcommon.Value, 21)
				for i := range m {
					m[i] = pcommon.NewValueEmpty()
				}
				m[1].SetStr("")
				m[2].SetStr("v")
				m[3].SetBool(false)
				m[4].SetBool(true)
				m[5].SetInt(0)
				m[6].SetInt(1)
				m[7].SetDouble(0)
				m[8].SetDouble(1)

				m[9].SetEmptySlice()
				m[10].SetEmptySlice().AppendEmpty()
				m[11].SetEmptySlice().AppendEmpty().SetStr("")
				m[12].SetEmptySlice().AppendEmpty().SetStr("v")

				m[13].SetEmptyBytes()
				m[14].SetEmptyBytes().FromRaw([]byte{0})
				m[15].SetEmptyBytes().FromRaw([]byte{1})

				m[16].SetEmptyMap()
				m[17].SetEmptyMap().PutStr("k", "")
				m[18].SetEmptyMap().PutBool("k", false)
				m[19].SetEmptyMap().PutEmptyMap("")
				m[20].SetEmptyMap().PutEmptyMap("k")

				return m
			}(),
			equal: false,
		},
		{
			name:   "empty_values",
			values: []pcommon.Value{pcommon.NewValueEmpty(), pcommon.NewValueEmpty()},
			equal:  true,
		},
		{
			name:   "empty_strings",
			values: []pcommon.Value{pcommon.NewValueStr(""), pcommon.NewValueStr("")},
			equal:  true,
		},
		{
			name:   "strings",
			values: []pcommon.Value{pcommon.NewValueStr("v"), pcommon.NewValueStr("v")},
			equal:  true,
		},
		{
			name:   "int",
			values: []pcommon.Value{pcommon.NewValueInt(1), pcommon.NewValueInt(1)},
			equal:  true,
		},
		{
			name:   "double",
			values: []pcommon.Value{pcommon.NewValueDouble(1), pcommon.NewValueDouble(1)},
			equal:  true,
		},
		{
			name:   "bool",
			values: []pcommon.Value{pcommon.NewValueBool(true), pcommon.NewValueBool(true)},
			equal:  true,
		},
		{
			name:   "empty_bytes",
			values: []pcommon.Value{pcommon.NewValueBytes(), pcommon.NewValueBytes()},
			equal:  true,
		},
		{
			name: "bytes",
			values: func() []pcommon.Value {
				v1 := pcommon.NewValueBytes()
				require.NoError(t, v1.FromRaw([]byte{0}))
				v2 := pcommon.NewValueBytes()
				require.NoError(t, v2.FromRaw([]byte{0}))
				return []pcommon.Value{v1, v2}
			}(),
			equal: true,
		},
		{
			name:   "empty_slices",
			values: []pcommon.Value{pcommon.NewValueSlice(), pcommon.NewValueSlice()},
			equal:  true,
		},
		{
			name: "slices_with_empty_items",
			values: func() []pcommon.Value {
				v1 := pcommon.NewValueSlice()
				v1.Slice().AppendEmpty()
				v2 := pcommon.NewValueSlice()
				v2.Slice().AppendEmpty()
				return []pcommon.Value{v1, v2}
			}(),
			equal: true,
		},
		{
			name: "slices",
			values: func() []pcommon.Value {
				v1 := pcommon.NewValueSlice()
				v1.Slice().AppendEmpty().SetStr("v")
				v2 := pcommon.NewValueSlice()
				v2.Slice().AppendEmpty().SetStr("v")
				return []pcommon.Value{v1, v2}
			}(),
			equal: true,
		},
		{
			name:   "empty_maps",
			values: []pcommon.Value{pcommon.NewValueMap(), pcommon.NewValueMap()},
			equal:  true,
		},
		{
			name: "maps",
			values: func() []pcommon.Value {
				v1 := pcommon.NewValueMap()
				v1.Map().PutStr("k1", "v")
				v1.Map().PutInt("k2", 0)
				v2 := pcommon.NewValueMap()
				v2.Map().PutInt("k2", 0)
				v2.Map().PutStr("k1", "v")
				return []pcommon.Value{v1, v2}
			}(),
			equal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < len(tt.values); i++ {
				for j := i + 1; j < len(tt.values); j++ {
					if tt.equal {
						assert.Equal(t, ValueHash(tt.values[i]), ValueHash(tt.values[j]),
							"values %d %v and %d %v must have the same hash", i, tt.values[i].AsRaw(), j, tt.values[j].AsRaw())
					} else {
						assert.NotEqual(t, ValueHash(tt.values[i]), ValueHash(tt.values[j]),
							"values %d %v and %d %v must have different hashes", i, tt.values[i].AsRaw(), j, tt.values[j].AsRaw())
					}
				}
			}
		})
	}
}

func TestMapValueHashNotEqual(t *testing.T) {
	tests := []struct {
		name string
		m    pcommon.Map
		v    pcommon.Value
	}{
		{
			name: "empty",
			v:    pcommon.NewValueMap(),
			m:    pcommon.NewMap(),
		},
		{
			name: "not_empty",
			v: func() pcommon.Value {
				v := pcommon.NewValueMap()
				v.Map().PutStr("k", "v")
				return v
			}(),
			m: func() pcommon.Map {
				m := pcommon.NewMap()
				m.PutStr("k", "v")
				return m
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEqual(t, ValueHash(tt.v), MapHash(tt.m),
				"value %v and map %v must have different hashes", tt.v.AsRaw(), tt.m.AsRaw())
		})
	}
}

func TestHash(t *testing.T) {
	assert.Equal(t, emptyHash, Hash())

	m := pcommon.NewMap()
	m.PutStr("k", "v")
	v := pcommon.NewValueInt(1)

	mapHashA, mapHashB := Hash(WithMap(m)), Hash(WithMap(m))
	assert.Equal(t, mapHashA, mapHashB)

	valueHashA, valueHashB := Hash(WithValue(v)), Hash(WithValue(v))
	assert.Equal(t, valueHashA, valueHashB)

	stringHashA, stringHashB := Hash(WithString("s")), Hash(WithString("s"))
	assert.Equal(t, stringHashA, stringHashB)
	assert.NotEqual(t, Hash(WithString("s1")), Hash(WithString("s2")))

	assert.NotEqual(t, mapHashA, valueHashA)
	assert.NotEqual(t, Hash(), Hash(WithString("")))

	// Combining options in one call hashes them in order.
	combinedA, combinedB := Hash(WithMap(m), WithValue(v)), Hash(WithMap(m), WithValue(v))
	assert.Equal(t, combinedA, combinedB)
	assert.NotEqual(t, Hash(WithMap(m), WithValue(v)), Hash(WithValue(v), WithMap(m)))
}

func TestHash64(t *testing.T) {
	hashA, hashB := Hash64(WithString("s")), Hash64(WithString("s"))
	assert.Equal(t, hashA, hashB)
	assert.NotEqual(t, Hash64(WithString("s1")), Hash64(WithString("s2")))
}

func BenchmarkMapHashFourItems(b *testing.B) {
	m := pcommon.NewMap()
	m.PutStr("test-string-key2", "test-value-2")
	m.PutStr("test-string-key1", "test-value-1")
	m.PutInt("test-int-key", 123)
	m.PutBool("test-bool-key", true)

	b.ReportAllocs()

	for b.Loop() {
		MapHash(m)
	}
}

func BenchmarkMapHashEightItems(b *testing.B) {
	m := pcommon.NewMap()
	m.PutStr("test-string-key2", "test-value-2")
	m.PutStr("test-string-key1", "test-value-1")
	m.PutInt("test-int-key", 123)
	m.PutBool("test-bool-key", true)
	m.PutStr("test-string-key3", "test-value-3")
	m.PutDouble("test-double-key2", 22.123)
	m.PutDouble("test-double-key1", 11.123)
	m.PutEmptyBytes("test-bytes-key").FromRaw([]byte("abc"))

	b.ReportAllocs()

	for b.Loop() {
		MapHash(m)
	}
}

func BenchmarkMapHashWithEmbeddedSliceAndMap(b *testing.B) {
	m := pcommon.NewMap()
	m.PutStr("test-string-key2", "test-value-2")
	m.PutStr("test-string-key1", "test-value-1")
	m.PutInt("test-int-key", 123)
	m.PutBool("test-bool-key", true)
	m.PutStr("test-string-key3", "test-value-3")
	m.PutDouble("test-double-key2", 22.123)
	m.PutDouble("test-double-key1", 11.123)
	m.PutEmptyBytes("test-bytes-key").FromRaw([]byte("abc"))
	m1 := m.PutEmptyMap("test-map-key")
	m1.PutStr("test-embedded-string-key", "test-embedded-string-value")
	m1.PutDouble("test-embedded-double-key", 22.123)
	m1.PutInt("test-embedded-int-key", 234)
	sl := m.PutEmptySlice("test-slice-key")
	sl.AppendEmpty().SetStr("test-slice-string-1")
	sl.AppendEmpty().SetStr("test-slice-string-2")
	sl.AppendEmpty().SetStr("test-slice-string-3")

	b.ReportAllocs()

	for b.Loop() {
		MapHash(m)
	}
}
