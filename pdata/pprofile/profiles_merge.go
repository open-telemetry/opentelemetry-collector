// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// MergeTo merges the current Profiles into dest, updating the destination
// dictionary as needed and appending the resource profiles.
// The source Profiles is consumed and marked read-only after this operation.
func (ms Profiles) MergeTo(dest Profiles) error {
	ms.getState().AssertMutable()
	dest.getState().AssertMutable()
	if ms.getOrig() == dest.getOrig() {
		return nil
	}

	reserveDictionaryZeroValues(dest.Dictionary())

	if err := ms.switchDictionary(ms.Dictionary(), dest.Dictionary()); err != nil {
		return err
	}

	ms.ResourceProfiles().MoveAndAppendTo(dest.ResourceProfiles())
	ms.MarkReadOnly()

	return nil
}

// reserveDictionaryZeroValues seeds index 0 of each of dst's tables with that
// table's zero value, but only for tables that are still empty. Per the
// ProfilesDictionary contract, index 0 of every table MUST hold the zero
// value for that table's element type, since unset references (e.g. an
// unset Strindex) resolve to it. Tables that already hold entries are left
// untouched, since inserting at index 0 would invalidate every existing
// reference into them.
func reserveDictionaryZeroValues(dst ProfilesDictionary) {
	if dst.StringTable().Len() == 0 {
		dst.StringTable().Append("")
	}
	if dst.AttributeTable().Len() == 0 {
		dst.AttributeTable().AppendEmpty()
	}
	if dst.FunctionTable().Len() == 0 {
		dst.FunctionTable().AppendEmpty()
	}
	if dst.LinkTable().Len() == 0 {
		dst.LinkTable().AppendEmpty()
	}
	if dst.LocationTable().Len() == 0 {
		dst.LocationTable().AppendEmpty()
	}
	if dst.MappingTable().Len() == 0 {
		dst.MappingTable().AppendEmpty()
	}
	if dst.StackTable().Len() == 0 {
		dst.StackTable().AppendEmpty()
	}
}
