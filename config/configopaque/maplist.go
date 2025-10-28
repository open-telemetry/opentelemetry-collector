// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configopaque // import "go.opentelemetry.io/collector/config/configopaque"

import (
	"cmp"
	"fmt"
	"iter"
	"slices"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

// Pair is an element of a MapList, and consists of a name and an opaque value.
type Pair struct {
	Name  string `mapstructure:"name"`
	Value String `mapstructure:"value"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// MapList is a replacement for map[string]configopaque.String with a similar API,
// which can also be unmarshalled from (and is stored as) a list of name/value pairs.
//
// Pairs are assumed to have distinct names. This is checked during config validation.
type MapList []Pair

var _ confmap.Unmarshaler = (*MapList)(nil)

// Unmarshal is called by the Collector when unmarshalling from a map.
// When the input config is a slice, this will be skipped,
// and mapstructure's default unmarshalling logic will be used.
func (ml *MapList) Unmarshal(conf *confmap.Conf) error {
	var m2 map[string]String
	if err := conf.Unmarshal(&m2); err != nil {
		return err
	}
	*ml = make(MapList, 0, len(m2))
	for name, value := range m2 {
		*ml = append(*ml, Pair{
			Name:  name,
			Value: value,
		})
	}
	slices.SortFunc(*ml, func(p1, p2 Pair) int {
		return cmp.Compare(p1.Name, p2.Name)
	})
	return nil
}

var _ xconfmap.Validator = MapList(nil)

func (ml MapList) Validate() error {
	// Check for duplicate keys
	counts := make(map[string]int, len(ml))
	for _, OpaquePair := range ml {
		counts[OpaquePair.Name]++
	}
	if len(counts) == len(ml) {
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

var _ iter.Seq2[string, String] = MapList(nil).Iter

// Iter is an iterator over key/value pairs for use in for-range loops.
// It is the MapList equivalent of directly ranging over a map.
func (ml MapList) Iter(yield func(name string, value String) bool) {
	for _, OpaquePair := range ml {
		if !yield(OpaquePair.Name, OpaquePair.Value) {
			break
		}
	}
}

// Get looks up a pair's value based on its name.
// It is the MapList equivalent of `val, ok := m[key]`.
// However, it has linear time complexity.
func (ml MapList) Get(name string) (val String, ok bool) {
	for _, OpaquePair := range ml {
		if OpaquePair.Name == name {
			return OpaquePair.Value, true
		}
	}
	return val, false
}

// Set sets the value corresponding to a given name.
// It is the MapList equivalent of `m[key] = val`.
// However, it has linear time complexity,
// and does not affect shallow copies.
func (ml *MapList) Set(name string, val String) {
	if ml == nil {
		panic("assignment to entry in nil *MapList")
	}
	for i, OpaquePair := range *ml {
		if OpaquePair.Name == name {
			*ml = slices.Clone(*ml)
			(*ml)[i].Value = val
			return
		}
	}
	*ml = append(make(MapList, 0, len(*ml)+1), *ml...)
	*ml = append(*ml, Pair{Name: name, Value: val})
}
