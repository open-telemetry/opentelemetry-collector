// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestMarshalProfiles(t *testing.T) {
	tests := []struct {
		name     string
		input    pprofile.Profiles
		expected string
	}{
		{
			name:     "empty profile",
			input:    pprofile.NewProfiles(),
			expected: "",
		},
		{
			name: "one profile",
			input: func() pprofile.Profiles {
				profiles := pprofile.NewProfiles()
				dic := profiles.Dictionary()
				dic.StringTable().Append("")
				dic.StringTable().Append("key1")

				a := dic.AttributeTable().AppendEmpty()
				a.SetKeyStrindex(1)
				a.Value().SetStr("value1")

				profile := profiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				profiles.ResourceProfiles().At(0).SetSchemaUrl("https://example.com/resource")
				profiles.ResourceProfiles().At(0).Resource().Attributes().PutStr("resourceKey", "resourceValue")
				profiles.ResourceProfiles().At(0).ScopeProfiles().At(0).SetSchemaUrl("https://example.com/scope")
				profiles.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().SetName("scope-name")
				profiles.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().SetVersion("1.2.3")
				profiles.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().Attributes().PutStr("scopeKey", "scopeValue")
				profile.SetProfileID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
				profile.Samples().AppendEmpty()
				profile.Samples().AppendEmpty()
				profile.AttributeIndices().Append(0)
				return profiles
			}(),
			expected: `ResourceProfiles #0 [https://example.com/resource] resourceKey=resourceValue
ScopeProfiles #0 scope-name@1.2.3 [https://example.com/scope] scopeKey=scopeValue
0102030405060708090a0b0c0d0e0f10 samples=2 key1=value1
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := NewNormalProfilesMarshaler().MarshalProfiles(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(output))
		})
	}
}
