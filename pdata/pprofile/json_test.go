// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONUnmarshalerDisallowUnknownFields(t *testing.T) {
	input := []byte(`{"resourceProfiles":[],"unexpected":"value"}`)

	t.Run("default ignores unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{}
		pd, err := u.UnmarshalProfiles(input)
		require.NoError(t, err)
		assert.Equal(t, 0, pd.ResourceProfiles().Len())
	})

	t.Run("strict mode errors on unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalProfiles(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode errors on nested unknown fields", func(t *testing.T) {
		nested := []byte(`{"resourceProfiles":[{"resource":{},"unexpected":"value"}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalProfiles(nested)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode accepts valid OTLP JSON", func(t *testing.T) {
		valid := []byte(`{"resourceProfiles":[{"resource":{}}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		pd, err := u.UnmarshalProfiles(valid)
		require.NoError(t, err)
		assert.Equal(t, 1, pd.ResourceProfiles().Len())
	})
}
