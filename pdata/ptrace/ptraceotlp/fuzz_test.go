// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ptraceotlp // import "go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"

import (
	"bytes"
	"testing"
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
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		er = NewExportRequest()
		err = er.UnmarshalJSON(b1)
		if err != nil {
			t.Fatalf("failed to unmarshal valid bytes:  %v", err)
		}
		b2, err := er.MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		if !bytes.Equal(b1, b2) {
			t.Fatalf("%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
		}
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
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		er = NewExportResponse()
		err = er.UnmarshalJSON(b1)
		if err != nil {
			t.Fatalf("failed to unmarshal valid bytes:  %v", err)
		}
		b2, err := er.MarshalJSON()
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		if !bytes.Equal(b1, b2) {
			t.Fatalf("%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
		}
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
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		er = NewExportRequest()
		err = er.UnmarshalProto(b1)
		if err != nil {
			t.Fatalf("failed to unmarshal valid bytes:  %v", err)
		}
		b2, err := er.MarshalProto()
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		if !bytes.Equal(b1, b2) {
			t.Fatalf("%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
		}
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
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		er = NewExportResponse()
		err = er.UnmarshalProto(b1)
		if err != nil {
			t.Fatalf("failed to unmarshal valid bytes:  %v", err)
		}
		b2, err := er.MarshalProto()
		if err != nil {
			t.Fatalf("failed to marshal valid struct:  %v", err)
		}

		if !bytes.Equal(b1, b2) {
			t.Fatalf("%s. \nexpected %d but got %d\n", unexpectedBytes, b1, b2)
		}
	})
}
