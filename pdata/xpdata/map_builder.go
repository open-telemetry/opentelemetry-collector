package xpdata

import (
	"go.opentelemetry.io/collector/pdata/internal"
	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// MapBuilder is an experimental struct which can be used to create a pcommon.Map more efficiently
// than by repeated use of the Put family of methods, which check for duplicate keys on every call
// (a linear time operation).
// A zero-initialized MapBuilder is ready for use.
type MapBuilder struct {
	state internal.State
	pairs []otlpcommon.KeyValue
}

// EnsureCapacity increases the capacity of this MapBuilder instance, if necessary,
// to ensure that it can hold at least the number of elements specified by the capacity argument.
func (mb *MapBuilder) EnsureCapacity(capacity int) {
	mb.state.AssertMutable()
	oldValues := mb.pairs
	if capacity <= cap(oldValues) {
		return
	}
	mb.pairs = make([]otlpcommon.KeyValue, len(oldValues), capacity)
	copy(mb.pairs, oldValues)
}

func (mb *MapBuilder) getValue(i int) pcommon.Value {
	return pcommon.Value(internal.NewValue(&mb.pairs[i].Value, &mb.state))
}

// AppendEmpty appends a key/value pair to the MapBuilder and return the inserted value.
// This method does not check for duplicate keys and has an amortized constant time complexity.
func (mb *MapBuilder) AppendEmpty(k string) pcommon.Value {
	mb.state.AssertMutable()
	mb.pairs = append(mb.pairs, otlpcommon.KeyValue{Key: k})
	return mb.getValue(len(mb.pairs) - 1)
}

// UnsafeIntoMap transfers the contents of a MapBuilder into a Map, without checking for duplicate keys.
// If the MapBuilder contains duplicate keys, the behavior of the resulting Map is unspecified;
// consider using DistinctIntoMap if you are unsure or performance is not a concern.
// This operation has constant time complexity and makes no allocations.
// After this operation, the MapBuilder becomes read-only.
func (mb *MapBuilder) UnsafeIntoMap(m pcommon.Map) {
	mb.state.AssertMutable()
	internal.GetMapState(internal.Map(m)).AssertMutable()
	mb.state = internal.StateReadOnly // to avoid modifying a Map later marked as ReadOnly through builder Values
	*internal.GetOrigMap(internal.Map(m)) = mb.pairs
}
