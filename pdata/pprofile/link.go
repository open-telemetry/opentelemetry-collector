// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Equal checks equality with another Link
func (l Link) Equal(val Link) bool {
	return l.TraceID() == val.TraceID() &&
		l.SpanID() == val.SpanID()
}
