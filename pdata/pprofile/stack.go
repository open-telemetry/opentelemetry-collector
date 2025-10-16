// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Stack
func (ms Stack) Equal(val Stack) bool {
	if ms.LocationIndices().Len() != val.LocationIndices().Len() {
		return false
	}

	for i := range ms.LocationIndices().Len() {
		if ms.LocationIndices().At(i) != val.LocationIndices().At(i) {
			return false
		}
	}

	return true
}
