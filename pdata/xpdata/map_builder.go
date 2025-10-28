// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata // import "go.opentelemetry.io/collector/pdata/xpdata"

import (
	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// MapBuilder is an experimental struct which can be used to create a pcommon.Map more efficiently
// than by repeated use of the Put family of methods, which check for duplicate keys on every call
// (a linear time operation).
// A zero-initialized MapBuilder is ready for use.
type MapBuilder struct {
	state internal.State
	pairs []internal.KeyValue
}

// EnsureCapacity increases the capacity of this MapBuilder instance, if necessary,
// to ensure that it can hold at least the number of elements specified by the capacity argument.
func (mb *MapBuilder) EnsureCapacity(capacity int) {
	oldValues := mb.pairs
	if capacity <= cap(oldValues) {
		return
	}
	mb.pairs = make([]internal.KeyValue, len(oldValues), capacity)
	copy(mb.pairs, oldValues)
}

func (mb *MapBuilder) getValue(i int) pcommon.Value {
	return pcommon.Value(internal.NewValueWrapper(&mb.pairs[i].Value, &mb.state))
}

// AppendEmpty appends a key/value pair to the MapBuilder and return the inserted value.
// This method does not check for duplicate keys and has an amortized constant time complexity.
func (mb *MapBuilder) AppendEmpty(k string) pcommon.Value {
	mb.pairs = append(mb.pairs, internal.KeyValue{Key: k})
	return mb.getValue(len(mb.pairs) - 1)
}

// UnsafeIntoMap transfers the contents of a MapBuilder into a MapWrapper, without checking for duplicate keys.
// If the MapBuilder contains duplicate keys, the behavior of the resulting MapWrapper is unspecified.
// This operation has constant time complexity and makes no allocations.
// After this operation, the MapBuilder is reset to an empty state.
func (mb *MapBuilder) UnsafeIntoMap(m pcommon.Map) {
	internal.GetMapState(internal.MapWrapper(m)).AssertMutable()
	*internal.GetMapOrig(internal.MapWrapper(m)) = mb.pairs
	mb.pairs = nil
}
