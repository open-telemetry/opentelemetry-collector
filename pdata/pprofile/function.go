// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// Equal checks equality with another Function
func (fn Function) Equal(val Function) bool {
	return fn.NameStrindex() == val.NameStrindex() &&
		fn.SystemNameStrindex() == val.SystemNameStrindex() &&
		fn.FilenameStrindex() == val.FilenameStrindex() &&
		fn.StartLine() == val.StartLine()
}

// switchDictionary updates the Function, switching its indices from one
// dictionary to another.
func (ms Function) switchDictionary(src, dst ProfilesDictionary) error {
	if ms.NameStrindex() > 0 {
		if src.StringTable().Len() < int(ms.NameStrindex()) {
			return fmt.Errorf("invalid name index %d", ms.NameStrindex())
		}

		idx, err := SetString(dst.StringTable(), src.StringTable().At(int(ms.NameStrindex())))
		if err != nil {
			return fmt.Errorf("couldn't set name: %w", err)
		}
		ms.SetNameStrindex(idx)
	}

	if ms.SystemNameStrindex() > 0 {
		if src.StringTable().Len() < int(ms.SystemNameStrindex()) {
			return fmt.Errorf("invalid system name index %d", ms.SystemNameStrindex())
		}

		idx, err := SetString(dst.StringTable(), src.StringTable().At(int(ms.SystemNameStrindex())))
		if err != nil {
			return fmt.Errorf("couldn't set system name: %w", err)
		}
		ms.SetSystemNameStrindex(idx)
	}

	if ms.FilenameStrindex() > 0 {
		if src.StringTable().Len() < int(ms.FilenameStrindex()) {
			return fmt.Errorf("invalid filename index %d", ms.FilenameStrindex())
		}

		idx, err := SetString(dst.StringTable(), src.StringTable().At(int(ms.FilenameStrindex())))
		if err != nil {
			return fmt.Errorf("couldn't set filename: %w", err)
		}
		ms.SetFilenameStrindex(idx)
	}

	return nil
}
