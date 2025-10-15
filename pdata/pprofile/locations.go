// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"
)

// FromLocationIndices builds a slice containing all the locations of a Stack.
// Updates made to the returned map will not be applied back to the Stack.
func FromLocationIndices(table LocationSlice, record Stack) LocationSlice {
	m := NewLocationSlice()
	m.EnsureCapacity(record.LocationIndices().Len())

	for _, idx := range record.LocationIndices().All() {
		l := table.At(int(idx))
		l.CopyTo(m.AppendEmpty())
	}

	return m
}

var (
	errTooManyLocationTableEntries   = errors.New("too many entries in LocationTable")
	errTooManyLocationIndicesEntries = errors.New("too many entries in LocationIndices")
)

// PutLocation updates a LocationTable and a Stack's LocationIndices to
// add or update a location.
//
// Deprecated: [v0.138.0] use SetLocation instead.
func PutLocation(table LocationSlice, record Stack, loc Location) error {
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
		return errTooManyLocationIndicesEntries
	}

	id, err := SetLocation(table, loc)
	if err != nil {
		return err
	}
	record.LocationIndices().Append(id)
	return nil
}

// SetLocation updates a LocationTable, adding or providing a value and returns
// its index.
func SetLocation(table LocationSlice, loc Location) (int32, error) {
	for j, a := range table.All() {
		if a.Equal(loc) {
			if j > math.MaxInt32 {
				return 0, errTooManyLocationTableEntries
			}
			return int32(j), nil //nolint:gosec // G115 overflow checked
		}
	}

	if table.Len() >= math.MaxInt32 {
		return 0, errTooManyLocationTableEntries
	}

	loc.CopyTo(table.AppendEmpty())
	return int32(table.Len() - 1), nil //nolint:gosec // G115 overflow checked
}
