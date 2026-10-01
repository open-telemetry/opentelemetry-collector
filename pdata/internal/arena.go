// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"reflect"
	"unsafe"

	"go.opentelemetry.io/collector/pdata/internal/metadata"
)

const (
	// payloadChunkSize is the default size of byte chunks used for wire-adjacent
	// payload (SetName / CopyString / new []byte). Large values get a dedicated chunk.
	payloadChunkSize = 16 << 10
	// graphChunkBytes is the target size of each typed graph block (messages, slice backings).
	graphChunkBytes = 2 << 20
)

// Arena is a request-scoped allocator owned by State.
//
// Two regions:
//   - payload: 16KB (or larger) []byte chunks for strings/bytes
//   - graph: typed 2MB-class slabs for pointerful proto structs and slice backings
//
// The protobuf wire buffer is retained separately and released with the arena.
// Go GC owns *Arena; dropping the State arena releases every chunk together.
type Arena struct {
	wire []byte

	payload      [][]byte
	payloadChunk int
	payloadOff   int

	slabs  map[reflect.Type]any
	resets []func()

	// heapRefs roots any Go heap object that could not be placed in a chunk.
	heapRefs []any
}

func newArena() *Arena {
	return &Arena{slabs: make(map[reflect.Type]any)}
}

func (a *Arena) reset() {
	a.wire = nil
	a.payloadChunk = 0
	a.payloadOff = 0
	a.heapRefs = a.heapRefs[:0]
	for _, r := range a.resets {
		r()
	}
}

func (a *Arena) retainWire(buf []byte) {
	a.wire = buf
}

func (a *Arena) allocPayload(n int) []byte {
	if n == 0 {
		return nil
	}
	if n >= payloadChunkSize {
		b := make([]byte, n)
		a.payload = append(a.payload, b)
		return b
	}
	for {
		if a.payloadChunk < len(a.payload) {
			cur := a.payload[a.payloadChunk]
			if a.payloadOff+n <= len(cur) {
				sl := cur[a.payloadOff : a.payloadOff+n]
				a.payloadOff += n
				return sl
			}
			a.payloadChunk++
			a.payloadOff = 0
			continue
		}
		a.payload = append(a.payload, make([]byte, payloadChunkSize))
		a.payloadChunk = len(a.payload) - 1
		a.payloadOff = 0
	}
}

type typedSlab[T any] struct {
	blocks [][]T
	b, i   int
}

// Alloc returns a zero T from st's graph arena, or a heap allocation when no arena is attached.
func Alloc[T any](st *State) *T {
	if st == nil || st.arena == nil {
		return new(T)
	}
	return alloc[T](st.arena)
}

func alloc[T any](a *Arena) *T {
	rt := reflect.TypeFor[*T]()
	s, ok := a.slabs[rt].(*typedSlab[T])
	if !ok {
		s = &typedSlab[T]{}
		a.slabs[rt] = s
		a.resets = append(a.resets, s.reset)
	}
	return s.next()
}

func (s *typedSlab[T]) next() *T {
	if len(s.blocks) == 0 {
		s.grow()
	}
	if s.i >= len(s.blocks[s.b]) {
		if s.b+1 < len(s.blocks) {
			s.b++
			s.i = 0
		} else {
			s.grow()
		}
	}
	p := &s.blocks[s.b][s.i]
	var zero T
	*p = zero
	s.i++
	return p
}

func (s *typedSlab[T]) grow() {
	maxN := graphLen[T]()
	n := 32
	if len(s.blocks) > 0 {
		n = min(len(s.blocks[len(s.blocks)-1])*2, maxN)
		if n <= len(s.blocks[len(s.blocks)-1]) {
			n = maxN
		}
	}
	s.blocks = append(s.blocks, make([]T, n))
	s.b = len(s.blocks) - 1
	s.i = 0
}

func (s *typedSlab[T]) reset() {
	for bi, block := range s.blocks {
		if bi < s.b {
			clear(block)
			continue
		}
		if bi == s.b {
			clear(block[:min(len(block), s.i)])
			continue
		}
		clear(block)
	}
	s.b, s.i = 0, 0
}

func graphLen[T any]() int {
	sz := int(unsafe.Sizeof(*new(T)))
	if sz < 1 {
		sz = 1
	}
	n := graphChunkBytes / sz
	if n < 32 {
		return 32
	}
	return n
}

type sliceSlab[T any] struct {
	blocks [][]T
	b, off int
}

// AllocSlice returns a slice backed by st's graph arena when present.
func AllocSlice[T any](st *State, length, capacity int) []T {
	if capacity < length {
		capacity = length
	}
	if capacity == 0 {
		return nil
	}
	if st == nil || st.arena == nil {
		return make([]T, length, capacity)
	}
	return allocSlice[T](st.arena, length, capacity)
}

func allocSlice[T any](a *Arena, length, capacity int) []T {
	rt := reflect.TypeFor[[]T]()
	s, ok := a.slabs[rt].(*sliceSlab[T])
	if !ok {
		s = &sliceSlab[T]{}
		a.slabs[rt] = s
		a.resets = append(a.resets, s.reset)
	}
	return s.next(length, capacity)
}

func (s *sliceSlab[T]) next(length, capacity int) []T {
	for {
		cur := s.cur()
		if s.off+capacity <= len(cur) {
			sl := cur[s.off : s.off+length : s.off+capacity]
			clear(sl)
			s.off += capacity
			return sl
		}
		if s.b+1 < len(s.blocks) {
			s.b++
			s.off = 0
			continue
		}
		n := max(capacity, 16)
		if len(cur) > 0 {
			n = min(max(n, len(cur)*2), max(capacity, graphLen[T]()))
		}
		s.blocks = append(s.blocks, make([]T, n))
		s.b = len(s.blocks) - 1
		s.off = 0
	}
}

func (s *sliceSlab[T]) cur() []T {
	if len(s.blocks) == 0 {
		return nil
	}
	return s.blocks[s.b]
}

func (s *sliceSlab[T]) reset() {
	for bi, block := range s.blocks {
		if bi < s.b {
			clear(block)
			continue
		}
		if bi == s.b {
			clear(block[:min(len(block), s.off)])
			continue
		}
		clear(block)
	}
	s.b, s.off = 0, 0
}

func growCap(need int) int {
	n := 1
	for n < need {
		n *= 2
	}
	return n
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
	if st == nil || st.arena == nil || v == nil {
		return
	}
	st.arena.heapRefs = append(st.arena.heapRefs, v)
}

// BorrowString returns a string pointing into buf when the arena retains the wire buffer.
func BorrowString(st *State, buf []byte, start, end int) string {
	if start == end {
		return ""
	}
	if st == nil || st.arena == nil || st.arena.wire == nil {
		return internPayloadString(st, buf[start:end])
	}
	return unsafe.String(&buf[start], end-start)
}

// BorrowBytes returns a []byte pointing into buf when the arena retains the wire buffer.
func BorrowBytes(st *State, buf []byte, start, end int) []byte {
	if start == end {
		return nil
	}
	if st == nil || st.arena == nil || st.arena.wire == nil {
		return internPayloadBytes(st, buf[start:end])
	}
	return buf[start:end]
}

func internPayloadString(st *State, src []byte) string {
	if len(src) == 0 {
		return ""
	}
	if st == nil || st.arena == nil {
		return string(src)
	}
	dst := st.arena.allocPayload(len(src))
	copy(dst, src)
	return unsafe.String(&dst[0], len(dst))
}

func internPayloadBytes(st *State, src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	if st == nil || st.arena == nil {
		nb := make([]byte, len(src))
		copy(nb, src)
		return nb
	}
	dst := st.arena.allocPayload(len(src))
	copy(dst, src)
	return dst
}

func (st *State) aliasesArenaBytes(b []byte) bool {
	if st == nil || st.arena == nil || len(b) == 0 {
		return false
	}
	p := uintptr(unsafe.Pointer(unsafe.SliceData(b)))
	n := len(b)
	if len(st.arena.wire) > 0 && aliasesPtr(st.arena.wire, p, n) {
		return true
	}
	for _, chunk := range st.arena.payload {
		if aliasesPtr(chunk, p, n) {
			return true
		}
	}
	return false
}

func (st *State) aliasesArenaString(s string) bool {
	if st == nil || st.arena == nil || s == "" {
		return false
	}
	p := uintptr(unsafe.Pointer(unsafe.StringData(s)))
	n := len(s)
	if len(st.arena.wire) > 0 && aliasesPtr(st.arena.wire, p, n) {
		return true
	}
	for _, chunk := range st.arena.payload {
		if aliasesPtr(chunk, p, n) {
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
	if st == nil || st.arena == nil || len(st.arena.wire) == 0 || len(b) == 0 {
		return false
	}
	return aliasesPtr(st.arena.wire, uintptr(unsafe.Pointer(unsafe.SliceData(b))), len(b))
}

func useProtoArena() bool {
	return metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
}
