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

// reserveDictionaryZeroValues keeps index 0 of every dictionary table free for
// the zero value of that table, which ProfilesDictionary in profiles.proto
// requires so that an unset index resolves to nothing rather than to real data.
//
// Only empty tables get an entry. A table that already holds entries keeps
// them where they are: appending would not put the zero value at index 0, and
// inserting at the front would shift every index already pointing into it.
func reserveDictionaryZeroValues(dic ProfilesDictionary) {
	if dic.StringTable().Len() == 0 {
		dic.StringTable().Append("")
	}
	if dic.AttributeTable().Len() == 0 {
		dic.AttributeTable().AppendEmpty()
	}
	if dic.StackTable().Len() == 0 {
		dic.StackTable().AppendEmpty()
	}
	if dic.LocationTable().Len() == 0 {
		dic.LocationTable().AppendEmpty()
	}
	if dic.FunctionTable().Len() == 0 {
		dic.FunctionTable().AppendEmpty()
	}
	if dic.MappingTable().Len() == 0 {
		dic.MappingTable().AppendEmpty()
	}
	if dic.LinkTable().Len() == 0 {
		dic.LinkTable().AppendEmpty()
	}
}
