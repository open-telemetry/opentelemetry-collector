// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"sync"
	"unsafe"

	"go.opentelemetry.io/collector/pdata/internal/metadata"
	"go.opentelemetry.io/collector/pdata/internal/proto"
)

// chunkSize is the single raw buffer owned by an arena.
const chunkSize = 1 << 20

// Arena is one raw buffer and a bump offset. Every type is carved from buf.
// A request that does not fit takes another Arena on its State.
type Arena struct {
	buf []byte
	off int
}

// arenaPool reuses arenas, including their raw buffers, across requests.
var arenaPool = sync.Pool{
	New: func() any {
		return &Arena{}
	},
}

func getArena() *Arena {
	a := arenaPool.Get().(*Arena)
	a.off = 0
	if len(a.buf) < chunkSize {
		a.buf = make([]byte, chunkSize)
	}
	return a
}

var bumpZero byte

func (a *Arena) tryBump(size, align int) (unsafe.Pointer, bool) {
	if len(a.buf) == 0 || size <= 0 {
		return nil, false
	}
	if align <= 0 {
		align = 1
	}
	off := (a.off + align - 1) &^ (align - 1)
	if off+size > len(a.buf) {
		return nil, false
	}
	a.off = off + size
	return unsafe.Pointer(unsafe.SliceData(a.buf[off : off+size])), true
}

func (st *State) bump(size, align int) unsafe.Pointer {
	if size <= 0 {
		return unsafe.Pointer(&bumpZero)
	}
	if align <= 0 {
		align = 1
	}
	for {
		if st.ai < len(st.arenas) {
			if p, ok := st.arenas[st.ai].tryBump(size, align); ok {
				return p
			}
			st.ai++
			continue
		}
		a := getArena()
		if size+align > len(a.buf) {
			a.buf = make([]byte, size+align)
			a.off = 0
		}
		st.arenas = append(st.arenas, a)
		st.ai = len(st.arenas) - 1
	}
}

// Alloc returns a zero T carved from st's raw buffer, or a heap allocation when no arena is attached.
func Alloc[T any](st *State) *T {
	if st == nil || len(st.arenas) == 0 {
		return new(T)
	}
	return alloc[T](st)
}

func alloc[T any](st *State) *T {
	var zero T
	size := int(unsafe.Sizeof(zero))
	align := int(unsafe.Alignof(zero))
	p := st.bump(size, align)
	out := (*T)(p)
	*out = zero
	return out
}

// AllocSlice returns a slice backed by st's graph arena when present.
func AllocSlice[T any](st *State, length, capacity int) []T {
	if capacity < length {
		capacity = length
	}
	if capacity == 0 {
		return nil
	}
	if st == nil || len(st.arenas) == 0 {
		return make([]T, length, capacity)
	}
	return allocSlice[T](st, length, capacity)
}

func allocSlice[T any](st *State, length, capacity int) []T {
	var zero T
	elem := int(unsafe.Sizeof(zero))
	align := int(unsafe.Alignof(zero))
	if elem == 0 {
		return make([]T, length, capacity)
	}
	p := st.bump(elem*capacity, align)
	s := unsafe.Slice((*T)(p), capacity)
	clear(s)
	return s[:length:capacity]
}

func growCap(need int) int {
	n := 1
	for n < need {
		n *= 2
	}
	return n
}

// appendCountLimit is the largest remainder we scan to size a repeated field exactly.
// A bigger message doubles capacity instead; walking a 10MB parent costs more than the copies.
const appendCountLimit = 4096

// AppendCounted appends v like Append. The first element of a repeated field is
// sized to the number of remaining occurrences of fieldNum in buf[pos:], so later
// elements fill that slice instead of growing and copying.
func AppendCounted[T any](st *State, s []T, v T, buf []byte, pos int, fieldNum int32) []T {
	if cap(s) > len(s) {
		s = s[:len(s)+1]
		s[len(s)-1] = v
		return s
	}
	n := len(s) + 1
	if len(s) == 0 && len(buf)-pos <= appendCountLimit {
		n += proto.CountField(buf, pos, fieldNum)
	} else {
		n = growCap(n)
	}
	ns := AllocSlice[T](st, len(s)+1, n)
	copy(ns, s)
	ns[len(s)] = v
	return ns
}

// Append appends v to s using graph-arena backing when an arena is attached.
func Append[T any](st *State, s []T, v T) []T {
	if cap(s) > len(s) {
		s = s[:len(s)+1]
		s[len(s)-1] = v
		return s
	}
	ns := AllocSlice[T](st, len(s)+1, growCap(len(s)+1))
	copy(ns, s)
	ns[len(s)] = v
	return ns
}

// AppendSeq appends src onto dst using graph-arena backing when needed.
func AppendSeq[T any](st *State, dst, src []T) []T {
	if len(src) == 0 {
		return dst
	}
	if cap(dst)-len(dst) >= len(src) {
		return append(dst, src...)
	}
	ns := AllocSlice[T](st, len(dst)+len(src), growCap(len(dst)+len(src)))
	copy(ns, dst)
	copy(ns[len(dst):], src)
	return ns
}

// CopyStringSlice copies src strings into dest payload-backed storage.
func CopyStringSlice(st *State, dst, src []string) []string {
	out := CopySlice(st, dst, src)
	for i, s := range out {
		out[i] = CopyString(st, s)
	}
	return out
}

// CopySlice copies src into dest-arena memory, reusing dst cap when possible.
func CopySlice[T any](st *State, dst, src []T) []T {
	if len(src) == 0 {
		if dst == nil {
			return nil
		}
		return dst[:0]
	}
	if cap(dst) >= len(src) {
		dst = dst[:len(src)]
		copy(dst, src)
		return dst
	}
	ns := AllocSlice[T](st, len(src), len(src))
	copy(ns, src)
	return ns
}

// KeepRef roots a Go heap object on the arena until reset/drop.
func KeepRef(st *State, v any) {
	if st == nil || len(st.arenas) == 0 || v == nil {
		return
	}
	st.heapRefs = append(st.heapRefs, v)
}

// BorrowString returns a string pointing into buf when the state retains the wire buffer.
func BorrowString(st *State, buf []byte, start, end int) string {
	if start == end {
		return ""
	}
	if st == nil || len(st.arenas) == 0 || st.wire == nil {
		return internPayloadString(st, buf[start:end])
	}
	return unsafe.String(&buf[start], end-start)
}

// BorrowBytes returns a []byte pointing into buf when the state retains the wire buffer.
func BorrowBytes(st *State, buf []byte, start, end int) []byte {
	if start == end {
		return nil
	}
	if st == nil || len(st.arenas) == 0 || st.wire == nil {
		return internPayloadBytes(st, buf[start:end])
	}
	return buf[start:end]
}

func internPayloadString(st *State, src []byte) string {
	if len(src) == 0 {
		return ""
	}
	if st == nil || len(st.arenas) == 0 {
		return string(src)
	}
	dst := st.allocPayload(len(src))
	copy(dst, src)
	return unsafe.String(&dst[0], len(dst))
}

func internPayloadBytes(st *State, src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	if st == nil || len(st.arenas) == 0 {
		nb := make([]byte, len(src))
		copy(nb, src)
		return nb
	}
	dst := st.allocPayload(len(src))
	copy(dst, src)
	return dst
}

func (st *State) allocPayload(n int) []byte {
	if n == 0 {
		return nil
	}
	p := st.bump(n, 1)
	return unsafe.Slice((*byte)(p), n)
}

func (st *State) aliasesArenaBytes(b []byte) bool {
	if st == nil || len(st.arenas) == 0 || len(b) == 0 {
		return false
	}
	p := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	n := len(b)
	if len(st.wire) > 0 && aliasesPtr(st.wire, p, n) {
		return true
	}
	for _, a := range st.arenas {
		if aliasesPtr(a.buf, p, n) {
			return true
		}
	}
	return false
}

func (st *State) aliasesArenaString(s string) bool {
	if st == nil || len(st.arenas) == 0 || s == "" {
		return false
	}
	p := uintptr(unsafe.Pointer(unsafe.StringData(s)))
	n := len(s)
	if len(st.wire) > 0 && aliasesPtr(st.wire, p, n) {
		return true
	}
	for _, a := range st.arenas {
		if aliasesPtr(a.buf, p, n) {
			return true
		}
	}
	return false
}

func aliasesPtr(buf []byte, p uintptr, n int) bool {
	if len(buf) == 0 {
		return false
	}
	wp := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))
	return p >= wp && p+uintptr(n) <= wp+uintptr(len(buf))
}

// CopyString clones s into dest payload unless it already aliases dest wire/payload.
func CopyString(st *State, s string) string {
	if s == "" {
		return s
	}
	if !useProtoArena() {
		return s
	}
	if st != nil && st.aliasesArenaString(s) {
		return s
	}
	return internPayloadString(st, unsafe.Slice(unsafe.StringData(s), len(s)))
}

// CopyBytes clones b into dest payload unless it already aliases dest wire/payload.
func CopyBytes(st *State, b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	if !useProtoArena() {
		nb := make([]byte, len(b))
		copy(nb, b)
		return nb
	}
	if st != nil && st.aliasesArenaBytes(b) {
		return b
	}
	return internPayloadBytes(st, b)
}

// CopyOnWriteBytes copies b into payload memory if it aliases the retained wire buffer.
func (st *State) CopyOnWriteBytes(b *[]byte) {
	if st == nil || len(*b) == 0 {
		return
	}
	if !st.aliasesWire(*b) {
		return
	}
	*b = internPayloadBytes(st, *b)
}

func (st *State) aliasesWire(b []byte) bool {
	if st == nil || len(st.wire) == 0 || len(b) == 0 {
		return false
	}
	return aliasesPtr(st.wire, uintptr(unsafe.Pointer(unsafe.SliceData(b))), len(b))
}

func useProtoArena() bool {
	return metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
}
