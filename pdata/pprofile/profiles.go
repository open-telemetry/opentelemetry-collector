// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// MarkReadOnly marks the ResourceProfiles as shared so that no further modifications can be done on it.
func (ms Profiles) MarkReadOnly() {
	ms.getState().MarkReadOnly()
}

// IsReadOnly returns true if this ResourceProfiles instance is read-only.
func (ms Profiles) IsReadOnly() bool {
	return ms.getState().IsReadOnly()
}

// SampleCount calculates the total number of samples.
func (ms Profiles) SampleCount() int {
	sampleCount := 0
	rps := ms.ResourceProfiles()
	for i := 0; i < rps.Len(); i++ {
		rp := rps.At(i)
		sps := rp.ScopeProfiles()
		for j := 0; j < sps.Len(); j++ {
			pcs := sps.At(j).Profiles()
			for k := 0; k < pcs.Len(); k++ {
				sampleCount += pcs.At(k).Sample().Len()
			}
		}
	}
	return sampleCount
}
