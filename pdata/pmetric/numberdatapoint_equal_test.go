// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNumberDataPointEqual(t *testing.T) {
	t.Run("BackwardCompatibility", func(t *testing.T) {
		// Test that Equal without options works exactly as before
		dp1 := NewNumberDataPoint()
		dp1.SetIntValue(42)
		dp1.SetStartTimestamp(pcommon.Timestamp(1234567890))
		dp1.SetTimestamp(pcommon.Timestamp(1234567891))

		dp2 := NewNumberDataPoint()
		dp2.SetIntValue(42)
		dp2.SetStartTimestamp(pcommon.Timestamp(1234567890))
		dp2.SetTimestamp(pcommon.Timestamp(1234567891))

		dp3 := NewNumberDataPoint()
		dp3.SetIntValue(43) // Different value
		dp3.SetStartTimestamp(pcommon.Timestamp(1234567890))
		dp3.SetTimestamp(pcommon.Timestamp(1234567891))

		assert.True(t, dp1.Equal(dp2), "identical NumberDataPoints should be equal")
		assert.False(t, dp1.Equal(dp3), "different NumberDataPoints should not be equal")
	})

	t.Run("BothIntValues", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetIntValue(42)

		dp2 := NewNumberDataPoint()
		dp2.SetIntValue(42)

		assert.True(t, dp1.Equal(dp2), "same int values should be equal")
	})

	t.Run("BothDoubleValues", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetDoubleValue(3.14159)

		dp2 := NewNumberDataPoint()
		dp2.SetDoubleValue(3.14159)

		assert.True(t, dp1.Equal(dp2), "same double values should be equal")
	})

	t.Run("MixedTypes", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetIntValue(42)

		dp2 := NewNumberDataPoint()
		dp2.SetDoubleValue(42.0)

		assert.False(t, dp1.Equal(dp2), "different value types should not be equal")
	})

	t.Run("DifferentIntValues", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetIntValue(42)

		dp2 := NewNumberDataPoint()
		dp2.SetIntValue(43)

		assert.False(t, dp1.Equal(dp2), "different int values should not be equal")
	})

	t.Run("DifferentDoubleValues", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetDoubleValue(3.14159)

		dp2 := NewNumberDataPoint()
		dp2.SetDoubleValue(2.71828)

		assert.False(t, dp1.Equal(dp2), "different double values should not be equal")
	})

	t.Run("StrictDoubleComparison", func(t *testing.T) {
		// Test that without CompareOption, double comparison is strict
		dp1 := NewNumberDataPoint()
		dp1.SetDoubleValue(1.234567)

		dp2 := NewNumberDataPoint()
		dp2.SetDoubleValue(1.234568) // Very small difference

		assert.False(t, dp1.Equal(dp2), "strict double comparison should detect small differences")
	})

	t.Run("DifferentAttributes", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetIntValue(42)
		dp1.Attributes().PutStr("key1", "value1")

		dp2 := NewNumberDataPoint()
		dp2.SetIntValue(42)
		dp2.Attributes().PutStr("key1", "value2") // Different attribute value

		assert.False(t, dp1.Equal(dp2), "different attributes should not be equal")
	})

	t.Run("WithExemplars", func(t *testing.T) {
		dp1 := NewNumberDataPoint()
		dp1.SetIntValue(42)
		ex1 := dp1.Exemplars().AppendEmpty()
		ex1.SetIntValue(10)

		dp2 := NewNumberDataPoint()
		dp2.SetIntValue(42)
		ex2 := dp2.Exemplars().AppendEmpty()
		ex2.SetIntValue(10)

		assert.True(t, dp1.Equal(dp2), "same exemplars should be equal")

		// Different exemplar value
		ex2.SetIntValue(20)
		assert.False(t, dp1.Equal(dp2), "different exemplars should not be equal")
	})
}
