// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package maplist // import "go.opentelemetry.io/collector/internal/maplist"

import (
	"cmp"
	"fmt"
	"iter"
	"slices"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Pair[T any] struct {
	Name  string `mapstructure:"name"`
	Value T      `mapstructure:"value"`
}

// *MapList[T] is a replacement for map[string]T with a similar API,
// which can also be unmarshalled from (and is stored as) a list of name/value pairs.
//
// Pairs are assumed to have distinct names. This is checked during config validation.
//
// Similar to native maps, a nil *MapList[T] is treated the same as an empty one
// for read operations, but write operations will panic.
type MapList[T any] []Pair[T]

// New is the MapList equivalent of `make(map[string]T)`.
func New[T any]() *MapList[T] {
	return new(MapList[T])
}

// WithCapacity is the MapList equivalent of `make(map[string]T, cap)`.
func WithCapacity[T any](capacity int) *MapList[T] {
	ml := new(MapList[T])
	*ml = make([]Pair[T], 0, capacity)
	return ml
}

var _ confmap.Unmarshaler = (*MapList[string])(nil)

// Unmarshal is called by the Collector when unmarshaling from a map.
// When the input config is a slice, this will be skipped,
// and mapstructure's default unmarshalling logic will be used.
func (ml *MapList[T]) Unmarshal(conf *confmap.Conf) error {
	var m2 map[string]T
	if err := conf.Unmarshal(&m2); err != nil {
		return err
	}
	ml.FromMap(m2)
	return nil
}

var _ xconfmap.Validator = (*MapList[string])(nil)

func (ml *MapList[T]) Validate() error {
	if ml == nil {
		return nil
	}

	// Check for duplicate keys
	counts := make(map[string]int, len(*ml))
	for _, pair := range *ml {
		counts[pair.Name]++
	}
	if len(counts) == len(*ml) {
		return nil
	}
	var duplicates []string
	for name, cnt := range counts {
		if cnt > 1 {
			duplicates = append(duplicates, name)
		}
	}
	slices.Sort(duplicates)
	return fmt.Errorf("duplicate keys in map-style list: %v", duplicates)
}

var _ iter.Seq2[string, string] = (*MapList[string])(nil).Iter

// Iter is an iterator over key/value pairs for use in for-range loops.
// It is the MapList equivalent of directly ranging over a map.
func (ml *MapList[T]) Iter(yield func(name string, value T) bool) {
	if ml == nil {
		return
	}
	for _, pair := range *ml {
		if !yield(pair.Name, pair.Value) {
			break
		}
	}
}

// TryGet looks up a pair's value based on its name.
// It is the MapList equivalent of `val, ok := m[key]`.
// However, it has linear time complexity.
func (ml *MapList[T]) TryGet(name string) (val T, ok bool) {
	if ml == nil {
		return val, false
	}
	for _, pair := range *ml {
		if pair.Name == name {
			return pair.Value, true
		}
	}
	return val, false
}

// Get looks up a pair's value based on its name.
// It is the MapList equivalent of `m[key]`.
// However, it has linear time complexity.
func (ml *MapList[T]) Get(name string) T {
	val, _ := ml.TryGet(name)
	return val
}

// Set sets the value corresponding to a given name.
// It is the MapList equivalent of `m[key] = val`.
// However, it has linear time complexity.
func (ml *MapList[T]) Set(name string, val T) {
	if ml == nil {
		panic("assignment to entry in nil MapList")
	}
	for i, pair := range *ml {
		if pair.Name == name {
			(*ml)[i].Value = val
			return
		}
	}
	*ml = append(*ml, Pair[T]{Name: name, Value: val})
}

// Len returns a MapList's length.
// It is the MapList equivalent of `len(m)`.
func (ml *MapList[T]) Len() int {
	if ml == nil {
		return 0
	}
	return len(*ml)
}

// ToMap converts a MapList[T] to a map[string]T.
func (ml *MapList[T]) ToMap() map[string]T {
	if ml == nil {
		return nil
	}
	m := make(map[string]T, len(*ml))
	for _, pair := range *ml {
		m[pair.Name] = pair.Value
	}
	return m
}

// FromMap sets a MapList's contents from an equivalent map.
// The resulting pairs are stored to facilitate comparisons in tests.
func (ml *MapList[T]) FromMap(m map[string]T) {
	*ml = make(MapList[T], 0, len(m))
	for name, value := range m {
		*ml = append(*ml, Pair[T]{
			Name:  name,
			Value: value,
		})
	}
	slices.SortFunc(*ml, func(p1, p2 Pair[T]) int {
		return cmp.Compare(p1.Name, p2.Name)
	})
}

// FromMap converts a map[string]T to a new *MapList[T].
// The resulting pairs are stored to facilitate comparisons in tests.
func FromMap[T any](m map[string]T) *MapList[T] {
	ml := new(MapList[T])
	ml.FromMap(m)
	return ml
}
