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

	tmpDstDict := NewProfilesDictionary()
	dest.Dictionary().CopyTo(tmpDstDict)

	if err := tmp.switchDictionary(tmp.Dictionary(), tmpDstDict); err != nil {
		return err
	}

	tmpDstDict.CopyTo(dest.Dictionary())
	tmp.ResourceProfiles().MoveAndAppendTo(dest.ResourceProfiles())

	return nil
}
