// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package plogotlp // import "go.opentelemetry.io/collector/pdata/plog/plogotlp"

import (
	"testing"
)

func FuzzRequestUnmarshalJSON(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		er := NewExportRequest()
		//nolint: errcheck
		er.UnmarshalJSON(data)
	})
}

func FuzzResponseUnmarshalJSON(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		er := NewExportResponse()
		//nolint: errcheck
		er.UnmarshalJSON(data)
	})
}

func FuzzRequestUnmarshalProto(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		er := NewExportRequest()
		//nolint: errcheck
		er.UnmarshalProto(data)
	})
}

func FuzzResponseUnmarshalProto(f *testing.F) {
	f.Fuzz(func(_ *testing.T, data []byte) {
		er := NewExportResponse()
		//nolint: errcheck
		er.UnmarshalProto(data)
	})
}
