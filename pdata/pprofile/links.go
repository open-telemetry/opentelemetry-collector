// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"errors"
	"fmt"
	"math"
)

var errTooManyLinkTableEntries = errors.New("too many entries in LinkTable")

// SetLink updates a LinkTable and a Sample's LinkIndex to
// add or update a link.
func SetLink(table LinkSlice, record Sample, li Link) error {
	idx := int(record.LinkIndex())
	if idx > 0 {
		if idx >= table.Len() {
			return fmt.Errorf("index value %d out of range for LinkIndex", idx)
		}
		mapAt := table.At(idx)
		if mapAt.Equal(li) {
			// Link already exists, nothing to do.
			return nil
		}
	}

	for j, l := range table.All() {
		if l.Equal(li) {
			if j > math.MaxInt32 {
				return errTooManyLinkTableEntries
			}
			// Add the index of the existing link to the indices.
			record.SetLinkIndex(int32(j)) //nolint:gosec // G115 overflow checked
			return nil
		}
	}

	if table.Len() >= math.MaxInt32 {
		return errTooManyLinkTableEntries
	}

	li.CopyTo(table.AppendEmpty())
	record.SetLinkIndex(int32(table.Len() - 1)) //nolint:gosec // G115 overflow checked
	return nil
}
