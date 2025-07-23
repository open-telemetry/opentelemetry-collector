// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Function
func (fn Function) Equal(val Function) bool {
	return fn.NameStrindex() == val.NameStrindex() &&
		fn.SystemNameStrindex() == val.SystemNameStrindex() &&
		fn.FilenameStrindex() == val.FilenameStrindex() &&
		fn.StartLine() == val.StartLine()
}
