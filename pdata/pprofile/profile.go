// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "go.opentelemetry.io/collector/pdata/pcommon"

// Attributes returns the Attributes associated with this Profile.
//
// Deprecated: [v0.117.0] This field has been removed, and replaced with AttributeIndices
func (ms Profile) Attributes() pcommon.Map {
	return pcommon.NewMap()
}
