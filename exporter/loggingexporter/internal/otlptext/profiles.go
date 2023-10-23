// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/loggingexporter/internal/otlptext"

import (
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// NewTextProfilesMarshaler returns a pprofile.Marshaler to encode to OTLP text bytes.
func NewTextProfilesMarshaler() pprofile.Marshaler {
	return textProfilesMarshaler{}
}

type textProfilesMarshaler struct{}

// MarshalProfiles pprofile.Profiles to OTLP text.
func (textProfilesMarshaler) MarshalProfiles(ld pprofile.Profiles) ([]byte, error) {
	buf := dataBuffer{}
	rls := ld.ResourceProfiles()
	for i := 0; i < rls.Len(); i++ {
		buf.logEntry("ResourceProfile #%d", i)
		rl := rls.At(i)
		buf.logEntry("Resource SchemaURL: %s", rl.SchemaUrl())
		buf.logAttributes("Resource attributes", rl.Resource().Attributes())
		ills := rl.ScopeProfiles()
		for j := 0; j < ills.Len(); j++ {
			buf.logEntry("ScopeProfiles #%d", j)
			ils := ills.At(j)
			buf.logEntry("ScopeProfiles SchemaURL: %s", ils.SchemaUrl())
			buf.logInstrumentationScope(ils.Scope())

			logs := ils.Profiles()
			for k := 0; k < logs.Len(); k++ {
				buf.logEntry("ProfileRecord #%d", k)
				lr := logs.At(k)
				buf.logEntry("Profile ID: %s", lr.ProfileID())
				buf.logEntry("StartTime: %s", lr.StartTime())
				buf.logEntry("EndTime: %s", lr.EndTime())
				buf.logAttributes("Attributes", lr.Attributes())
			}
		}
	}

	return buf.buf.Bytes(), nil
}
