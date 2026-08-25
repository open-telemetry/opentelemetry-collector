// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var unexpectedBytes = "expected the same bytes from unmarshaling and marshaling."

func FuzzUnmarshalJSONValue(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		u1 := &JSONUnmarshaler{}
		ld1, err := u1.UnmarshalValue(data)
		if err != nil {
			return
		}
		m1 := &JSONMarshaler{}
		b1, err := m1.MarshalValue(ld1)
		require.NoError(t, err, "failed to marshal valid struct")

		u2 := &JSONUnmarshaler{}
		ld2, err := u2.UnmarshalValue(b1)
		require.NoError(t, err, "failed to unmarshal valid bytes")
		m2 := &JSONMarshaler{}
		b2, err := m2.MarshalValue(ld2)
		require.NoError(t, err, "failed to marshal valid struct")

		require.True(t, bytes.Equal(b1, b2), "%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
	})
}
