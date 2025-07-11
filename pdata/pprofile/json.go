// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pprofile.Profiles to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalProfiles to the OTLP/JSON format.
func (*JSONMarshaler) MarshalProfiles(td Profiles) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.ProfilesToProto(internal.Profiles(td))
	err := json.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

var _ Unmarshaler = (*JSONUnmarshaler)(nil)

type JSONUnmarshaler struct{}

func (d *JSONUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	td := otlpprofile.ProfilesData{}
	if err := json.Unmarshal(bytes.NewReader(buf), &td); err != nil {
		return Profiles{}, err
	}

	otlp.MigrateProfiles(td.ResourceProfiles)
	return Profiles(internal.ProfilesFromProto(td)), nil
}
