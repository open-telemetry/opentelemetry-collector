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

// MarshalProfiles to the OTLP/JSON format.
func (*JSONMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	pd.marshalJSONStream(dest)
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
	td := NewProfiles()
	td.unmarshalJSONIter(iter)
	if iter.Error() != nil {
		return Profiles{}, iter.Error()
	}
	otlp.MigrateProfiles(td.getOrig().ResourceProfiles)
	return td, nil
}
