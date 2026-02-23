// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	stdjson "encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProfilesWithAttributes creates a Profiles with resource and scope
// attributes for testing reference conversion.
func newProfilesWithAttributes() Profiles {
	profiles := NewProfiles()
	profiles.Dictionary().StringTable().Append("") // index 0

	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("service.name", "test-service")
	rp.Resource().Attributes().PutStr("host.name", "test-host")

	sp := rp.ScopeProfiles().AppendEmpty()
	sp.Scope().Attributes().PutStr("scope.attr", "scope-value")

	return profiles
}

func TestJSONMarshalConvertsToReferences(t *testing.T) {
	marshaler := JSONMarshaler{}
	jsonBytes, err := marshaler.MarshalProfiles(newProfilesWithAttributes())
	require.NoError(t, err)

	// Parse the JSON output to verify references were used
	var parsed map[string]any
	require.NoError(t, stdjson.Unmarshal(jsonBytes, &parsed))

	// The dictionary's stringTable should contain the attribute keys and values
	dictionary, ok := parsed["dictionary"].(map[string]any)
	require.True(t, ok, "JSON output should contain a dictionary object")
	stringTable, ok := dictionary["stringTable"].([]any)
	require.True(t, ok, "dictionary should contain a stringTable array")

	tableStrs := make([]string, len(stringTable))
	for i, v := range stringTable {
		tableStrs[i], _ = v.(string)
	}
	assert.Contains(t, tableStrs, "service.name")
	assert.Contains(t, tableStrs, "test-service")
	assert.Contains(t, tableStrs, "host.name")
	assert.Contains(t, tableStrs, "test-host")
	assert.Contains(t, tableStrs, "scope.attr")
	assert.Contains(t, tableStrs, "scope-value")
}

func TestJSONUnmarshalResolvesReferences(t *testing.T) {
	profiles := newProfilesWithAttributes()

	// Manually convert to references before marshaling, so the JSON output
	// contains key_ref/string_value_ref regardless of whether the JSON
	// marshaler itself calls convertProfilesToReferences.
	convertProfilesToReferences(profiles)

	marshaler := JSONMarshaler{}
	jsonBytes, err := marshaler.MarshalProfiles(profiles)
	require.NoError(t, err)

	// Unmarshal and verify references were resolved
	unmarshaler := JSONUnmarshaler{}
	restored, err := unmarshaler.UnmarshalProfiles(jsonBytes)
	require.NoError(t, err)

	rp := restored.ResourceProfiles().At(0)
	serviceNameVal, ok := rp.Resource().Attributes().Get("service.name")
	assert.True(t, ok, "service.name attribute should be accessible after JSON unmarshal")
	assert.Equal(t, "test-service", serviceNameVal.Str())

	hostNameVal, ok := rp.Resource().Attributes().Get("host.name")
	assert.True(t, ok, "host.name attribute should be accessible after JSON unmarshal")
	assert.Equal(t, "test-host", hostNameVal.Str())

	sp := rp.ScopeProfiles().At(0)
	scopeAttrVal, ok := sp.Scope().Attributes().Get("scope.attr")
	assert.True(t, ok, "scope.attr should be accessible after JSON unmarshal")
	assert.Equal(t, "scope-value", scopeAttrVal.Str())
}
