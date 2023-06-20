// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
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
			name: "profiles_with_one_record",
			in:   testdata.GenerateProfiles(1),
			out:  "one_record.out",
		},
		{
			name: "profiles_with_two_records",
			in:   testdata.GenerateProfiles(2),
			out:  "two_records.out",
		},
		{
			name: "profiles_with_embedded_maps",
			in: func() pprofile.Profiles {
				ls := pprofile.NewProfiles()
				l := ls.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
				l.SetStartTime(pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)))
				l.SetSeverityNumber(pprofile.SeverityNumberInfo)
				l.SetSeverityText("INFO")
				bm := l.Body().SetEmptyMap()
				bm.PutStr("key1", "val1")
				bmm := bm.PutEmptyMap("key2")
				bmm.PutStr("key21", "val21")
				bmm.PutStr("key22", "val22")
				am := l.Attributes().PutEmptyMap("key1")
				am.PutStr("key11", "val11")
				am.PutStr("key12", "val12")
				am.PutEmptyMap("key13").PutStr("key131", "val131")
				l.Attributes().PutStr("key2", "val2")
				return ls
			}(),
			out: "embedded_maps.out",
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
