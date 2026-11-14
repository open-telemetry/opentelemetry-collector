// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// MergeTo merges the current Profiles into dest, updating the destination
// dictionary as needed and appending the resource profiles.
// The source Profiles is consumed and marked read-only after this operation.
//
// MergeTo indexes the destination dictionary from scratch on every call. To
// merge many Profiles into the same destination (for example when assembling
// an exporter batch), use a Merger so the index is built only once.
func (ms Profiles) MergeTo(dest Profiles) error {
	ms.getState().AssertMutable()
	dest.getState().AssertMutable()
	return NewMerger(dest).Merge(ms)
}
