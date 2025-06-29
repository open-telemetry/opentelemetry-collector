// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"testing"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func BenchmarkCopyOrigSlice(b *testing.B) {
	tests := []struct {
		name   string
		srcLen int
	}{
		{name: "0_to_7", srcLen: 0},
		{name: "1_to_7", srcLen: 1},
		{name: "7_to_7", srcLen: 7},
		{name: "10_to_7", srcLen: 10},
		{name: "20_to_7", srcLen: 20},
		{name: "50_to_7", srcLen: 50},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			src := make([]otlpcommon.AnyValue, tt.srcLen)
			for i := 0; i < tt.srcLen; i++ {
				src[i] = otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "test_value"}}
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				dest := GenerateTestSlice()
				CopyOrigSlice(*dest.orig, src)
			}
		})
	}
}
