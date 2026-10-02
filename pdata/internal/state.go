// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"sync/atomic"
	"unsafe"

	"go.opentelemetry.io/collector/pdata/internal/metadata"
	"go.opentelemetry.io/collector/pdata/internal/proto"
)

// State defines an ownership state of pmetric.Metrics, plog.Logs, ptrace.Traces or pprofile.Profiles.
type State struct {
	refs  atomic.Int32
	state uint32

	// arenas holds every arena used by this request. The last one is current.
	arenas []*Arena
	wire   []byte
}

const (
	defaultState          uint32 = 0
	stateReadOnlyBit             = uint32(1 << 0)
	statePipelineOwnedBit        = uint32(1 << 1)
)

func NewState() *State {
	st := &State{
		state: defaultState,
	}
	st.refs.Store(1)
	if useProtoArena() {
		st.arenas = []*Arena{getArena(chunkSize)}
	}
	return st
}

func useProtoArena() bool {
	return metadata.PdataUseProtoPoolingFeatureGate.IsEnabled()
}

func (st *State) arena() *Arena {
	return st.arenas[len(st.arenas)-1]
}

// newArena appends a pooled arena that can hold size bytes and makes it current.
func (st *State) newArena(size int) *Arena {
	a := getArena(max(chunkSize, size))
	st.arenas = append(st.arenas, a)
	return a
}

// Alloc returns a zero T from st's current arena, or a heap allocation when no arena is attached.
// When the arena is full, st takes a new arena and allocates again.
func Alloc[T any](st *State) *T {
	if st == nil || len(st.arenas) == 0 {
		return new(T)
	}
	out, err := arenaAlloc[T](st.arena())
	if err == nil {
		return out
	}
	var zero T
	out, _ = arenaAlloc[T](st.newArena(int(unsafe.Sizeof(zero) + unsafe.Alignof(zero))))
	return out
}

// AllocSlice returns a zeroed slice from st's current arena, or a heap slice when no arena is attached.
// When the arena is full, st takes a new arena and allocates again.
func AllocSlice[T any](st *State, length, capacity int) []T {
	if capacity < length {
		capacity = length
	}
	if capacity == 0 {
		return nil
	}
	var zero T
	if st == nil || len(st.arenas) == 0 || unsafe.Sizeof(zero) == 0 {
		return make([]T, length, capacity)
	}
	s, err := arenaAllocSlice[T](st.arena(), length, capacity)
	if err == nil {
		return s
	}
	s, _ = arenaAllocSlice[T](st.newArena(int(unsafe.Sizeof(zero))*capacity+int(unsafe.Alignof(zero))), length, capacity)
	return s
}

// allocBytes returns n bytes from st's current arena. When the arena is full, st takes a new arena and allocates again.
func (st *State) allocBytes(n int) []byte {
	b, err := st.arena().allocBytes(n)
	if err == nil {
		return b
	}
	b, _ = st.newArena(n).allocBytes(n)
	return b
}

// RetainWire keeps the protobuf input buffer alive so string/[]byte fields may alias it.
func (st *State) RetainWire(buf []byte) {
	if st == nil || len(st.arenas) == 0 {
		return
	}
	st.wire = buf
}

// CloneAndRetainWire copies buf and retains the copy when arenas are attached.
func (st *State) CloneAndRetainWire(buf []byte) []byte {
	if st == nil || len(st.arenas) == 0 {
		return buf
	}
	owned := append([]byte(nil), buf...)
	st.wire = owned
	return owned
}

// ResetArena keeps the first arena for the next request and returns the rest to the pool.
func (st *State) ResetArena() {
	if st == nil {
		return
	}
	if len(st.arenas) > 0 {
		for _, a := range st.arenas[1:] {
			a.Release()
		}
		st.arenas[0].Reset()
		st.arenas = st.arenas[:1]
	}
	st.wire = nil
}

// DropArena returns every arena on st to the pool.
func (st *State) DropArena() {
	if st == nil {
		return
	}
	for _, a := range st.arenas {
		a.Release()
	}
	st.arenas = nil
	st.wire = nil
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

// Append appends v to s using arena backing when an arena is attached.
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

// AppendSeq appends src onto dst using arena backing when needed.
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

// CopyStringSlice copies src strings into dest arena storage.
func CopyStringSlice(st *State, dst, src []string) []string {
	out := CopySlice(st, dst, src)
	for i, s := range out {
		out[i] = CopyString(st, s)
	}
	return out
}

// CopySlice copies src into dest arena memory, reusing dst cap when possible.
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
	dst := st.allocBytes(len(src))
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
	dst := st.allocBytes(len(src))
	copy(dst, src)
	return dst
}

// CopyString clones s into dest arena unless it already aliases dest wire/arena memory.
func CopyString(st *State, s string) string {
	if s == "" {
		return s
	}
	if !useProtoArena() {
		return s
	}
	if st != nil && st.aliases(unsafe.Pointer(unsafe.StringData(s)), len(s)) {
		return s
	}
	return internPayloadString(st, unsafe.Slice(unsafe.StringData(s), len(s)))
}

// CopyBytes clones b into dest arena unless it already aliases dest wire/arena memory.
func CopyBytes(st *State, b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	if !useProtoArena() {
		nb := make([]byte, len(b))
		copy(nb, b)
		return nb
	}
	if st != nil && st.aliases(unsafe.Pointer(unsafe.SliceData(b)), len(b)) {
		return b
	}
	return internPayloadBytes(st, b)
}

// CopyOnWriteBytes copies b into arena memory if it aliases the retained wire buffer.
func (st *State) CopyOnWriteBytes(b *[]byte) {
	if st == nil || len(*b) == 0 {
		return
	}
	if !aliasesPtr(st.wire, uintptr(unsafe.Pointer(unsafe.SliceData(*b))), len(*b)) {
		return
	}
	*b = internPayloadBytes(st, *b)
}

// aliases reports whether the n bytes at p are in st's wire buffer or one of its arenas.
func (st *State) aliases(p unsafe.Pointer, n int) bool {
	if len(st.arenas) == 0 || n == 0 {
		return false
	}
	if aliasesPtr(st.wire, uintptr(p), n) {
		return true
	}
	for _, a := range st.arenas {
		if a.Contains(p, n) {
			return true
		}
	}
	return false
}

func (st *State) MarkReadOnly() {
	st.state |= stateReadOnlyBit
}

func (st *State) IsReadOnly() bool {
	return st.state&stateReadOnlyBit != 0
}

// AssertMutable panics if the state is not StateMutable.
func (st *State) AssertMutable() {
	if st.state&stateReadOnlyBit != 0 {
		panic("invalid access to shared data")
	}
}

// MarkPipelineOwned marks the data as owned by the pipeline, returns true if the data were
// previously not owned by the pipeline, otherwise false.
func (st *State) MarkPipelineOwned() bool {
	if st.state&statePipelineOwnedBit != 0 {
		return false
	}
	st.state |= statePipelineOwnedBit
	return true
}

// Ref add one to the count of active references.
func (st *State) Ref() {
	st.refs.Add(1)
}

// Unref returns true if reference count got to 0 which means no more active references,
// otherwise it returns false.
func (st *State) Unref() bool {
	refs := st.refs.Add(-1)
	switch {
	case refs > 0:
		return false
	case refs == 0:
		return true
	default:
		panic("Cannot unref freed data")
	}
}
