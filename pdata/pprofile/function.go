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
func (fn Function) switchDictionary(src, dst ProfilesDictionary) error {
	if src.StringTable().Len() <= int(fn.NameStrindex()) {
		return fmt.Errorf("invalid name index %d", fn.NameStrindex())
	}

	nameIdx, err := SetString(dst.StringTable(), src.StringTable().At(int(fn.NameStrindex())))
	if err != nil {
		return fmt.Errorf("couldn't set name: %w", err)
	}
	fn.SetNameStrindex(nameIdx)

	if src.StringTable().Len() <= int(fn.SystemNameStrindex()) {
		return fmt.Errorf("invalid system name index %d", fn.SystemNameStrindex())
	}

	sysIdx, err := SetString(dst.StringTable(), src.StringTable().At(int(fn.SystemNameStrindex())))
	if err != nil {
		return fmt.Errorf("couldn't set system name: %w", err)
	}
	fn.SetSystemNameStrindex(sysIdx)

	if src.StringTable().Len() <= int(fn.FilenameStrindex()) {
		return fmt.Errorf("invalid filename index %d", fn.FilenameStrindex())
	}

	fileIdx, err := SetString(dst.StringTable(), src.StringTable().At(int(fn.FilenameStrindex())))
	if err != nil {
		return fmt.Errorf("couldn't set filename: %w", err)
	}
	fn.SetFilenameStrindex(fileIdx)

	return nil
}
