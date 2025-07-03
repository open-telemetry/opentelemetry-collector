// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareOptions(t *testing.T) {
	t.Run("FloatTolerance option", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{FloatTolerance(0.001)})
		assert.InDelta(t, 0.001, cfg.floatTolerance, 1e-9)
	})

	t.Run("IgnoreFields option", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{IgnoreFields("field1", "field2")})
		assert.True(t, cfg.ShouldIgnoreField("field1"))
		assert.True(t, cfg.ShouldIgnoreField("field2"))
		assert.False(t, cfg.ShouldIgnoreField("field3"))
	})

	t.Run("IgnorePaths option", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{IgnorePaths("path1.subpath", "path2.*")})
		assert.True(t, cfg.ShouldIgnorePath("path1.subpath"))
		assert.True(t, cfg.ShouldIgnorePath("path2.anything"))
		assert.False(t, cfg.ShouldIgnorePath("path3.subpath"))
	})

	t.Run("Multiple options", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{
			FloatTolerance(0.01),
			IgnoreFields("timestamp"),
			IgnorePaths("nested.*"),
		})
		assert.InDelta(t, 0.01, cfg.floatTolerance, 1e-9)
		assert.True(t, cfg.ShouldIgnoreField("timestamp"))
		assert.True(t, cfg.ShouldIgnorePath("nested.field"))
	})
}

func TestCompareFloat64(t *testing.T) {
	t.Run("Strict equality when no tolerance", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{})

		// Same values should be equal
		assert.True(t, cfg.CompareFloat64(1.234567, 1.234567))

		// Different values should not be equal (strict)
		assert.False(t, cfg.CompareFloat64(1.234567, 1.234568))
		assert.False(t, cfg.CompareFloat64(1.0, 1.0000001))
	})

	t.Run("Tolerance comparison when tolerance set", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{FloatTolerance(0.001)})

		// Within tolerance should be equal
		assert.True(t, cfg.CompareFloat64(1.234567, 1.234568)) // diff = 0.000001 < 0.001
		assert.True(t, cfg.CompareFloat64(1.0, 1.0005))        // diff = 0.0005 < 0.001

		// Outside tolerance should not be equal
		assert.False(t, cfg.CompareFloat64(1.0, 1.002))         // diff = 0.002 > 0.001
		assert.False(t, cfg.CompareFloat64(1.234567, 1.235567)) // diff = 0.001 not < 0.001
	})

	t.Run("Zero tolerance means strict equality", func(t *testing.T) {
		cfg := NewCompareConfig([]CompareOption{FloatTolerance(0)})

		assert.True(t, cfg.CompareFloat64(1.234567, 1.234567))
		assert.False(t, cfg.CompareFloat64(1.234567, 1.234568))
	})
}

func TestValueEqualWithOptions(t *testing.T) {
	t.Run("Float64 values backward compatibility", func(t *testing.T) {
		v1 := NewValueDouble(1.234567)
		v2 := NewValueDouble(1.234567)
		v3 := NewValueDouble(1.234568)

		// Same values should be equal
		assert.True(t, v1.Equal(v2))

		// Different values should not be equal without options (strict)
		assert.False(t, v1.Equal(v3))

		// Different values should be equal with tolerance
		assert.True(t, v1.Equal(v3, FloatTolerance(0.001)))
	})

	t.Run("String values", func(t *testing.T) {
		v1 := NewValueStr("hello")
		v2 := NewValueStr("hello")
		v3 := NewValueStr("world")

		assert.True(t, v1.Equal(v2))
		assert.False(t, v1.Equal(v3))
		assert.False(t, v1.Equal(v3, FloatTolerance(0.001))) // tolerance should not affect strings
	})
}

func TestFloat64SliceEqualWithOptions(t *testing.T) {
	t.Run("Backward compatibility - strict equality", func(t *testing.T) {
		slice1 := NewFloat64Slice()
		slice1.Append(1.234567, 2.345678)

		slice2 := NewFloat64Slice()
		slice2.Append(1.234567, 2.345678)

		slice3 := NewFloat64Slice()
		slice3.Append(1.234568, 2.345678) // slightly different first value

		// Same values should be equal
		assert.True(t, slice1.Equal(slice2))

		// Different values should not be equal without options (strict)
		assert.False(t, slice1.Equal(slice3))

		// Different values should be equal with tolerance
		assert.True(t, slice1.Equal(slice3, FloatTolerance(0.001)))
	})
}
