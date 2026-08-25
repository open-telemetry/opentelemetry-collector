// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/internal/telemetry"

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func TestToZapFields(t *testing.T) {
	tests := []struct {
		attrs    attribute.Set
		expected []zap.Field
	}{
		{
			attrs: attribute.NewSet(
				attribute.String("string_key", "string_value"),
			),
			expected: []zap.Field{
				zap.String("string_key", "string_value"),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Bool("bool_key", true),
			),
			expected: []zap.Field{
				zap.Bool("bool_key", true),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Int64("int64_key", 42),
			),
			expected: []zap.Field{
				zap.Int64("int64_key", 42),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Float64("float64_key", 3.14),
			),
			expected: []zap.Field{
				zap.Float64("float64_key", 3.14),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.BoolSlice("bool_slice_key", []bool{true, false, true}),
			),
			expected: []zap.Field{
				zap.Bools("bool_slice_key", []bool{true, false, true}),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Int64Slice("int64_slice_key", []int64{1, 2, 3}),
			),
			expected: []zap.Field{
				zap.Int64s("int64_slice_key", []int64{1, 2, 3}),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.Float64Slice("float64_slice_key", []float64{1.1, 2.2, 3.3}),
			),
			expected: []zap.Field{
				zap.Float64s("float64_slice_key", []float64{1.1, 2.2, 3.3}),
			},
		},
		{
			attrs: attribute.NewSet(
				attribute.StringSlice("string_slice_key", []string{"a", "b", "c"}),
			),
			expected: []zap.Field{
				zap.Strings("string_slice_key", []string{"a", "b", "c"}),
			},
		},
	}

	for _, tt := range tests {
		name := "<empty>"
		if tt.attrs.Len() > 0 {
			attr, ok := tt.attrs.Get(0)
			if ok {
				name = string(attr.Key)
			}
		}
		t.Run(name, func(t *testing.T) {
			result := ToZapFields(tt.attrs.ToSlice())
			require.Equal(t, tt.expected, result)
		})
	}
}
