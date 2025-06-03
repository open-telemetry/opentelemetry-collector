// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// MoveAndAppendTo moves all elements from all the dictionary's internal slices appends them to the dest.
// The current dictionary will be cleared.
func (pd ProfilesDictionary) MoveAndAppendTo(dest ProfilesDictionary) {
	dest.state.AssertMutable()
	pd.MappingTable().MoveAndAppendTo(dest.MappingTable())
	pd.LocationTable().MoveAndAppendTo(dest.LocationTable())
	pd.FunctionTable().MoveAndAppendTo(dest.FunctionTable())
	pd.LinkTable().MoveAndAppendTo(dest.LinkTable())
	pd.StringTable().MoveAndAppendTo(dest.StringTable())
	pd.AttributeTable().MoveAndAppendTo(dest.AttributeTable())
	pd.AttributeUnits().MoveAndAppendTo(dest.AttributeUnits())
}
