// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "go.opentelemetry.io/collector/pdata/pcommon"

// EndTime returns the end time associated with this Profile.
//
// Deprecated: [v0.115.0] This field has been removed, and replaced with Duration
func (ms Profile) EndTime() pcommon.Timestamp {
	return pcommon.Timestamp(0)
}

// SetEndTime replaces the end time associated with this Profile.
//
// Deprecated: [v0.115.0] This field has been removed, and replaced with Duration
func (ms Profile) SetEndTime(pcommon.Timestamp) {
}
