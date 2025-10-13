// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"
)

var errTooManyStackTableEntries = errors.New("too many entries in StackTable")

// SetStack updates a StackTable and a Sample's StackIndex to
// set its stack.
func SetStack(table StackSlice, record Sample, st Stack) error {
	idx := int(record.StackIndex())
	if idx > 0 {
		if idx >= table.Len() {
			return fmt.Errorf("index value %d out of range for StackIndex", idx)
		}
		mapAt := table.At(idx)
		if mapAt.Equal(st) {
			// Stack already exists, nothing to do.
			return nil
		}
	}

	for j, l := range table.All() {
		if l.Equal(st) {
			if j > math.MaxInt32 {
				return errTooManyStackTableEntries
			}
			// Add the index of the existing stack to the indices.
			record.SetStackIndex(int32(j)) //nolint:gosec // G115 overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errTooManyStackTableEntries
	}

	st.CopyTo(table.AppendEmpty())
	record.SetStackIndex(int32(table.Len() - 1)) //nolint:gosec // G115 overflow checked
	return nil
}
