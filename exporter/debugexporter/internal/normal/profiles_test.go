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
				profile := profiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				profile.SetProfileID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
				profile.Sample().AppendEmpty()
				profile.Sample().AppendEmpty()
				profile.AttributeIndices().Append(0)
				a := profile.AttributeTable().AppendEmpty()
				a.SetKey("key1")
				a.Value().SetStr("value1")
				return profiles
			}(),
			expected: `0102030405060708090a0b0c0d0e0f10 samples=2 key1=value1
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
