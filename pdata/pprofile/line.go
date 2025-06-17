// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another LineSlice
func (l LineSlice) Equal(val LineSlice) bool {
	if l.Len() != val.Len() {
		return false
	}

	for i := range l.Len() {
		if !l.At(i).Equal(val.At(i)) {
			return false
		}
	}

	return true
}

// Equal checks equality with another Line
func (l Line) Equal(val Line) bool {
	return l.Column() == val.Column() &&
		l.FunctionIndex() == val.FunctionIndex() &&
		l.Line() == val.Line()
}
