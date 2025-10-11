// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package maplist // import "go.opentelemetry.io/collector/internal/maplist"

import (
	"cmp"
	"slices"

	"go.opentelemetry.io/collector/confmap"
)

type Pair[T any] struct {
	Name  string `mapstructure:"name"`
	Value T      `mapstructure:"value"`
}

// MapList[T] is equivalent to []Pair[T],
// but can additionally be unmarshalled from a map.
type MapList[T any] []Pair[T]

var _ confmap.Unmarshaler = (*MapList[string])(nil)

// Unmarshal is called by the Collector when unmarshaling from a map.
// When the input config is a slice, this will be skipped,
// and mapstructure's default unmarshal logic will be used.
func (ml *MapList[T]) Unmarshal(conf *confmap.Conf) error {
	var m2 map[string]T
	if err := conf.Unmarshal(&m2); err != nil {
		return err
	}
	*ml = FromMap(m2)
	return nil
}

// Get looks up the first pair with the given name.
// If one is found, returns its value and true.
// Otherwise, returns a zero value and false.
func (ml MapList[T]) Get(name string) (val T, ok bool) {
	for _, header := range ml {
		if header.Name == name {
			return header.Value, true
		}
	}
	return val, false
}

// ToMap converts a MapList[T] to a map[string]T.
// In the presence of duplicate keys, the last value will be used.
func (ml MapList[T]) ToMap() map[string]T {
	m := make(map[string]T, len(ml))
	for _, pair := range ml {
		m[pair.Name] = pair.Value
	}
	return m
}

// FromMap converts a map[string]T to a MapList[T].
// The output pairs are sorted by name to allow comparisons in tests.
func FromMap[T any](m map[string]T) MapList[T] {
	ml := make(MapList[T], 0, len(m))
	for name, value := range m {
		ml = append(ml, Pair[T]{
			Name:  name,
			Value: value,
		})
	}
	slices.SortFunc(ml, func(p1, p2 Pair[T]) int {
		return cmp.Compare(p1.Name, p2.Name)
	})
	return ml
}
