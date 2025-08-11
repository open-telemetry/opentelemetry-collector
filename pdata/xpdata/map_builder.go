package xpdata

import (
	"cmp"
	"errors"
	"slices"

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

func compareKeyValues(kv1 otlpcommon.KeyValue, kv2 otlpcommon.KeyValue) int {
	return cmp.Compare(kv1.Key, kv2.Key)
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

// SortedIntoMap transfers the contents of a MapBuilder into a Map, checking that the keys are all distinct,
// and in alphabetical order.
// If this check fails, the call will return an error without modifying the Map.
// This operation has linear time complexity and makes no allocations.
// After this operation, the MapBuilder becomes read-only.
func (mb *MapBuilder) SortedIntoMap(m pcommon.Map) error {
	for i := range len(mb.pairs) - 1 {
		if mb.pairs[i].Key >= mb.pairs[i+1].Key {
			return errors.New("Keys added to MapBuilder are not in strictly ascending order")
		}
	}
	mb.UnsafeIntoMap(m)
	return nil
}

// SortedIntoMap transfers the contents of a MapBuilder into a Map, checking that the keys are all distinct.
// If this check fails, the call will return an error without modifying the Map.
// This operation will sort the keys, and has an O(n log n) time complexity.
// After this operation, the MapBuilder becomes read-only.
func (mb *MapBuilder) DistinctIntoMap(m pcommon.Map) error {
	slices.SortFunc(mb.pairs, compareKeyValues)
	for i := range len(mb.pairs) - 1 {
		if mb.pairs[i].Key == mb.pairs[i+1].Key {
			return errors.New("Keys added to MapBuilder are not distinct")
		}
	}
	mb.UnsafeIntoMap(m)
	return nil
}

// SortedIntoMap transfers the contents of a MapBuilder into a Map, merging values with the same key with a
// user-defined `merge` function.
// `merge` will be called with a slice `vals` of at least 2 values, and must set `vals[0]` as the merged output.
// This operation will sort the keys, and has an O(n log n) time complexity (assuming `merge` is linear).
// After this operation, the MapBuilder becomes read-only.
func (mb *MapBuilder) MergeIntoMap(m pcommon.Map, merge func(vals []pcommon.Value)) {
	if len(mb.pairs) > 0 {
		slices.SortFunc(mb.pairs, compareKeyValues)
		groupStart := 0
		groupEnd := 1
		groupKey := mb.pairs[0].Key
		for {
			if groupEnd < len(mb.pairs) && mb.pairs[groupEnd].Key == groupKey {
				groupEnd++
				continue
			}
			groupSize := groupEnd - groupStart
			if groupSize > 1 {
				vals := make([]pcommon.Value, groupSize)
				for i := range groupSize {
					vals[i] = mb.getValue(groupStart + i)
				}
				// We expect the result to be placed in vals[0]
				merge(vals)
				mb.pairs = slices.Delete(mb.pairs, groupStart+1, groupEnd)
				groupEnd = groupStart + 1
			}
			if groupEnd == len(mb.pairs) {
				break
			}
			groupStart = groupEnd
			groupEnd = groupEnd + 1
			groupKey = mb.pairs[groupStart].Key
		}
	}
	mb.UnsafeIntoMap(m)
}
