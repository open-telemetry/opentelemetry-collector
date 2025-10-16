// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another KeyValueAndUnit
// It assumes both structs refer to the same dictionary.
func (ms KeyValueAndUnit) Equal(val KeyValueAndUnit) bool {
	return ms.KeyStrindex() == val.KeyStrindex() &&
		ms.UnitStrindex() == val.UnitStrindex() &&
		ms.Value().Equal(val.Value())
}
