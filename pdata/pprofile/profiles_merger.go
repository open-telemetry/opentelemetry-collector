// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

// Merger merges Profiles into a single destination Profiles while keeping the
// dictionary dedup index alive across merges. Profiles.MergeTo rebuilds that
// index over the whole destination dictionary on every call, so assembling a
// batch out of many small requests is quadratic in the batch size; a Merger
// builds the index once and each Merge then costs proportionally to the
// source only.
//
// The Merger owns the destination for its lifetime: mutating dest or its
// dictionary outside Merge leaves the index stale and can append duplicate
// dictionary entries. If the destination is changed by any other path, drop
// the Merger and build a new one.
//
// Experimental: this API is at an early stage of development and may change
// without backward compatibility notice.
type Merger struct {
	dest Profiles
	mi   *mergeIndex
}

// NewMerger returns a Merger that merges Profiles into dest, indexing dest's
// current dictionary up front.
func NewMerger(dest Profiles) *Merger {
	dest.getState().AssertMutable()
	return &Merger{
		dest: dest,
		mi:   newMergeIndex(dest.Dictionary()),
	}
}

// Merge merges src into the destination, updating the destination dictionary
// as needed and appending the resource profiles. The source is consumed and
// marked read-only after this operation, mirroring Profiles.MergeTo.
func (m *Merger) Merge(src Profiles) error {
	src.getState().AssertMutable()
	m.dest.getState().AssertMutable()
	if src.getOrig() == m.dest.getOrig() {
		return nil
	}

	if err := src.switchDictionaryWithIndex(src.Dictionary(), m.dest.Dictionary(), m.mi); err != nil {
		return err
	}

	src.ResourceProfiles().MoveAndAppendTo(m.dest.ResourceProfiles())
	src.MarkReadOnly()

	return nil
}
