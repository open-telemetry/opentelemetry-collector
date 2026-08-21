// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONUnmarshalerDisallowUnknownFields(t *testing.T) {
	// Input contains a JSON object field ("unexpected") that is not part of
	// the OTLP/JSON schema for LogsData.
	input := []byte(`{"resourceLogs":[],"unexpected":"value"}`)

	t.Run("default ignores unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{}
		ld, err := u.UnmarshalLogs(input)
		require.NoError(t, err)
		assert.Equal(t, 0, ld.ResourceLogs().Len())
	})

	t.Run("strict mode errors on unknown fields", func(t *testing.T) {
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalLogs(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode errors on nested unknown fields", func(t *testing.T) {
		nested := []byte(`{"resourceLogs":[{"resource":{},"unexpected":"value"}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalLogs(nested)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected"`)
	})

	t.Run("strict mode accepts valid OTLP JSON", func(t *testing.T) {
		valid := []byte(`{"resourceLogs":[{"resource":{}}]}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		ld, err := u.UnmarshalLogs(valid)
		require.NoError(t, err)
		assert.Equal(t, 1, ld.ResourceLogs().Len())
	})

	t.Run("strict mode reports only the first unknown field", func(t *testing.T) {
		multi := []byte(`{"unexpected1":"a","unexpected2":"b"}`)
		u := &JSONUnmarshaler{DisallowUnknownFields: true}
		_, err := u.UnmarshalLogs(multi)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown field "unexpected1"`)
		assert.NotContains(t, err.Error(), `unknown field "unexpected2"`)
	})
}
