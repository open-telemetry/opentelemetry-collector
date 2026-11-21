// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// Equal checks equality with another Location
func (ms Location) Equal(val Location) bool {
	return ms.MappingIndex() == val.MappingIndex() &&
		ms.Address() == val.Address() &&
		ms.AttributeIndices().Equal(val.AttributeIndices()) &&
		ms.Lines().Equal(val.Lines())
}

// switchDictionary updates the Location, switching its indices from one
// dictionary to another.
func (ms Location) switchDictionary(src, dst ProfilesDictionary, mi *mergeIndex) error {
	if ms.MappingIndex() > 0 {
		if src.MappingTable().Len() <= int(ms.MappingIndex()) {
			return fmt.Errorf("invalid mapping index %d", ms.MappingIndex())
		}

		idx, err := mi.mapping(src, dst, ms.MappingIndex())
		if err != nil {
			return fmt.Errorf("couldn't set mapping: %w", err)
		}
		ms.SetMappingIndex(idx)
	}

	for i, v := range ms.AttributeIndices().All() {
		if v < 0 || src.AttributeTable().Len() <= int(v) {
			return fmt.Errorf("invalid attribute index %d", v)
		}

		idx, err := mi.attribute(src, dst, v)
		if err != nil {
			return fmt.Errorf("couldn't set attribute %d: %w", i, err)
		}
		ms.AttributeIndices().SetAt(i, idx)
	}

	for i, v := range ms.Lines().All() {
		if err := v.switchDictionary(src, dst, mi); err != nil {
			return fmt.Errorf("couldn't switch dictionary for line %d: %w", i, err)
		}
	}

	return nil
}
