// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"
)

var errTooManyMappingTableEntries = errors.New("too many entries in MappingTable")

// SetMapping updates a MappingTable and a Location's MappingIndex to
// add or update a mapping.
func SetMapping(table MappingSlice, record Location, ma Mapping) error {
	idx := int(record.MappingIndex())
	if idx > 0 {
		if idx >= table.Len() {
			return fmt.Errorf("index value %d out of range for MappingIndex", idx)
		}
		mapAt := table.At(idx)
		if mapAt.Equal(ma) {
			// Mapping already exists, nothing to do.
			return nil
		}
	}

	for j, m := range table.All() {
		if m.Equal(ma) {
			if j > math.MaxInt32 {
				return errTooManyMappingTableEntries
			}
			// Add the index of the existing mapping to the indices.
			record.SetMappingIndex(int32(j)) //nolint:gosec // G115 overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errTooManyMappingTableEntries
	}

	ma.CopyTo(table.AppendEmpty())
	record.SetMappingIndex(int32(table.Len() - 1)) //nolint:gosec // G115 overflow checked
	return nil
}
