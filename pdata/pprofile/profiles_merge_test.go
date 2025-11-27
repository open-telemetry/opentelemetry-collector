// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilesMergeTo(t *testing.T) {
	src := NewProfiles()
	dest := NewProfiles()

	src.ResourceProfiles().AppendEmpty()
	src.ResourceProfiles().AppendEmpty()
	dest.ResourceProfiles().AppendEmpty()

	require.NoError(t, src.MergeTo(dest))

	assert.Equal(t, 3, dest.ResourceProfiles().Len())
	assert.Equal(t, 0, src.ResourceProfiles().Len())
	assert.True(t, src.IsReadOnly())
}

func TestProfilesMergeToSelf(t *testing.T) {
	profiles := NewProfiles()
	profiles.Dictionary().StringTable().Append("", "test")
	profiles.ResourceProfiles().AppendEmpty()

	require.NoError(t, profiles.MergeTo(profiles))

	assert.Equal(t, 2, profiles.Dictionary().StringTable().Len())
	assert.Equal(t, 1, profiles.ResourceProfiles().Len())
}

func TestProfilesMergeToError(t *testing.T) {
	src := NewProfiles()
	dest := NewProfiles()

	stackTable := src.Dictionary().StackTable()
	stackTable.AppendEmpty()
	stack := stackTable.AppendEmpty()
	stack.LocationIndices().Append(1)

	locationTable := src.Dictionary().LocationTable()
	locationTable.AppendEmpty()
	locationTable.AppendEmpty().SetMappingIndex(1)

	sample := src.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty().
		Samples().AppendEmpty()
	sample.SetStackIndex(1)

	err := src.MergeTo(dest)
	require.Error(t, err)

	assert.Equal(t, 0, dest.ResourceProfiles().Len())
}
