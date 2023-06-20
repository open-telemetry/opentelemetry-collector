// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestSplitProfiles_noop(t *testing.T) {
	td := testdata.GenerateProfiles(20)
	splitSize := 40
	split := splitProfiles(splitSize, td)
	assert.Equal(t, td, split)

	i := 0
	td.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().RemoveIf(func(_ pprofile.ProfileRecord) bool {
		i++
		return i > 5
	})
	assert.EqualValues(t, td, split)
}

func TestSplitProfiles(t *testing.T) {
	ld := testdata.GenerateProfiles(20)
	profiles := ld.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(0, i))
	}
	cp := pprofile.NewProfiles()
	cpProfiles := cp.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()
	cpProfiles.EnsureCapacity(5)
	ld.ResourceProfiles().At(0).Resource().CopyTo(
		cp.ResourceProfiles().At(0).Resource())
	ld.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().CopyTo(
		cp.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope())
	profiles.At(0).CopyTo(cpProfiles.AppendEmpty())
	profiles.At(1).CopyTo(cpProfiles.AppendEmpty())
	profiles.At(2).CopyTo(cpProfiles.AppendEmpty())
	profiles.At(3).CopyTo(cpProfiles.AppendEmpty())
	profiles.At(4).CopyTo(cpProfiles.AppendEmpty())

	splitSize := 5
	split := splitProfiles(splitSize, ld)
	assert.Equal(t, splitSize, split.ProfileRecordCount())
	assert.Equal(t, cp, split)
	assert.Equal(t, 15, ld.ProfileRecordCount())
	assert.Equal(t, "test-profile-int-0-0", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-4", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(4).SeverityText())

	split = splitProfiles(splitSize, ld)
	assert.Equal(t, 10, ld.ProfileRecordCount())
	assert.Equal(t, "test-profile-int-0-5", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-9", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(4).SeverityText())

	split = splitProfiles(splitSize, ld)
	assert.Equal(t, 5, ld.ProfileRecordCount())
	assert.Equal(t, "test-profile-int-0-10", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-14", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(4).SeverityText())

	split = splitProfiles(splitSize, ld)
	assert.Equal(t, 5, ld.ProfileRecordCount())
	assert.Equal(t, "test-profile-int-0-15", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-19", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(4).SeverityText())
}

func TestSplitProfilesMultipleResourceProfiles(t *testing.T) {
	td := testdata.GenerateProfiles(20)
	profiles := td.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(0, i))
	}
	// add second index to resource profiles
	testdata.GenerateProfiles(20).
		ResourceProfiles().At(0).CopyTo(td.ResourceProfiles().AppendEmpty())
	profiles = td.ResourceProfiles().At(1).ScopeProfiles().At(0).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(1, i))
	}

	splitSize := 5
	split := splitProfiles(splitSize, td)
	assert.Equal(t, splitSize, split.ProfileRecordCount())
	assert.Equal(t, 35, td.ProfileRecordCount())
	assert.Equal(t, "test-profile-int-0-0", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-4", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(4).SeverityText())
}

func TestSplitProfilesMultipleResourceProfiles_split_size_greater_than_profile_size(t *testing.T) {
	td := testdata.GenerateProfiles(20)
	profiles := td.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(0, i))
	}
	// add second index to resource profiles
	testdata.GenerateProfiles(20).
		ResourceProfiles().At(0).CopyTo(td.ResourceProfiles().AppendEmpty())
	profiles = td.ResourceProfiles().At(1).ScopeProfiles().At(0).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(1, i))
	}

	splitSize := 25
	split := splitProfiles(splitSize, td)
	assert.Equal(t, splitSize, split.ProfileRecordCount())
	assert.Equal(t, 40-splitSize, td.ProfileRecordCount())
	assert.Equal(t, 1, td.ResourceProfiles().Len())
	assert.Equal(t, "test-profile-int-0-0", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-19", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(19).SeverityText())
	assert.Equal(t, "test-profile-int-1-0", split.ResourceProfiles().At(1).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-1-4", split.ResourceProfiles().At(1).ScopeProfiles().At(0).Profiles().At(4).SeverityText())
}

func TestSplitProfilesMultipleILL(t *testing.T) {
	td := testdata.GenerateProfiles(20)
	profiles := td.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(0, i))
	}
	// add second index to ILL
	td.ResourceProfiles().At(0).ScopeProfiles().At(0).
		CopyTo(td.ResourceProfiles().At(0).ScopeProfiles().AppendEmpty())
	profiles = td.ResourceProfiles().At(0).ScopeProfiles().At(1).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(1, i))
	}

	// add third index to ILL
	td.ResourceProfiles().At(0).ScopeProfiles().At(0).
		CopyTo(td.ResourceProfiles().At(0).ScopeProfiles().AppendEmpty())
	profiles = td.ResourceProfiles().At(0).ScopeProfiles().At(2).Profiles()
	for i := 0; i < profiles.Len(); i++ {
		profiles.At(i).SetSeverityText(getTestProfileSeverityText(2, i))
	}

	splitSize := 40
	split := splitProfiles(splitSize, td)
	assert.Equal(t, splitSize, split.ProfileRecordCount())
	assert.Equal(t, 20, td.ProfileRecordCount())
	assert.Equal(t, "test-profile-int-0-0", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).SeverityText())
	assert.Equal(t, "test-profile-int-0-4", split.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(4).SeverityText())
}
