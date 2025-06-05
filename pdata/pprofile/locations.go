// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

type locatable interface {
	LocationIndices() pcommon.Int32Slice
}

// FromLocationIndices builds a slice containing all the locations of a record.
// The record can be any struct that implements a `LocationIndices` method.
// Updates made to the returned map will not be applied back to the record.
func FromLocationIndices(table LocationSlice, record locatable) LocationSlice {
	m := NewLocationSlice()
	m.EnsureCapacity(record.LocationIndices().Len())

	for _, idx := range record.LocationIndices().All() {
		l := table.At(int(idx))
		l.CopyTo(m.AppendEmpty())
	}

	return m
}

var errTooManyLocationTableEntries = errors.New("too many entries in LocationTable")

// PutLocation updates a LocationTable and a record's LocationIndices to
// add or update a location.
// The record can be any struct that implements a `LocationIndices` method.
func PutLocation(table LocationSlice, record locatable, loc Location) error {
	for i, locIdx := range record.LocationIndices().All() {
		idx := int(locIdx)
		if idx < 0 || idx >= table.Len() {
			return fmt.Errorf("index value %d out of range in LocationIndices[%d]", idx, i)
		}
		locAt := table.At(idx)
		if locAt.Equal(loc) {
			// Location already exists, nothing to do.
			return nil
		}
	}

	if record.LocationIndices().Len() >= math.MaxInt32 {
		return errors.New("too many entries in LocationIndices")
	}

	for j, a := range table.All() {
		if a.Equal(loc) {
			if j > math.MaxInt32 {
				return errTooManyLocationTableEntries
			}
			// Add the index of the existing location to the indices.
			record.LocationIndices().Append(int32(j)) //nolint:gosec // overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errTooManyLocationTableEntries
	}

	loc.CopyTo(table.AppendEmpty())
	record.LocationIndices().Append(int32(table.Len() - 1)) //nolint:gosec // overflow checked
	return nil
}
