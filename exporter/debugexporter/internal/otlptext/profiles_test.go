// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfilesText(t *testing.T) {
	tests := []struct {
		name string
		in   pprofile.Profiles
		out  string
	}{
		{
			name: "empty_profiles",
			in:   pprofile.NewProfiles(),
			out:  "empty.out",
		},
		{
			name: "two_profiles",
			in:   extendProfiles(testdata.GenerateProfiles(2)),
			out:  "two_profiles.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextProfilesMarshaler().MarshalProfiles(tt.in)
			require.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "profiles", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

// GenerateExtendedProfiles generates dummy profiling data with extended values for tests
func extendProfiles(profiles pprofile.Profiles) pprofile.Profiles {
	sc := profiles.ResourceProfiles().At(0).ScopeProfiles().At(0)
	profilesCount := profiles.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().Len()
	for i := 0; i < profilesCount; i++ {
		switch i % 2 {
		case 0:
			profile := sc.Profiles().At(i)
			profile.Profile().LocationIndices().FromRaw([]int64{1})
			label := profile.Profile().Sample().At(0).Label().AppendEmpty()
			label.SetKey(1)
			label.SetStr(2)
			label.SetNum(3)
			label.SetNumUnit(4)

			location := profile.Profile().Location().AppendEmpty()
			location.SetID(2)
			location.SetMappingIndex(3)
			location.SetAddress(4)
			line := location.Line().AppendEmpty()
			line.SetFunctionIndex(1)
			line.SetLine(2)
			line.SetColumn(3)
			location.SetIsFolded(true)
			location.SetTypeIndex(5)
			location.Attributes().FromRaw([]uint64{6, 7})

			_ = profile.Profile().AttributeTable().FromRaw(map[string]any{
				"value": map[string]any{
					"intValue": "42",
				},
			})

			attributeUnits := profile.Profile().AttributeUnits().AppendEmpty()
			attributeUnits.SetAttributeKey(1)
			attributeUnits.SetUnit(5)

			profile.Profile().StringTable().Append("foobar")
		case 1:
			profile := sc.Profiles().At(i)
			profile.Profile().SetDropFrames(1)
			profile.Profile().SetKeepFrames(2)

			mapping := profile.Profile().Mapping().AppendEmpty()
			mapping.SetID(1)
			mapping.SetMemoryStart(2)
			mapping.SetMemoryLimit(3)
			mapping.SetFileOffset(4)
			mapping.SetFilename(5)
			mapping.SetBuildID(6)
			mapping.Attributes().FromRaw([]uint64{7, 8})
			mapping.SetHasFunctions(true)
			mapping.SetHasFilenames(true)
			mapping.SetHasLineNumbers(true)
			mapping.SetHasInlineFrames(true)

			function := profile.Profile().Function().AppendEmpty()
			function.SetID(1)
			function.SetName(2)
			function.SetSystemName(3)
			function.SetFilename(4)
			function.SetStartLine(5)

			linkTable := profile.Profile().LinkTable().AppendEmpty()
			linkTable.SetTraceID([16]byte{0x03, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
			linkTable.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})

			profile.Profile().Comment().FromRaw([]int64{1, 2})
		}
	}
	return profiles
}
