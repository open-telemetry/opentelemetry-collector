// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Mapping
func (m Mapping) Equal(val Mapping) bool {
	return m.MemoryStart() == val.MemoryStart() &&
		m.MemoryLimit() == val.MemoryLimit() &&
		m.FileOffset() == val.FileOffset() &&
		m.FilenameStrindex() == val.FilenameStrindex() &&
		m.AttributeIndices().Equal(val.AttributeIndices()) &&
		m.HasFunctions() == val.HasFunctions() &&
		m.HasFilenames() == val.HasFilenames() &&
		m.HasLineNumbers() == val.HasLineNumbers() &&
		m.HasInlineFrames() == val.HasInlineFrames()
}
