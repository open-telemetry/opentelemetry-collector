// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Location
func (ms Location) Equal(val Location) bool {
	return ms.MappingIndex() == val.MappingIndex() &&
		ms.Address() == val.Address() &&
		ms.AttributeIndices().Equal(val.AttributeIndices()) &&
		ms.Line().Equal(val.Line())
}
