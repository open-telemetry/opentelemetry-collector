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
			profile.LocationIndices().FromRaw([]int32{1})

			location := profile.LocationTable().AppendEmpty()
			location.SetMappingIndex(3)
			location.SetAddress(4)
			line := location.Line().AppendEmpty()
			line.SetFunctionIndex(1)
			line.SetLine(2)
			line.SetColumn(3)
			location.SetIsFolded(true)
			location.AttributeIndices().FromRaw([]int32{6, 7})

			at := profile.AttributeTable()
			a := at.AppendEmpty()
			a.SetKey("intValue")
			a.Value().SetInt(42)

			attributeUnits := profile.AttributeUnits().AppendEmpty()
			attributeUnits.SetAttributeKeyStrindex(1)
			attributeUnits.SetUnitStrindex(5)

			profile.StringTable().Append("foobar")
		case 1:
			profile := sc.Profiles().At(i)

			mapping := profile.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(2)
			mapping.SetMemoryLimit(3)
			mapping.SetFileOffset(4)
			mapping.SetFilenameStrindex(5)
			mapping.AttributeIndices().FromRaw([]int32{7, 8})
			mapping.SetHasFunctions(true)
			mapping.SetHasFilenames(true)
			mapping.SetHasLineNumbers(true)
			mapping.SetHasInlineFrames(true)

			function := profile.FunctionTable().AppendEmpty()
			function.SetNameStrindex(2)
			function.SetSystemNameStrindex(3)
			function.SetFilenameStrindex(4)
			function.SetStartLine(5)

			linkTable := profile.LinkTable().AppendEmpty()
			linkTable.SetTraceID([16]byte{0x03, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
			linkTable.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})

			profile.CommentStrindices().FromRaw([]int32{1, 2})
		}
	}
	return profiles
}
