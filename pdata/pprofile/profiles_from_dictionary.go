// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// NewProfilesFromDictionary moves rps, whose indices refer to dict, into a new
// Profiles with its own dictionary. The new dictionary only holds the entries
// rps references, plus the empty entry at index 0 of every table, so it is
// suited to splitting Profiles into parts that each carry only what they use.
//
// dict is not modified. If an error is returned, rps may have been partially
// translated and must not be used.
func NewProfilesFromDictionary(dict ProfilesDictionary, rps ResourceProfilesSlice) (Profiles, error) {
	pd := NewProfiles()
	dst := pd.Dictionary()
	dst.StringTable().Append("")
	dst.AttributeTable().AppendEmpty()
	dst.StackTable().AppendEmpty()
	dst.LocationTable().AppendEmpty()
	dst.FunctionTable().AppendEmpty()
	dst.MappingTable().AppendEmpty()
	dst.LinkTable().AppendEmpty()

	mi := newSplitIndex(dst)
	for i, rp := range rps.All() {
		if err := rp.switchDictionary(dict, dst, mi); err != nil {
			return Profiles{}, fmt.Errorf("error switching dictionary for resource profile %d: %w", i, err)
		}
	}
	rps.MoveAndAppendTo(pd.ResourceProfiles())
	return pd, nil
}
