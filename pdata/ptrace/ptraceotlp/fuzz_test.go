// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp // import "go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var unexpectedBytes = "expected the same bytes from unmarshaling and marshaling."

func FuzzRequestUnmarshalJSON(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		er := NewExportRequest()
		err := er.UnmarshalJSON(data)
		if err != nil {
			return
		}
		b1, err := er.MarshalJSON()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		er = NewExportRequest()
		require.NoErrorf(t, er.UnmarshalJSON(b1), "failed to unmarshal valid bytes")
		b2, err := er.MarshalJSON()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		require.Truef(t, bytes.Equal(b1, b2), "%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
	})
}

func FuzzResponseUnmarshalJSON(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		er := NewExportResponse()
		err := er.UnmarshalJSON(data)
		if err != nil {
			return
		}
		b1, err := er.MarshalJSON()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		er = NewExportResponse()
		require.NoErrorf(t, er.UnmarshalJSON(b1), "failed to unmarshal valid bytes")
		b2, err := er.MarshalJSON()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		require.Truef(t, bytes.Equal(b1, b2), "%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
	})
}

func FuzzRequestUnmarshalProto(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		er := NewExportRequest()
		err := er.UnmarshalProto(data)
		if err != nil {
			return
		}
		b1, err := er.MarshalProto()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		er = NewExportRequest()
		require.NoErrorf(t, er.UnmarshalProto(b1), "failed to unmarshal valid bytes")
		b2, err := er.MarshalProto()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		require.Truef(t, bytes.Equal(b1, b2), "%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
	})
}

func FuzzResponseUnmarshalProto(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		er := NewExportResponse()
		err := er.UnmarshalProto(data)
		if err != nil {
			return
		}
		b1, err := er.MarshalProto()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		er = NewExportResponse()
		require.NoErrorf(t, er.UnmarshalProto(b1), "failed to unmarshal valid bytes")
		b2, err := er.MarshalProto()
		require.NoErrorf(t, err, "failed to marshal valid struct")

		require.Truef(t, bytes.Equal(b1, b2), "%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
	})
}
