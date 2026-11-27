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

	addMissingZeroEntries(dest.Dictionary())
	if err := ms.switchDictionary(ms.Dictionary(), dest.Dictionary()); err != nil {
		return err
	}

	ms.ResourceProfiles().MoveAndAppendTo(dest.ResourceProfiles())
	ms.MarkReadOnly()

	return nil
}

// addMissingZeroEntries appends the zero value entry to every empty table of
// d. Index 0 means unset, so without it the first entry merged into an empty
// dictionary, such as one from NewProfiles, would land on index 0 and every
// reference to it would resolve as unset.
func addMissingZeroEntries(d ProfilesDictionary) {
	if d.StringTable().Len() == 0 {
		d.StringTable().Append("")
	}
	if d.AttributeTable().Len() == 0 {
		d.AttributeTable().AppendEmpty()
	}
	if d.StackTable().Len() == 0 {
		d.StackTable().AppendEmpty()
	}
	if d.LocationTable().Len() == 0 {
		d.LocationTable().AppendEmpty()
	}
	if d.FunctionTable().Len() == 0 {
		d.FunctionTable().AppendEmpty()
	}
	if d.MappingTable().Len() == 0 {
		d.MappingTable().AppendEmpty()
	}
	if d.LinkTable().Len() == 0 {
		d.LinkTable().AppendEmpty()
	}
}
