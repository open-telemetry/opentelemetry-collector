// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/normal"

import (
	"bytes"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

type normalProfilesMarshaler struct{}

// Ensure normalProfilesMarshaller implements interface pprofile.Marshaler
var _ pprofile.Marshaler = normalProfilesMarshaler{}

// NewNormalProfilesMarshaler returns a pprofile.Marshaler for normal verbosity. It writes one line of text per log record
func NewNormalProfilesMarshaler() pprofile.Marshaler {
	return normalProfilesMarshaler{}
}

func (normalProfilesMarshaler) MarshalProfiles(pd pprofile.Profiles) ([]byte, error) {
	var buffer bytes.Buffer
	for i := 0; i < pd.ResourceProfiles().Len(); i++ {
		resourceProfiles := pd.ResourceProfiles().At(i)
		for j := 0; j < resourceProfiles.ScopeProfiles().Len(); j++ {
			scopeProfiles := resourceProfiles.ScopeProfiles().At(j)
			for k := 0; k < scopeProfiles.Profiles().Len(); k++ {
				profile := scopeProfiles.Profiles().At(k)

				buffer.WriteString(profile.ProfileID().String())

				buffer.WriteString(" samples=")
				buffer.WriteString(strconv.Itoa(profile.Sample().Len()))

				if profile.Attributes().Len() > 0 {
					profileAttributes := writeAttributes(profile.Attributes())
					buffer.WriteString(" ")
					buffer.WriteString(strings.Join(profileAttributes, " "))
				}

				buffer.WriteString("\n")
			}
		}
	}
	return buffer.Bytes(), nil
}
