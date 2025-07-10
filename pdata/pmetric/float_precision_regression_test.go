// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestHistogramDataPointFloatPrecisionRegression(t *testing.T) {
	t.Run("sum_precision_issue", func(t *testing.T) {
		dp1 := NewHistogramDataPoint()
		dp1.SetCount(75)
		dp1.SetSum(1345.32) // Exact value
		dp1.ExplicitBounds().FromRaw([]float64{0, 5, 10, 25, 50})

		dp2 := NewHistogramDataPoint()
		dp2.SetCount(75)
		dp2.SetSum(1345.3200000000002) // Value with floating point precision error
		dp2.ExplicitBounds().FromRaw([]float64{0, 5, 10, 25, 50})

		assert.False(t, dp1.Equal(dp2), "Should detect the precision difference with strict comparison")

		// ExplicitBounds should be equal since they have identical values
		assert.True(t, dp1.ExplicitBounds().Equal(dp2.ExplicitBounds()), "ExplicitBounds should be equal")
	})

	t.Run("explicit_bounds_precision_issue", func(t *testing.T) {
		// Test Float64Slice precision directly
		bounds1 := pcommon.NewFloat64Slice()
		bounds1.FromRaw([]float64{0.0, 1.5, 3.0, 7.5, 15.0})

		bounds2 := pcommon.NewFloat64Slice()
		// Simulate floating point calculation errors
		bounds2.FromRaw([]float64{0.0, 1.5000000000000002, 3.0000000000000004, 7.5, 15.000000000000002})

		// Strict comparison should detect differences
		assert.False(t, bounds1.Equal(bounds2), "Strict comparison should detect small differences")

		// With tolerance, should be equal
		assert.True(t, bounds1.Equal(bounds2, pcommon.FloatTolerance(1e-10)), "Should be equal with reasonable tolerance")
	})

	t.Run("regression_test_complex_case", func(t *testing.T) {
		dp1 := NewHistogramDataPoint()
		dp1.SetCount(75)
		dp1.SetSum(1345.32)
		dp1.SetStartTimestamp(pcommon.Timestamp(1234567890))
		dp1.SetTimestamp(pcommon.Timestamp(1234567900))
		dp1.ExplicitBounds().FromRaw([]float64{0, 5, 10, 25, 50, 100, 250, 500, 1000, 2500})
		dp1.BucketCounts().FromRaw([]uint64{10, 15, 20, 10, 8, 5, 3, 2, 1, 1})

		dp2 := NewHistogramDataPoint()
		dp2.SetCount(75)
		dp2.SetSum(1345.3200000000002)
		dp2.SetStartTimestamp(pcommon.Timestamp(1234567890))
		dp2.SetTimestamp(pcommon.Timestamp(1234567900))
		dp2.ExplicitBounds().FromRaw([]float64{0, 5, 10, 25, 50, 100, 250, 500, 1000, 2500})
		dp2.BucketCounts().FromRaw([]uint64{10, 15, 20, 10, 8, 5, 3, 2, 1, 1})

		// This should demonstrate that our backward compatibility fix works
		// The Equal method should use strict comparison by default and detect the difference
		assert.False(t, dp1.Equal(dp2), "Should detect sum difference with strict comparison")

		assert.True(t, dp1.Attributes().Equal(dp2.Attributes()), "Attributes should be equal")
		assert.Equal(t, dp1.StartTimestamp(), dp2.StartTimestamp(), "StartTimestamp should be equal")
		assert.Equal(t, dp1.Timestamp(), dp2.Timestamp(), "Timestamp should be equal")
		assert.Equal(t, dp1.Count(), dp2.Count(), "Count should be equal")
		assert.True(t, dp1.BucketCounts().Equal(dp2.BucketCounts()), "BucketCounts should be equal")
		assert.True(t, dp1.ExplicitBounds().Equal(dp2.ExplicitBounds()), "ExplicitBounds should be equal")
		assert.True(t, dp1.Exemplars().Equal(dp2.Exemplars()), "Exemplars should be equal")
		assert.Equal(t, dp1.Flags(), dp2.Flags(), "Flags should be equal")

		assert.Equal(t, dp1.HasSum(), dp2.HasSum(), "HasSum should be equal")
		assert.NotEqual(t, dp1.Sum(), dp2.Sum(), "Sum values should be different (this is the issue)")

		// Demonstrate that with our new tolerance feature, they could be considered equal
		// (This is not the default behavior, but shows the feature works)
		// Note: We don't provide this functionality for HistogramDataPoint directly,
		// but users could implement their own tolerance-aware comparison if needed
	})

	t.Run("backward_compatibility_verification", func(t *testing.T) {
		dp1 := NewHistogramDataPoint()
		dp1.SetCount(100)
		dp1.SetSum(2500.75)
		dp1.ExplicitBounds().FromRaw([]float64{0, 10, 50, 100, 500})

		dp2 := NewHistogramDataPoint()
		dp2.SetCount(100)
		dp2.SetSum(2500.75) // Exactly the same
		dp2.ExplicitBounds().FromRaw([]float64{0, 10, 50, 100, 500})

		// These should be equal
		assert.True(t, dp1.Equal(dp2), "Identical HistogramDataPoints should be equal")
		assert.True(t, dp1.ExplicitBounds().Equal(dp2.ExplicitBounds()), "Identical ExplicitBounds should be equal")
	})
}

func TestFloat64SliceBackwardCompatibility(t *testing.T) {
	t.Run("identical_values", func(t *testing.T) {
		slice1 := pcommon.NewFloat64Slice()
		slice1.FromRaw([]float64{1.0, 2.0, 3.0})

		slice2 := pcommon.NewFloat64Slice()
		slice2.FromRaw([]float64{1.0, 2.0, 3.0})

		// Should be equal without any options (backward compatibility)
		assert.True(t, slice1.Equal(slice2), "Identical slices should be equal")
	})

	t.Run("different_values", func(t *testing.T) {
		slice1 := pcommon.NewFloat64Slice()
		slice1.FromRaw([]float64{1.0, 2.0, 3.0})

		slice2 := pcommon.NewFloat64Slice()
		slice2.FromRaw([]float64{1.0, 2.0, 3.1})

		// Should not be equal without tolerance
		assert.False(t, slice1.Equal(slice2), "Different slices should not be equal")
	})

	t.Run("precision_differences", func(t *testing.T) {
		slice1 := pcommon.NewFloat64Slice()
		slice1.FromRaw([]float64{1345.32})

		slice2 := pcommon.NewFloat64Slice()
		slice2.FromRaw([]float64{1345.3200000000002})

		// Should not be equal with strict comparison
		assert.False(t, slice1.Equal(slice2), "Should detect precision differences with strict comparison")

		// Should be equal with tolerance
		assert.True(t, slice1.Equal(slice2, pcommon.FloatTolerance(1e-10)), "Should be equal with tolerance")
	})
}
