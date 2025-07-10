// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCompareConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cfg := NewCompareConfig(nil)
		assert.InDelta(t, 0.0, cfg.floatTolerance, 1e-9)
		assert.NotNil(t, cfg.ignoredFields)
		assert.NotNil(t, cfg.ignoredPaths)
		assert.Empty(t, cfg.ignoredFields)
		assert.Empty(t, cfg.ignoredPaths)
	})

	t.Run("with options", func(t *testing.T) {
		opts := []CompareOption{
			FloatTolerance(0.01),
			IgnoreFields("field1", "field2"),
			IgnorePaths("path1", "path2.*"),
		}
		cfg := NewCompareConfig(opts)
		assert.InDelta(t, 0.01, cfg.floatTolerance, 1e-9)
		assert.True(t, cfg.ignoredFields["field1"])
		assert.True(t, cfg.ignoredFields["field2"])
		assert.True(t, cfg.ignoredPaths["path1"])
		assert.True(t, cfg.ignoredPaths["path2.*"])
	})
}

func TestFloatTolerance(t *testing.T) {
	cfg := &CompareConfig{}
	opt := FloatTolerance(0.1)
	opt.apply(cfg)
	assert.InDelta(t, 0.1, cfg.floatTolerance, 1e-9)

	// Zero tolerance should work
	opt2 := FloatTolerance(0.0)
	opt2.apply(cfg)
	assert.InDelta(t, 0.0, cfg.floatTolerance, 1e-9)
}

func TestIgnoreFields(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		opt := IgnoreFields("testField")
		cfg := &CompareConfig{ignoredFields: make(map[string]bool)}
		opt.apply(cfg)
		assert.True(t, cfg.ignoredFields["testField"])
	})

	t.Run("multiple fields", func(t *testing.T) {
		opt := IgnoreFields("field1", "field2", "field3")
		cfg := &CompareConfig{ignoredFields: make(map[string]bool)}
		opt.apply(cfg)
		assert.True(t, cfg.ignoredFields["field1"])
		assert.True(t, cfg.ignoredFields["field2"])
		assert.True(t, cfg.ignoredFields["field3"])
	})

	t.Run("empty fields", func(t *testing.T) {
		opt := IgnoreFields()
		cfg := &CompareConfig{ignoredFields: make(map[string]bool)}
		opt.apply(cfg)
		assert.Empty(t, cfg.ignoredFields)
	})
}

func TestIgnorePaths(t *testing.T) {
	t.Run("single path", func(t *testing.T) {
		opt := IgnorePaths("test.path")
		cfg := &CompareConfig{ignoredPaths: make(map[string]bool)}
		opt.apply(cfg)
		assert.True(t, cfg.ignoredPaths["test.path"])
	})

	t.Run("multiple paths", func(t *testing.T) {
		opt := IgnorePaths("path1", "path2.*", "nested.path")
		cfg := &CompareConfig{ignoredPaths: make(map[string]bool)}
		opt.apply(cfg)
		assert.True(t, cfg.ignoredPaths["path1"])
		assert.True(t, cfg.ignoredPaths["path2.*"])
		assert.True(t, cfg.ignoredPaths["nested.path"])
	})
}

func TestShouldIgnoreField(t *testing.T) {
	cfg := &CompareConfig{
		ignoredFields: map[string]bool{
			"ignoredField": true,
			"anotherField": true,
		},
	}

	t.Run("ignored field", func(t *testing.T) {
		assert.True(t, cfg.ShouldIgnoreField("ignoredField"))
		assert.True(t, cfg.ShouldIgnoreField("anotherField"))
	})

	t.Run("not ignored field", func(t *testing.T) {
		assert.False(t, cfg.ShouldIgnoreField("normalField"))
		assert.False(t, cfg.ShouldIgnoreField(""))
	})
}

func TestShouldIgnorePath(t *testing.T) {
	cfg := &CompareConfig{
		ignoredPaths: map[string]bool{
			"exact.path":   true,
			"wildcard.*":   true,
			"nested.sub.*": true,
		},
	}

	t.Run("exact path match", func(t *testing.T) {
		assert.True(t, cfg.ShouldIgnorePath("exact.path"))
	})

	t.Run("wildcard match", func(t *testing.T) {
		assert.True(t, cfg.ShouldIgnorePath("wildcard.anything"))
		assert.True(t, cfg.ShouldIgnorePath("wildcard.sub.field"))
		assert.True(t, cfg.ShouldIgnorePath("nested.sub.anything"))
	})

	t.Run("no match", func(t *testing.T) {
		assert.False(t, cfg.ShouldIgnorePath("different.path"))
		assert.False(t, cfg.ShouldIgnorePath("wildcar.test")) // typo, should not match
		assert.False(t, cfg.ShouldIgnorePath(""))
	})

	t.Run("partial wildcard match should not work", func(t *testing.T) {
		assert.False(t, cfg.ShouldIgnorePath("wild"))   // should not match "wildcard.*"
		assert.False(t, cfg.ShouldIgnorePath("nested")) // should not match "nested.sub.*"
	})
}

func TestCompareFloat64(t *testing.T) {
	t.Run("strict equality (zero tolerance)", func(t *testing.T) {
		cfg := &CompareConfig{floatTolerance: 0.0}

		assert.True(t, cfg.CompareFloat64(1.0, 1.0))
		assert.True(t, cfg.CompareFloat64(0.0, 0.0))
		assert.False(t, cfg.CompareFloat64(1.0, 1.1))
		assert.False(t, cfg.CompareFloat64(1.0000001, 1.0))
	})

	t.Run("with tolerance", func(t *testing.T) {
		cfg := &CompareConfig{floatTolerance: 0.01}

		assert.True(t, cfg.CompareFloat64(1.0, 1.0))
		assert.True(t, cfg.CompareFloat64(1.0, 1.005))
		assert.True(t, cfg.CompareFloat64(1.005, 1.0))
		assert.False(t, cfg.CompareFloat64(1.0, 1.02))
		assert.False(t, cfg.CompareFloat64(1.02, 1.0))
	})

	t.Run("edge cases", func(t *testing.T) {
		cfg := &CompareConfig{floatTolerance: 0.1}

		// Exactly at tolerance boundary
		assert.False(t, cfg.CompareFloat64(1.0, 1.1)) // difference equals tolerance, should be false
		assert.True(t, cfg.CompareFloat64(1.0, 1.05)) // difference less than tolerance

		// Negative numbers
		assert.True(t, cfg.CompareFloat64(-1.0, -1.05))
		assert.False(t, cfg.CompareFloat64(-1.0, -1.2))
	})

	t.Run("default tolerance behavior", func(t *testing.T) {
		cfg := NewCompareConfig(nil) // default tolerance is 0

		assert.True(t, cfg.CompareFloat64(1.0, 1.0))
		assert.False(t, cfg.CompareFloat64(1.0, 1.0000001))
	})
}

func TestMultipleOptionsIntegration(t *testing.T) {
	opts := []CompareOption{
		FloatTolerance(0.001),
		IgnoreFields("timestamp", "id"),
		IgnorePaths("metadata.*", "debug.info"),
	}

	cfg := NewCompareConfig(opts)

	assert.InDelta(t, 0.001, cfg.floatTolerance, 1e-9)
	assert.True(t, cfg.ShouldIgnoreField("timestamp"))
	assert.True(t, cfg.ShouldIgnoreField("id"))
	assert.False(t, cfg.ShouldIgnoreField("name"))

	assert.True(t, cfg.ShouldIgnorePath("debug.info"))
	assert.True(t, cfg.ShouldIgnorePath("metadata.version"))
	assert.False(t, cfg.ShouldIgnorePath("content.body"))

	assert.True(t, cfg.CompareFloat64(1.0, 1.0005))
	assert.False(t, cfg.CompareFloat64(1.0, 1.002))
}
