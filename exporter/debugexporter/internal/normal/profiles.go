// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/normal"

import (
	"bytes"
	"fmt"
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
	dic := pd.Dictionary()

	for i := 0; i < pd.ResourceProfiles().Len(); i++ {
		resourceProfiles := pd.ResourceProfiles().At(i)

		buffer.WriteString(fmt.Sprintf("ResourceProfiles #%d%s%s\n", i, writeResourceDetails(resourceProfiles.SchemaUrl()), writeAttributesString(resourceProfiles.Resource().Attributes())))

		for j := 0; j < resourceProfiles.ScopeProfiles().Len(); j++ {
			scopeProfiles := resourceProfiles.ScopeProfiles().At(j)

			buffer.WriteString(fmt.Sprintf("ScopeProfiles #%d%s%s\n", i, writeScopeDetails(scopeProfiles.Scope().Name(), scopeProfiles.Scope().Version(), scopeProfiles.SchemaUrl()), writeAttributesString(scopeProfiles.Scope().Attributes())))

			for k := 0; k < scopeProfiles.Profiles().Len(); k++ {
				profile := scopeProfiles.Profiles().At(k)

				buffer.WriteString(profile.ProfileID().String())

				buffer.WriteString(" samples=")
				buffer.WriteString(strconv.Itoa(profile.Samples().Len()))

				if profile.AttributeIndices().Len() > 0 {
					attrs := []string{}
					for _, i := range profile.AttributeIndices().AsRaw() {
						attrIdx := int(i)
						if attrIdx >= dic.AttributeTable().Len() {
							return nil, fmt.Errorf("invalid attribute index %d, attribute table size %d", attrIdx, dic.AttributeTable().Len())
						}
						a := dic.AttributeTable().At(attrIdx)
						keyIdx := int(a.KeyStrindex())
						if keyIdx >= dic.StringTable().Len() {
							return nil, fmt.Errorf("invalid string index %d, string table size %d", keyIdx, dic.StringTable().Len())
						}
						attrs = append(attrs, fmt.Sprintf("%s=%s", dic.StringTable().At(keyIdx), a.Value().AsString()))
					}

					buffer.WriteString(" ")
					buffer.WriteString(strings.Join(attrs, " "))
				}

				buffer.WriteString("\n")
			}
		}
	}
	return buffer.Bytes(), nil
}
