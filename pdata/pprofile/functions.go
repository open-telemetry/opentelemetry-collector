// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"
)

var errTooManyFunctionTableEntries = errors.New("too many entries in FunctionTable")

// SetFunction updates a FunctionTable and a Line's FunctionIndex to
// add or update a function.
func SetFunction(table FunctionSlice, record Line, fn Function) error {
	idx := int(record.FunctionIndex())
	if idx > 0 {
		if idx >= table.Len() {
			return fmt.Errorf("index value %d out of range for FunctionIndex", idx)
		}
		mapAt := table.At(idx)
		if mapAt.Equal(fn) {
			// Function already exists, nothing to do.
			return nil
		}
	}

	for j, m := range table.All() {
		if m.Equal(fn) {
			if j > math.MaxInt32 {
				return errTooManyFunctionTableEntries
			}
			// Add the index of the existing function to the indices.
			record.SetFunctionIndex(int32(j)) //nolint:gosec // G115 overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errTooManyFunctionTableEntries
	}

	fn.CopyTo(table.AppendEmpty())
	record.SetFunctionIndex(int32(table.Len() - 1)) //nolint:gosec // G115 overflow checked
	return nil
}
