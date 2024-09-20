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
			in:   testdata.GenerateExtendedProfiles(2),
			out:  "two_profiles.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextProfilesMarshaler().MarshalProfiles(tt.in)
			assert.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "profiles", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}
