// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"strconv"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

// NewTextProfilesMarshaler returns a pprofile.Marshaler to encode to OTLP text bytes.
func NewTextProfilesMarshaler() pprofile.Marshaler {
	return textProfilesMarshaler{}
}

type textProfilesMarshaler struct{}

// MarshalProfiles pprofile.Profiles to OTLP text.
func (textProfilesMarshaler) MarshalProfiles(pd pprofile.Profiles) ([]byte, error) {
	buf := dataBuffer{}
	dic := pd.ProfilesDictionary()
	rps := pd.ResourceProfiles()

	buf.logProfileMappings(dic.MappingTable())
	buf.logProfileLocations(dic.LocationTable())
	buf.logProfileFunctions(dic.FunctionTable())
	buf.logAttributesWithIndentation(
		"Attribute units",
		attributeUnitsToMap(dic.AttributeUnits()),
		0)

	buf.logAttributesWithIndentation(
		"Link table",
		linkTableToMap(dic.LinkTable()),
		0)

	buf.logStringTable(dic.StringTable())

	for i := 0; i < rps.Len(); i++ {
		buf.logEntry("ResourceProfiles #%d", i)
		rp := rps.At(i)
		buf.logEntry("Resource SchemaURL: %s", rp.SchemaUrl())
		buf.logAttributes("Resource attributes", rp.Resource().Attributes())
		ilps := rp.ScopeProfiles()
		for j := 0; j < ilps.Len(); j++ {
			buf.logEntry("ScopeProfiles #%d", j)
			ilp := ilps.At(j)
			buf.logEntry("ScopeProfiles SchemaURL: %s", ilp.SchemaUrl())
			buf.logInstrumentationScope(ilp.Scope())
			profiles := ilp.Profiles()
			for k := 0; k < profiles.Len(); k++ {
				buf.logEntry("Profile #%d", k)
				profile := profiles.At(k)
				buf.logAttr("Profile ID", profile.ProfileID())
				buf.logAttr("Start time", profile.Time().String())
				buf.logAttr("Duration", profile.Duration().String())
				buf.logAttr("Dropped attributes count", strconv.FormatUint(uint64(profile.DroppedAttributesCount()), 10))
				buf.logEntry("    Location indices: %d", profile.LocationIndices().AsRaw())

				buf.logProfileSamples(profile.Sample(), dic.AttributeTable())
				buf.logComment(profile.CommentStrindices())
			}
		}
	}

	return buf.buf.Bytes(), nil
}
