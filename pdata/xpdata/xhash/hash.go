// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xhash // import "go.opentelemetry.io/collector/pdata/xpdata/xhash"

import (
	"encoding/binary"
	"hash/maphash"
	"math"
	"sort"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// seed is process-wide: MapHash only needs to be consistent within one
// process's lifetime, never across processes or restarts.
var seed = maphash.MakeSeed()

// MapHash hashes m's keys in sorted order, independent of iteration order.
func MapHash(m pcommon.Map) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	writeMap(&h, m)
	return h.Sum64()
}

func writeMap(h *maphash.Hash, m pcommon.Map) {
	keys := make([]string, 0, m.Len())
	m.Range(func(k string, _ pcommon.Value) bool {
		keys = append(keys, k)
		return true
	})
	sort.Strings(keys)

	var buf [8]byte
	for _, k := range keys {
		v, _ := m.Get(k)
		binary.LittleEndian.PutUint64(buf[:], uint64(len(k)))
		h.Write(buf[:])
		h.WriteString(k)
		writeValue(h, &buf, v)
	}
}

func writeValue(h *maphash.Hash, buf *[8]byte, v pcommon.Value) {
	buf[0] = byte(v.Type())
	h.Write(buf[:1])
	switch v.Type() {
	case pcommon.ValueTypeStr:
		s := v.Str()
		binary.LittleEndian.PutUint64(buf[:], uint64(len(s)))
		h.Write(buf[:])
		h.WriteString(s)
	case pcommon.ValueTypeInt:
		binary.LittleEndian.PutUint64(buf[:], uint64(v.Int()))
		h.Write(buf[:])
	case pcommon.ValueTypeDouble:
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v.Double()))
		h.Write(buf[:])
	case pcommon.ValueTypeBool:
		if v.Bool() {
			buf[0] = 1
		} else {
			buf[0] = 0
		}
		h.Write(buf[:1])
	case pcommon.ValueTypeBytes:
		b := v.Bytes().AsRaw()
		binary.LittleEndian.PutUint64(buf[:], uint64(len(b)))
		h.Write(buf[:])
		h.Write(b)
	case pcommon.ValueTypeMap:
		writeMap(h, v.Map())
	case pcommon.ValueTypeSlice:
		s := v.Slice()
		binary.LittleEndian.PutUint64(buf[:], uint64(s.Len()))
		h.Write(buf[:])
		for i := 0; i < s.Len(); i++ {
			writeValue(h, buf, s.At(i))
		}
	}
}
