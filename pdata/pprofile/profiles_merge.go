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

// reserveDictionaryZeroValues ensures that index 0 of every table in the destination
// dictionary is reserved for that table's zero value, seeding only tables that are still empty.
func reserveDictionaryZeroValues(dict ProfilesDictionary) {
	if dict.StringTable().Len() == 0 {
		dict.StringTable().Append("")
	}
	if dict.MappingTable().Len() == 0 {
		dict.MappingTable().AppendEmpty()
	}
	if dict.LocationTable().Len() == 0 {
		dict.LocationTable().AppendEmpty()
	}
	if dict.FunctionTable().Len() == 0 {
		dict.FunctionTable().AppendEmpty()
	}
	if dict.LinkTable().Len() == 0 {
		dict.LinkTable().AppendEmpty()
	}
	if dict.AttributeTable().Len() == 0 {
		dict.AttributeTable().AppendEmpty()
	}
	if dict.StackTable().Len() == 0 {
		dict.StackTable().AppendEmpty()
	}
}
