// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// MergeTo merges the current Profiles into dest, updating the destination
// dictionary as needed and appending the resource profiles.
func (ms Profiles) MergeTo(dest Profiles) error {
	dest.getState().AssertMutable()
	if ms.getOrig() == dest.getOrig() {
		return nil
	}

	tmp := NewProfiles()
	ms.CopyTo(tmp)

	if err := tmp.switchDictionary(tmp.Dictionary(), dest.Dictionary()); err != nil {
		return err
	}

	tmp.ResourceProfiles().MoveAndAppendTo(dest.ResourceProfiles())

	return nil
}
