// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptrace // import "go.opentelemetry.io/collector/pdata/ptrace"

import (
	"bytes"
	"testing"
)

var unexpectedBytes = "expected the same bytes from unmarshaling and marshaling."

func FuzzUnmarshalJSONTraces(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		u1 := &JSONUnmarshaler{}
		ld1, err := u1.UnmarshalTraces(data)
		if err != nil {
			return
		}
		m1 := &JSONMarshaler{}
		b1, err := m1.MarshalTraces(ld1)
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		u2 := &JSONUnmarshaler{}
		ld2, err := u2.UnmarshalTraces(b1)
		if err != nil {
			t.Fatalf("failed to unmarshal valid bytes:  %v", err)
		}
		m2 := &JSONMarshaler{}
		b2, err := m2.MarshalTraces(ld2)
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		if !bytes.Equal(b1, b2) {
			t.Fatalf("%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
		}
	})
}

func FuzzUnmarshalPBTraces(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		u1 := &ProtoUnmarshaler{}
		ld1, err := u1.UnmarshalTraces(data)
		if err != nil {
			return
		}
		m1 := &ProtoMarshaler{}
		b1, err := m1.MarshalTraces(ld1)
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		u2 := &ProtoUnmarshaler{}
		ld2, err := u2.UnmarshalTraces(b1)
		if err != nil {
			t.Fatalf("failed to unmarshal valid bytes:  %v", err)
		}
		m2 := &ProtoMarshaler{}
		b2, err := m2.MarshalTraces(ld2)
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		if !bytes.Equal(b1, b2) {
			t.Fatalf("%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
		}
	})
}
