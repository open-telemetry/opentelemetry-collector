// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Mapping
func (ms Mapping) Equal(val Mapping) bool {
	return ms.MemoryStart() == val.MemoryStart() &&
		ms.MemoryLimit() == val.MemoryLimit() &&
		ms.FileOffset() == val.FileOffset() &&
		ms.FilenameStrindex() == val.FilenameStrindex() &&
		ms.AttributeIndices().Equal(val.AttributeIndices())
}
