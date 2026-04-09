// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pprofile.Profiles to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalProfiles marshals Profiles to OTLP/JSON format bytes.
// If the input data is read-only, it will be copied to a mutable
// instance before mutation.
func (*JSONMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	// Only copy if data is shared/read-only to avoid unnecessary allocation
	pdToUse := pd
	if pd.IsReadOnly() {
		pdCopy := NewProfiles()
		pd.CopyTo(pdCopy)
		pdToUse = pdCopy
	}

	// Convert strings to references for efficient transmission
	convertProfilesToReferences(pdToUse)

	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	pdToUse.getOrig().MarshalJSON(dest)
	if dest.Error() != nil {
		return nil, dest.Error()
	}
	return slices.Clone(dest.Buffer()), nil
}

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pprofile.Profiles.
type JSONUnmarshaler struct{}

// UnmarshalProfiles from OTLP/JSON format into pprofile.Profiles.
func (*JSONUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	iter := json.BorrowIterator(buf)
	defer json.ReturnIterator(iter)
	pd := NewProfiles()
	pd.getOrig().UnmarshalJSON(iter)
	if iter.Error() != nil {
		return Profiles{}, iter.Error()
	}
	otlp.MigrateProfiles(pd.getOrig().ResourceProfiles)

	// Resolve all string_value_ref and key_ref to their actual strings
	// so the pdata API works transparently
	resolveProfilesReferences(pd)

	return pd, nil
}
