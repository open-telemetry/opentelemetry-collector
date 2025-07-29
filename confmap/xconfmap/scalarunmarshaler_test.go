// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

type nullableWrapperType[T any] struct {
	inner  T
	wasNil bool
}

func (n *nullableWrapperType[T]) UnmarshalScalar(val any) error {
	if val == nil {
		n.wasNil = true
		return nil
	}
	v, ok := val.(T)
	if !ok {
		return errors.New("nullableWrapperType: wrong type")
	}
	n.inner = v
	return nil
}

func (n *nullableWrapperType[T]) ScalarType() any {
	return n.inner
}

type failingScalarUnmarshaler struct{}

func (f *failingScalarUnmarshaler) UnmarshalScalar(_ any) error {
	return errors.New("always fails")
}

func (f *failingScalarUnmarshaler) ScalarType() any {
	return 0
}

func TestUnmarshalConfig(t *testing.T) {
	wantCfg := &testConfig{
		Tma:       textMarshalerAlias("test"),
		Ntma:      nonTextMarshalerAlias("test"),
		Implint:   wrapperType[int]{inner: 1},
		Implstr:   wrapperType[string]{inner: "test"},
		Impltms:   wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{81}}},
		Recursive: wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}}},
	}

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	cfg := &testConfig{}
	require.NoError(t, cm.Unmarshal(cfg, WithScalarUnmarshaler()))

	require.Equal(t, wantCfg, cfg)
}

// TestUnmarshalScalarNullInput verifies that the hook calls UnmarshalScalar(nil)
// when the source value is a nil map, which is how mapstructure represents a
// YAML null for a map-typed value.
func TestUnmarshalScalarNullInput(t *testing.T) {
	type cfgWithNullable struct {
		Val nullableWrapperType[int] `mapstructure:"val"`
	}

	// A nil map value triggers the `from.Kind() == reflect.Map && from.IsNil()` branch.
	cm := confmap.NewFromStringMap(map[string]any{"val": map[string]any(nil)})
	var cfg cfgWithNullable
	require.NoError(t, cm.Unmarshal(&cfg, WithScalarUnmarshaler()))
	assert.True(t, cfg.Val.wasNil, "expected UnmarshalScalar to be called with nil")
	assert.Equal(t, 0, cfg.Val.inner, "inner value should remain zero after nil")
}

// TestUnmarshalScalarDecodeError verifies that errors from internal.Decode are
// propagated when the source value cannot be decoded into ScalarType().
func TestUnmarshalScalarDecodeError(t *testing.T) {
	type cfgWithInt struct {
		Val wrapperType[int] `mapstructure:"val"`
	}

	// A slice cannot be decoded into an int; this exercises the internal.Decode error path.
	cm := confmap.NewFromStringMap(map[string]any{"val": []string{"a", "b"}})
	cfg := cfgWithInt{}
	err := cm.Unmarshal(&cfg, WithScalarUnmarshaler())
	require.Error(t, err)
}

// TestUnmarshalScalarErrorPropagation verifies that an error returned by
// UnmarshalScalar surfaces as an error from confmap.Unmarshal.
func TestUnmarshalScalarErrorPropagation(t *testing.T) {
	type cfgWithFailing struct {
		Val failingScalarUnmarshaler `mapstructure:"val"`
	}

	cm := confmap.NewFromStringMap(map[string]any{"val": 42})
	var cfg cfgWithFailing
	err := cm.Unmarshal(&cfg, WithScalarUnmarshaler())
	require.Error(t, err)
	require.ErrorContains(t, err, "always fails")
}

// TestNonImplementingTypesUnaffected verifies that fields whose types do not
// implement ScalarUnmarshaler are decoded normally by mapstructure, even when
// WithScalarUnmarshaler is active alongside implementing fields.
func TestNonImplementingTypesUnaffected(t *testing.T) {
	type mixedCfg struct {
		Impl    wrapperType[int]        `mapstructure:"impl"`
		NonImpl NonImplWrapperType[int] `mapstructure:"non_impl"`
		Plain   int                     `mapstructure:"plain"`
	}

	cm := confmap.NewFromStringMap(map[string]any{
		"impl":  10,
		"plain": 99,
	})
	var cfg mixedCfg
	require.NoError(t, cm.Unmarshal(&cfg, WithScalarUnmarshaler()))
	assert.Equal(t, 10, cfg.Impl.inner, "implementing field should be decoded via UnmarshalScalar")
	assert.Equal(t, 99, cfg.Plain, "plain field should be decoded normally")
	assert.Equal(t, 0, cfg.NonImpl.inner, "non-implementing field should remain zero")
}
