// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	otlpcollectorprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/profiles/v1"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
)

type Profiles struct {
	orig *otlpcollectorprofile.ExportProfilesServiceRequest
}

func GetOrigProfiles(ms Profiles) *otlpcollectorprofile.ExportProfilesServiceRequest {
	return ms.orig
}

func NewProfiles(orig *otlpcollectorprofile.ExportProfilesServiceRequest) Profiles {
	return Profiles{orig: orig}
}

// ProfilesToProto internal helper to convert Profiles to protobuf representation.
func ProfilesToProto(l Profiles) otlpprofiles.ProfilesData {
	return otlpprofiles.ProfilesData{
		ResourceProfiles: l.orig.ResourceProfiles,
	}
}

// ProfilesFromProto internal helper to convert protobuf representation to Profiles.
func ProfilesFromProto(orig otlpprofiles.ProfilesData) Profiles {
	return Profiles{orig: &otlpcollectorprofile.ExportProfilesServiceRequest{
		ResourceProfiles: orig.ResourceProfiles,
	}}
}
