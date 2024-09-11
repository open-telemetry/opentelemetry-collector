// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/internal/data"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestReadOnlyProfilesInvalidUsage(t *testing.T) {
	profiles := NewProfiles()
	assert.False(t, profiles.IsReadOnly())
	res := profiles.ResourceProfiles().AppendEmpty().Resource()
	res.Attributes().PutStr("k1", "v1")
	profiles.MarkReadOnly()
	assert.True(t, profiles.IsReadOnly())
	assert.Panics(t, func() { res.Attributes().PutStr("k2", "v2") })
}

func BenchmarkProfilesUsage(b *testing.B) {
	profiles := NewProfiles()
	fillTestResourceProfilesSlice(profiles.ResourceProfiles())
	ts := pcommon.NewTimestampFromTime(time.Now())
	testValProfileID := ProfileID(data.ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
	testSecondValProfileID := ProfileID(data.ProfileID([16]byte{2, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))

	b.ReportAllocs()
	b.ResetTimer()

	for bb := 0; bb < b.N; bb++ {
		for i := 0; i < profiles.ResourceProfiles().Len(); i++ {
			rs := profiles.ResourceProfiles().At(i)
			res := rs.Resource()
			res.Attributes().PutStr("foo", "bar")
			v, ok := res.Attributes().Get("foo")
			assert.True(b, ok)
			assert.Equal(b, "bar", v.Str())
			v.SetStr("new-bar")
			assert.Equal(b, "new-bar", v.Str())
			res.Attributes().Remove("foo")
			for j := 0; j < rs.ScopeProfiles().Len(); j++ {
				iss := rs.ScopeProfiles().At(j)
				iss.Scope().SetName("new_test_name")
				assert.Equal(b, "new_test_name", iss.Scope().Name())
				for k := 0; k < iss.Profiles().Len(); k++ {
					s := iss.Profiles().At(k)
					s.SetProfileID(testValProfileID)
					assert.Equal(b, testValProfileID, s.ProfileID())
					s.SetStartTime(ts)
					assert.Equal(b, ts, s.StartTime())
					s.SetEndTime(ts)
					assert.Equal(b, ts, s.EndTime())
				}
				s := iss.Profiles().AppendEmpty()
				s.SetProfileID(testSecondValProfileID)
				s.SetStartTime(ts)
				s.SetEndTime(ts)
				s.Attributes().PutStr("foo1", "bar1")
				s.Attributes().PutStr("foo2", "bar2")
				iss.Profiles().RemoveIf(func(lr ProfileContainer) bool {
					return lr.ProfileID() == testSecondValProfileID
				})
			}
		}
	}
}
