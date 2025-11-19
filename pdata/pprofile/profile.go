// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import "go.opentelemetry.io/collector/pdata/pcommon"

// Duration returns the duration associated with this Profile.
//
// Deprecated: Use Profile.DurationNano instead.
func (ms Profile) Duration() pcommon.Timestamp {
	return pcommon.Timestamp(0)
}

// SetDuration replaces the duration associated with this Profile.
//
// Deprecated: Use Profile.SetDurationNano instead.
func (ms Profile) SetDuration(v pcommon.Timestamp) {
}
