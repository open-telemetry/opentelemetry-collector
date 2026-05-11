// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type textMarshalerStruct struct {
	id   int
	data []byte
}

func (tms textMarshalerStruct) MarshalText() ([]byte, error) {
	return tms.data, nil
}

func (tms *textMarshalerStruct) UnmarshalText(data []byte) error {
	tms.data = data
	return nil
}

type nonTextMarshalerStruct struct {
	id   int
	data []byte
}

type textMarshalerAlias string

func (tma textMarshalerAlias) MarshalText() ([]byte, error) {
	return bytes.NewBufferString(string(tma)).Bytes(), nil
}

func (tma *textMarshalerAlias) UnmarshalText(data []byte) error {
	*tma = textMarshalerAlias(data)
	return nil
}

type nonTextMarshalerAlias string

type NonImplWrapperType[T any] struct {
	inner T `mapstructure:"-"`
}

var (
	_ Unmarshaler       = (*wrapperType[any])(nil)
	_ ScalarMarshaler   = wrapperType[any]{}
	_ ScalarUnmarshaler = (*wrapperType[any])(nil)
)

type wrapperType[T any] struct {
	inner T `mapstructure:"-"`
}

func (wt *wrapperType[T]) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(&wt.inner); err != nil {
		return err
	}

	return nil
}

func (wt wrapperType[T]) Marshal(conf *Conf) error {
	if err := conf.Marshal(wt.inner); err != nil {
		return fmt.Errorf("failed to marshal wrapperType value: %w", err)
	}

	return nil
}

func (wt wrapperType[T]) MarshalScalar(sv ScalarValue) error {
	return sv.Marshal(wt.inner)
}

func (wt *wrapperType[T]) UnmarshalScalar(val ScalarValue) error {
	var v T
	if err := val.Unmarshal(&v); err != nil {
		return fmt.Errorf("could not unmarshal scalar: %w", err)
	}

	wt.inner = v
	return nil
}

type testScalarConf struct {
	// Handled by confmap, treated as string
	Tma         textMarshalerAlias                            `mapstructure:"text_marshaler_alias"`
	Ntma        nonTextMarshalerAlias                         `mapstructure:"non_text_marshaler_alias"`
	Nonimplint  NonImplWrapperType[int]                       `mapstructure:"non_impl_int"`
	Nonimplstr  NonImplWrapperType[string]                    `mapstructure:"non_impl_str"`
	Nonimpltms  NonImplWrapperType[textMarshalerStruct]       `mapstructure:"non_impl_text_marshaler_struct"`
	Nonimplntms NonImplWrapperType[nonTextMarshalerStruct]    `mapstructure:"non_impl_non_text_marshaler_struct"`
	Implint     wrapperType[int]                              `mapstructure:"impl_int"`
	Implstr     wrapperType[string]                           `mapstructure:"impl_str"`
	Impltms     wrapperType[textMarshalerStruct]              `mapstructure:"impl_text_marshaler_struct"`
	Implntms    wrapperType[nonTextMarshalerStruct]           `mapstructure:"impl_non_text_marshaler_struct"`
	Recursive   wrapperType[wrapperType[textMarshalerStruct]] `mapstructure:"recursive"`
}

func (cfg *testScalarConf) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	return nil
}

func TestMarshalConfig(t *testing.T) {
	cm := NewFromStringMap(newConfFromFile(t, filepath.Join("testdata", "scalar.yaml")))
	wantCfg := &testScalarConf{}
	require.NoError(t, cm.Unmarshal(wantCfg))
	require.NoError(t, cm.Marshal(wantCfg))

	conf := New()
	cfg := &testScalarConf{
		Tma:         textMarshalerAlias("test"),
		Ntma:        nonTextMarshalerAlias("test"),
		Nonimplint:  NonImplWrapperType[int]{inner: 1},
		Nonimplstr:  NonImplWrapperType[string]{inner: "test"},
		Nonimpltms:  NonImplWrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{47}}},
		Nonimplntms: NonImplWrapperType[nonTextMarshalerStruct]{inner: nonTextMarshalerStruct{id: 2, data: []byte{48}}},
		Implint:     wrapperType[int]{inner: 1},
		Implstr:     wrapperType[string]{inner: "test"},
		Impltms:     wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{81}}},
		Implntms:    wrapperType[nonTextMarshalerStruct]{inner: nonTextMarshalerStruct{id: 2, data: []byte{80}}},
		Recursive:   wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 2, data: []byte{80}}}},
	}

	require.NoError(t, conf.Marshal(cfg))
	require.Equal(t, cm.ToStringMap(), conf.ToStringMap())
}

// failingScalarMarshaler always returns an error from MarshalScalar.
type failingScalarMarshaler struct{}

func (f failingScalarMarshaler) MarshalScalar(_ ScalarValue) error {
	return errors.New("marshal always fails")
}

// TestMarshalScalarErrorPropagation verifies that an error returned by
// MarshalScalar surfaces as an error from Marshal.
func TestMarshalScalarErrorPropagation(t *testing.T) {
	type cfgWithFailing struct {
		Val failingScalarMarshaler `mapstructure:"val"`
	}

	cfg := cfgWithFailing{Val: failingScalarMarshaler{}}
	conf := New()
	err := conf.Marshal(&cfg)
	require.Error(t, err)
	require.ErrorContains(t, err, "marshal always fails")
}

// TestMarshalNonImplementingTypesUnaffected verifies that fields whose types do
// not implement ScalarMarshaler are encoded normally by mapstructure (as empty
// maps for unexported-field structs), while implementing fields produce their
// scalar representation.
func TestMarshalNonImplementingTypesUnaffected(t *testing.T) {
	type mixedCfg struct {
		Impl    wrapperType[int]        `mapstructure:"impl"`
		NonImpl NonImplWrapperType[int] `mapstructure:"non_impl"`
		Plain   int                     `mapstructure:"plain"`
	}

	cfg := &mixedCfg{
		Impl:    wrapperType[int]{inner: 42},
		NonImpl: NonImplWrapperType[int]{inner: 7},
		Plain:   99,
	}
	conf := New()
	require.NoError(t, conf.Marshal(cfg))

	m := conf.ToStringMap()
	require.Equal(t, 42, m["impl"], "implementing field should be encoded as scalar")
	require.Equal(t, 99, m["plain"], "plain field should be encoded as scalar")
	// NonImplWrapperType has no exported fields, so mapstructure encodes it as an empty map.
	_, ok := m["non_impl"]
	require.True(t, ok, "non-implementing field should still appear in output")
}

type nullableWrapperType[T any] struct {
	inner  T
	wasNil bool
}

func (n *nullableWrapperType[T]) UnmarshalScalar(val ScalarValue) error {
	raw := val.GetRaw()
	if raw == nil || (reflect.ValueOf(raw).Kind() == reflect.Map && reflect.ValueOf(raw).IsNil()) {
		n.wasNil = true
		return nil
	}
	var v T
	if err := val.Unmarshal(&v); err != nil {
		return fmt.Errorf("nullableWrapperType: %w", err)
	}
	n.inner = v
	return nil
}

type failingScalarUnmarshaler struct{}

func (f *failingScalarUnmarshaler) UnmarshalScalar(_ ScalarValue) error {
	return errors.New("always fails")
}

func TestUnmarshalConfig(t *testing.T) {
	wantCfg := &testScalarConf{
		Tma:       textMarshalerAlias("test"),
		Ntma:      nonTextMarshalerAlias("test"),
		Implint:   wrapperType[int]{inner: 1},
		Implstr:   wrapperType[string]{inner: "test"},
		Impltms:   wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{81}}},
		Recursive: wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}}},
	}

	cm := NewFromStringMap(newConfFromFile(t, filepath.Join("testdata", "scalar.yaml")))
	cfg := &testScalarConf{}
	require.NoError(t, cm.Unmarshal(cfg))

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
	cm := NewFromStringMap(map[string]any{"val": map[string]any(nil)})
	var cfg cfgWithNullable
	require.NoError(t, cm.Unmarshal(&cfg))
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
	cm := NewFromStringMap(map[string]any{"val": []string{"a", "b"}})
	cfg := cfgWithInt{}
	err := cm.Unmarshal(&cfg)
	require.Error(t, err)
}

// TestUnmarshalScalarErrorPropagation verifies that an error returned by
// UnmarshalScalar surfaces as an error from Unmarshal.
func TestUnmarshalScalarErrorPropagation(t *testing.T) {
	type cfgWithFailing struct {
		Val failingScalarUnmarshaler `mapstructure:"val"`
	}

	cm := NewFromStringMap(map[string]any{"val": 42})
	var cfg cfgWithFailing
	err := cm.Unmarshal(&cfg)
	require.Error(t, err)
	require.ErrorContains(t, err, "always fails")
}

// TestNonImplementingTypesUnaffected verifies that fields whose types do not
// implement ScalarUnmarshaler are decoded normally by mapstructure, even when
// implementing fields are present in the same struct.
func TestNonImplementingTypesUnaffected(t *testing.T) {
	type mixedCfg struct {
		Impl    wrapperType[int]        `mapstructure:"impl"`
		NonImpl NonImplWrapperType[int] `mapstructure:"non_impl"`
		Plain   int                     `mapstructure:"plain"`
	}

	cm := NewFromStringMap(map[string]any{
		"impl":  10,
		"plain": 99,
	})
	var cfg mixedCfg
	require.NoError(t, cm.Unmarshal(&cfg))
	assert.Equal(t, 10, cfg.Impl.inner, "implementing field should be decoded via UnmarshalScalar")
	assert.Equal(t, 99, cfg.Plain, "plain field should be decoded normally")
	assert.Equal(t, 0, cfg.NonImpl.inner, "non-implementing field should remain zero")
}
