// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestBuildAttributes(t *testing.T) {
	profile := NewProfile()
	att := profile.AttributeTable().AppendEmpty()
	att.SetKey("hello")
	att.Value().SetStr("world")
	att2 := profile.AttributeTable().AppendEmpty()
	att2.SetKey("bonjour")
	att2.Value().SetStr("monde")

	attrs, err := BuildAttributes(profile, profile)
	require.NoError(t, err)
	assert.Equal(t, attrs, pcommon.NewMap())

	// A Location with a single attribute
	loc := NewLocation()
	loc.AttributeIndices().Append(1)

	attrs, err = BuildAttributes(profile, loc)
	require.NoError(t, err)

	m := pcommon.NewMap()
	require.NoError(t, m.FromRaw(map[string]any{"hello": "world"}))
	assert.Equal(t, attrs, m)

	// A Mapping with two attributes
	mapp := NewLocation()
	mapp.AttributeIndices().Append(1, 2)

	attrs, err = BuildAttributes(profile, mapp)
	require.NoError(t, err)

	m = pcommon.NewMap()
	require.NoError(t, m.FromRaw(map[string]any{"hello": "world", "bonjour": "monde"}))
	assert.Equal(t, attrs, m)
}
