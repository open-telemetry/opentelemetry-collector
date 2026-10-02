// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"errors"
	"sync"
	"unsafe"
)

// chunkSize is the size of the raw buffer owned by a new arena.
const chunkSize = 2 << 20

var errArenaFull = errors.New("arena full")

var zeroByte byte

// Arena is one raw buffer and a bump offset. Every type is carved from buf.
type Arena struct {
	buf []byte
	off int
}

var arenaPool = sync.Pool{
	New: func() any {
		return &Arena{buf: make([]byte, chunkSize)}
	},
}

// getArena returns a pooled arena whose buffer holds at least size bytes.
func getArena(size int) *Arena {
	a := arenaPool.Get().(*Arena)
	if len(a.buf) < size {
		a.buf = make([]byte, size)
	}
	a.off = 0
	return a
}

// alloc carves size bytes aligned to align from the arena buffer.
// It returns errArenaFull and leaves the arena unchanged when they do not fit.
func (a *Arena) alloc(size, align int) (unsafe.Pointer, error) {
	if size <= 0 {
		return unsafe.Pointer(&zeroByte), nil
	}
	off := (a.off + align - 1) &^ (align - 1)
	if off+size > len(a.buf) {
		return nil, errArenaFull
	}
	a.off = off + size
	return unsafe.Pointer(&a.buf[off]), nil
}

// arenaAlloc returns a zero T carved from a, or errArenaFull.
// Go methods cannot have type parameters, so a is passed explicitly.
func arenaAlloc[T any](a *Arena) (*T, error) {
	var zero T
	p, err := a.alloc(int(unsafe.Sizeof(zero)), int(unsafe.Alignof(zero)))
	if err != nil {
		return nil, err
	}
	out := (*T)(p)
	*out = zero
	return out, nil
}

// arenaAllocSlice returns a []T with the given length and capacity carved from a, or errArenaFull.
// Only the live prefix is zeroed. Spare capacity keeps whatever the pooled buffer held, which is
// safe because every append writes a whole element before anything reads it, and the buffer is a
// []byte that the collector never scans for pointers.
func arenaAllocSlice[T any](a *Arena, length, capacity int) ([]T, error) {
	var zero T
	p, err := a.alloc(int(unsafe.Sizeof(zero))*capacity, int(unsafe.Alignof(zero)))
	if err != nil {
		return nil, err
	}
	s := unsafe.Slice((*T)(p), capacity)
	clear(s[:length])
	return s[:length:capacity], nil
}

// allocBytes returns n bytes carved from a, or errArenaFull.
func (a *Arena) allocBytes(n int) ([]byte, error) {
	p, err := a.alloc(n, 1)
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(p), n), nil
}

// Reset rewinds the arena so its buffer is reused from the start.
func (a *Arena) Reset() {
	a.off = 0
}

// Release resets the arena and returns it to the pool.
func (a *Arena) Release() {
	a.Reset()
	arenaPool.Put(a)
}

// Contains reports whether the n bytes at p lie inside the arena buffer.
func (a *Arena) Contains(p unsafe.Pointer, n int) bool {
	return aliasesPtr(a.buf, uintptr(p), n)
}

// aliasesPtr reports whether the n bytes at p lie entirely inside buf.
func aliasesPtr(buf []byte, p uintptr, n int) bool {
	if len(buf) == 0 {
		return false
	}
	wp := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))
	return p >= wp && p+uintptr(n) <= wp+uintptr(len(buf))
}
