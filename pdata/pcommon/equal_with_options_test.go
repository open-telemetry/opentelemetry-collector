// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueEqualWithOptions(t *testing.T) {
	t.Run("Float64 values with tolerance", func(t *testing.T) {
		v1 := NewValueDouble(1.234567)
		v2 := NewValueDouble(1.234568)
		v3 := NewValueDouble(1.235567)

		// Strict equality (default)
		//nolint:gocritic // Testing reflexivity is intentional
		assert.True(t, v1.Equal(v1))
		assert.False(t, v1.Equal(v2))
		assert.False(t, v1.Equal(v3))

		// With tolerance
		assert.True(t, v1.Equal(v2, FloatTolerance(0.001)))  // within tolerance
		assert.False(t, v1.Equal(v3, FloatTolerance(0.001))) // outside tolerance
		assert.True(t, v1.Equal(v3, FloatTolerance(0.002)))  // within larger tolerance
	})

	t.Run("Different value types", func(t *testing.T) {
		vStr := NewValueStr("hello")
		vInt := NewValueInt(42)
		vFloat := NewValueDouble(1.23)

		// Different types should not be equal regardless of options
		assert.False(t, vStr.Equal(vInt))
		assert.False(t, vStr.Equal(vInt, FloatTolerance(1.0)))
		assert.False(t, vInt.Equal(vFloat))
		assert.False(t, vInt.Equal(vFloat, FloatTolerance(100.0)))
	})

	t.Run("Map values with nested floats", func(t *testing.T) {
		m1 := NewValueMap()
		m1.Map().PutDouble("temperature", 23.456)
		m1.Map().PutStr("unit", "celsius")

		m2 := NewValueMap()
		m2.Map().PutDouble("temperature", 23.457) // slight difference
		m2.Map().PutStr("unit", "celsius")

		// Strict equality
		assert.False(t, m1.Equal(m2))

		// With tolerance
		assert.True(t, m1.Equal(m2, FloatTolerance(0.01)))
		assert.False(t, m1.Equal(m2, FloatTolerance(0.0005)))
	})

	t.Run("Slice values with nested floats", func(t *testing.T) {
		s1 := NewValueSlice()
		s1.Slice().AppendEmpty().SetDouble(1.234)
		s1.Slice().AppendEmpty().SetStr("test")
		s1.Slice().AppendEmpty().SetDouble(5.678)

		s2 := NewValueSlice()
		s2.Slice().AppendEmpty().SetDouble(1.235) // slight difference
		s2.Slice().AppendEmpty().SetStr("test")
		s2.Slice().AppendEmpty().SetDouble(5.679) // slight difference

		// Strict equality
		assert.False(t, s1.Equal(s2))

		// With tolerance
		assert.True(t, s1.Equal(s2, FloatTolerance(0.01)))
		assert.False(t, s1.Equal(s2, FloatTolerance(0.0005)))
	})
}

func TestFloat64SliceEqualWithOptions(t *testing.T) {
	t.Run("Backward compatibility", func(t *testing.T) {
		slice1 := NewFloat64Slice()
		slice1.Append(1.234567, 2.345678, 3.456789)

		slice2 := NewFloat64Slice()
		slice2.Append(1.234567, 2.345678, 3.456789)

		slice3 := NewFloat64Slice()
		slice3.Append(1.234568, 2.345679, 3.456790) // slight differences

		// Same values should be equal
		assert.True(t, slice1.Equal(slice2))

		// Different values should not be equal without options
		assert.False(t, slice1.Equal(slice3))

		// Different values should be equal with tolerance
		assert.True(t, slice1.Equal(slice3, FloatTolerance(0.001)))
		assert.False(t, slice1.Equal(slice3, FloatTolerance(0.0000005)))
	})

	t.Run("Edge cases", func(t *testing.T) {
		slice1 := NewFloat64Slice()
		slice1.Append(0.0, -1.5, 1000000.123)

		slice2 := NewFloat64Slice()
		slice2.Append(0.0000001, -1.5000001, 1000000.124) // tiny differences

		assert.False(t, slice1.Equal(slice2))                      // strict
		assert.True(t, slice1.Equal(slice2, FloatTolerance(0.01))) // with tolerance
	})

	t.Run("Different lengths", func(t *testing.T) {
		slice1 := NewFloat64Slice()
		slice1.Append(1.0, 2.0)

		slice2 := NewFloat64Slice()
		slice2.Append(1.0, 2.0, 3.0)

		// Different lengths should never be equal
		assert.False(t, slice1.Equal(slice2))
		assert.False(t, slice1.Equal(slice2, FloatTolerance(1.0)))
	})
}

func TestMapEqualWithOptions(t *testing.T) {
	t.Run("Float tolerance in map values", func(t *testing.T) {
		m1 := NewMap()
		m1.PutDouble("temperature", 23.456)
		m1.PutDouble("humidity", 60.123)
		m1.PutStr("location", "office")

		m2 := NewMap()
		m2.PutDouble("temperature", 23.457) // slight difference
		m2.PutDouble("humidity", 60.124)    // slight difference
		m2.PutStr("location", "office")

		// Strict equality
		assert.False(t, m1.Equal(m2))

		// With tolerance
		assert.True(t, m1.Equal(m2, FloatTolerance(0.01)))
		assert.False(t, m1.Equal(m2, FloatTolerance(0.0005)))
	})

	t.Run("Nested maps with floats", func(t *testing.T) {
		m1 := NewMap()
		nested1 := m1.PutEmptyMap("sensor")
		nested1.PutDouble("value", 25.123)
		nested1.PutStr("unit", "celsius")

		m2 := NewMap()
		nested2 := m2.PutEmptyMap("sensor")
		nested2.PutDouble("value", 25.124) // slight difference
		nested2.PutStr("unit", "celsius")

		assert.False(t, m1.Equal(m2))                      // strict
		assert.True(t, m1.Equal(m2, FloatTolerance(0.01))) // with tolerance
	})

	t.Run("Different keys", func(t *testing.T) {
		m1 := NewMap()
		m1.PutStr("key1", "value1")

		m2 := NewMap()
		m2.PutStr("key2", "value1")

		// Different keys should never be equal
		assert.False(t, m1.Equal(m2))
		assert.False(t, m1.Equal(m2, FloatTolerance(1.0)))
	})
}

func TestSliceEqualWithOptions(t *testing.T) {
	t.Run("Float tolerance in slice elements", func(t *testing.T) {
		s1 := NewSlice()
		s1.AppendEmpty().SetDouble(1.234)
		s1.AppendEmpty().SetStr("test")
		elem1 := s1.AppendEmpty().SetEmptyMap()
		elem1.PutDouble("nested", 5.678)

		s2 := NewSlice()
		s2.AppendEmpty().SetDouble(1.235) // slight difference
		s2.AppendEmpty().SetStr("test")
		elem2 := s2.AppendEmpty().SetEmptyMap()
		elem2.PutDouble("nested", 5.679) // slight difference

		assert.False(t, s1.Equal(s2))                      // strict
		assert.True(t, s1.Equal(s2, FloatTolerance(0.01))) // with tolerance
	})

	t.Run("Different lengths", func(t *testing.T) {
		s1 := NewSlice()
		s1.AppendEmpty().SetStr("test")

		s2 := NewSlice()
		s2.AppendEmpty().SetStr("test")
		s2.AppendEmpty().SetStr("test2")

		// Different lengths should never be equal
		assert.False(t, s1.Equal(s2))
		assert.False(t, s1.Equal(s2, FloatTolerance(1.0)))
	})
}

func TestResourceEqualWithOptions(t *testing.T) {
	t.Run("Float tolerance in resource attributes", func(t *testing.T) {
		r1 := NewResource()
		r1.Attributes().PutDouble("cpu.usage", 75.123)
		r1.Attributes().PutStr("host.name", "server1")
		r1.SetDroppedAttributesCount(1)

		r2 := NewResource()
		r2.Attributes().PutDouble("cpu.usage", 75.124) // slight difference
		r2.Attributes().PutStr("host.name", "server1")
		r2.SetDroppedAttributesCount(1)

		assert.False(t, r1.Equal(r2))                      // strict
		assert.True(t, r1.Equal(r2, FloatTolerance(0.01))) // with tolerance
	})

	t.Run("Different dropped counts", func(t *testing.T) {
		r1 := NewResource()
		r1.SetDroppedAttributesCount(1)

		r2 := NewResource()
		r2.SetDroppedAttributesCount(2)

		// Different counts should never be equal
		assert.False(t, r1.Equal(r2))
		assert.False(t, r1.Equal(r2, FloatTolerance(1.0)))
	})
}

func TestInstrumentationScopeEqualWithOptions(t *testing.T) {
	t.Run("Float tolerance in scope attributes", func(t *testing.T) {
		s1 := NewInstrumentationScope()
		s1.SetName("test.scope")
		s1.SetVersion("1.0.0")
		s1.Attributes().PutDouble("metric.version", 2.1)
		s1.SetDroppedAttributesCount(0)

		s2 := NewInstrumentationScope()
		s2.SetName("test.scope")
		s2.SetVersion("1.0.0")
		s2.Attributes().PutDouble("metric.version", 2.10001) // slight difference
		s2.SetDroppedAttributesCount(0)

		assert.False(t, s1.Equal(s2))                       // strict
		assert.True(t, s1.Equal(s2, FloatTolerance(0.001))) // with tolerance
	})

	t.Run("Different names", func(t *testing.T) {
		s1 := NewInstrumentationScope()
		s1.SetName("scope1")

		s2 := NewInstrumentationScope()
		s2.SetName("scope2")

		// Different names should never be equal
		assert.False(t, s1.Equal(s2))
		assert.False(t, s1.Equal(s2, FloatTolerance(1.0)))
	})
}

func TestPrimitiveSlicesEqualWithOptions(t *testing.T) {
	t.Run("Int64Slice - options should not affect", func(t *testing.T) {
		s1 := NewInt64Slice()
		s1.Append(1, 2, 3)

		s2 := NewInt64Slice()
		s2.Append(1, 2, 3)

		s3 := NewInt64Slice()
		s3.Append(1, 2, 4)

		assert.True(t, s1.Equal(s2))
		assert.True(t, s1.Equal(s2, FloatTolerance(0.1))) // options ignored for int slices
		assert.False(t, s1.Equal(s3))
		assert.False(t, s1.Equal(s3, FloatTolerance(1.0))) // still false
	})

	t.Run("StringSlice - options should not affect", func(t *testing.T) {
		s1 := NewStringSlice()
		s1.Append("hello", "world")

		s2 := NewStringSlice()
		s2.Append("hello", "world")

		s3 := NewStringSlice()
		s3.Append("hello", "universe")

		assert.True(t, s1.Equal(s2))
		assert.True(t, s1.Equal(s2, FloatTolerance(0.1))) // options ignored
		assert.False(t, s1.Equal(s3))
		assert.False(t, s1.Equal(s3, FloatTolerance(1.0))) // still false
	})

	t.Run("ByteSlice - options should not affect", func(t *testing.T) {
		s1 := NewByteSlice()
		s1.Append(1, 2, 3)

		s2 := NewByteSlice()
		s2.Append(1, 2, 3)

		s3 := NewByteSlice()
		s3.Append(1, 2, 4)

		assert.True(t, s1.Equal(s2))
		assert.True(t, s1.Equal(s2, FloatTolerance(0.1))) // options ignored
		assert.False(t, s1.Equal(s3))
		assert.False(t, s1.Equal(s3, FloatTolerance(1.0))) // still false
	})
}

func TestTraceStateEqualWithOptions(t *testing.T) {
	t.Run("TraceState ignores options", func(t *testing.T) {
		ts1 := NewTraceState()
		ts1.FromRaw("key1=value1,key2=value2")

		ts2 := NewTraceState()
		ts2.FromRaw("key1=value1,key2=value2")

		ts3 := NewTraceState()
		ts3.FromRaw("key1=value1,key2=value3")

		assert.True(t, ts1.Equal(ts2))
		assert.True(t, ts1.Equal(ts2, FloatTolerance(0.1))) // options ignored
		assert.False(t, ts1.Equal(ts3))
		assert.False(t, ts1.Equal(ts3, FloatTolerance(1.0))) // still false
	})
}

// Test complex nested scenarios
func TestComplexNestedEqualWithOptions(t *testing.T) {
	t.Run("Deeply nested structures with floats", func(t *testing.T) {
		// Create complex nested structure
		m1 := NewMap()
		m1.PutStr("service", "api")
		metrics1 := m1.PutEmptySlice("metrics")
		metric1 := metrics1.AppendEmpty().SetEmptyMap()
		metric1.PutStr("name", "cpu.usage")
		metric1.PutDouble("value", 75.123456)
		samples1 := metric1.PutEmptySlice("samples")
		samples1.AppendEmpty().SetDouble(75.123)
		samples1.AppendEmpty().SetDouble(75.124)

		// Create similar structure with slight float differences
		m2 := NewMap()
		m2.PutStr("service", "api")
		metrics2 := m2.PutEmptySlice("metrics")
		metric2 := metrics2.AppendEmpty().SetEmptyMap()
		metric2.PutStr("name", "cpu.usage")
		metric2.PutDouble("value", 75.123457) // tiny difference
		samples2 := metric2.PutEmptySlice("samples")
		samples2.AppendEmpty().SetDouble(75.124) // tiny difference
		samples2.AppendEmpty().SetDouble(75.125) // tiny difference

		// Test with value wrapper
		v1 := NewValueMap()
		m1.CopyTo(v1.Map())
		v2 := NewValueMap()
		m2.CopyTo(v2.Map())

		assert.False(t, v1.Equal(v2))                            // strict comparison fails
		assert.True(t, v1.Equal(v2, FloatTolerance(0.01)))       // tolerance comparison succeeds
		assert.False(t, v1.Equal(v2, FloatTolerance(0.0000005))) // very small tolerance fails
	})
}
