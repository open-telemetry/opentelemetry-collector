// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Location
func (l Location) Equal(val Location) bool {
	return l.MappingIndex() == val.MappingIndex() &&
		l.Address() == val.Address() &&
		l.AttributeIndices().Equal(val.AttributeIndices()) &&
		l.IsFolded() == val.IsFolded() &&
		l.Line().Equal(val.Line())
}
