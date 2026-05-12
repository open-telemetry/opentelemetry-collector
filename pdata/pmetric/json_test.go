// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONUnmarshalerDisallowUnknownFields(t *testing.T) {
	input := []byte(`{"resourceMetrics":[],"unexpected":"value"}`)

	t.Run("default ignores unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{}
		md, err := u.UnmarshalMetrics(input)
		require.NoError(t, err)
		assert.Equal(t, 0, md.ResourceMetrics().Len())
	})

	t.Run("strict mode errors on unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalMetrics(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode errors on nested unknown fields", func(t *testing.T) {
		nested := []byte(`{"resourceMetrics":[{"resource":{},"unexpected":"value"}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalMetrics(nested)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode accepts valid OTLP JSON", func(t *testing.T) {
		valid := []byte(`{"resourceMetrics":[{"resource":{}}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		md, err := u.UnmarshalMetrics(valid)
		require.NoError(t, err)
		assert.Equal(t, 1, md.ResourceMetrics().Len())
	})
}
