// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONUnmarshalerDisallowUnknownFields(t *testing.T) {
	input := []byte(`{"resourceSpans":[],"unexpected":"value"}`)

	t.Run("default ignores unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{}
		td, err := u.UnmarshalTraces(input)
		require.NoError(t, err)
		assert.Equal(t, 0, td.ResourceSpans().Len())
	})

	t.Run("strict mode errors on unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalTraces(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode errors on nested unknown fields", func(t *testing.T) {
		nested := []byte(`{"resourceSpans":[{"resource":{},"unexpected":"value"}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalTraces(nested)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode accepts valid OTLP JSON", func(t *testing.T) {
		valid := []byte(`{"resourceSpans":[{"resource":{}}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		td, err := u.UnmarshalTraces(valid)
		require.NoError(t, err)
		assert.Equal(t, 1, td.ResourceSpans().Len())
	})

	t.Run("strict mode reports only the first unknown field", func(t *testing.T) {
		multi := []byte(`{"unexpected1":"a","unexpected2":"b"}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalTraces(multi)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected1"`)
		assert.NotContains(t, err.Error(), `unknown field "unexpected2"`)
	})
}
