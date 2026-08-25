// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

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
	Tma          textMarshalerAlias                            `mapstructure:"text_marshaler_alias"`
	Ntma         nonTextMarshalerAlias                         `mapstructure:"non_text_marshaler_alias"`
	Nonimplint   NonImplWrapperType[int]                       `mapstructure:"non_impl_int"`
	Nonimplstr   NonImplWrapperType[string]                    `mapstructure:"non_impl_str"`
	Nonimpltms   NonImplWrapperType[textMarshalerStruct]       `mapstructure:"non_impl_text_marshaler_struct"`
	Nonimplntms  NonImplWrapperType[nonTextMarshalerStruct]    `mapstructure:"non_impl_non_text_marshaler_struct"`
	Implint      wrapperType[int]                              `mapstructure:"impl_int"`
	Implintptr   wrapperType[any]                              `mapstructure:"impl_int_ptr"`
	ImplintNull  wrapperType[int]                              `mapstructure:"impl_int_null"`
	ImplintUnset wrapperType[int]                              `mapstructure:"impl_int_unset"`
	Implstr      wrapperType[string]                           `mapstructure:"impl_str"`
	Impltms      wrapperType[textMarshalerStruct]              `mapstructure:"impl_text_marshaler_struct"`
	Implntms     wrapperType[nonTextMarshalerStruct]           `mapstructure:"impl_non_text_marshaler_struct"`
	Recursive    wrapperType[wrapperType[textMarshalerStruct]] `mapstructure:"recursive"`
}

func (cfg *testScalarConf) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(cfg); err != nil {
		return err
	}

	return nil
}

type failingScalarConfig struct{}

func (f failingScalarConfig) MarshalScalar(_ ScalarValue) error {
	return errors.New("always fails")
}

func (f *failingScalarConfig) UnmarshalScalar(_ ScalarValue) error {
	return errors.New("always fails")
}

type nonApplicableScalarConfig struct{}

func (f nonApplicableScalarConfig) MarshalScalar(_ ScalarValue) error {
	return ErrValueNotApplicable
}

func (f *nonApplicableScalarConfig) Marshal(_ *Conf) error {
	return nil
}

func (f *nonApplicableScalarConfig) UnmarshalScalar(_ ScalarValue) error {
	return ErrValueNotApplicable
}

func (f *nonApplicableScalarConfig) Unmarshal(_ *Conf) error {
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
		Implintptr:  wrapperType[any]{inner: nil},
		Implstr:     wrapperType[string]{inner: "test"},
		Impltms:     wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{81}}},
		Implntms:    wrapperType[nonTextMarshalerStruct]{inner: nonTextMarshalerStruct{id: 2, data: []byte{80}}},
		Recursive:   wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 2, data: []byte{80}}}},
	}

	require.NoError(t, conf.Marshal(cfg))
	require.Equal(t, cm.ToStringMap(), conf.ToStringMap())
}

func TestMarshalScalarErrorPropagation(t *testing.T) {
	type cfgWithFailing struct {
		Val failingScalarConfig `mapstructure:"val"`
	}

	cfg := cfgWithFailing{Val: failingScalarConfig{}}
	conf := New()
	err := conf.Marshal(&cfg)
	require.Error(t, err)
	require.ErrorContains(t, err, "always fails")
}

func TestMarshalNonApplicable(t *testing.T) {
	type cfgWithNonApplicable struct {
		Val nonApplicableScalarConfig `mapstructure:"val"`
	}

	cfg := cfgWithNonApplicable{Val: nonApplicableScalarConfig{}}
	conf := New()
	err := conf.Marshal(&cfg)
	require.NoError(t, err)
}

func TestUnmarshalConfig(t *testing.T) {
	wantCfg := &testScalarConf{
		Tma:          textMarshalerAlias("test"),
		Ntma:         nonTextMarshalerAlias("test"),
		Implint:      wrapperType[int]{inner: 1},
		ImplintNull:  wrapperType[int]{inner: 0},
		ImplintUnset: wrapperType[int]{inner: 3},
		Implstr:      wrapperType[string]{inner: "test"},
		Impltms:      wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{81}}},
		Recursive:    wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}}},
	}

	cm := NewFromStringMap(newConfFromFile(t, filepath.Join("testdata", "scalar.yaml")))
	cfg := &testScalarConf{
		ImplintNull:  wrapperType[int]{inner: 2},
		ImplintUnset: wrapperType[int]{inner: 3},
	}
	require.NoError(t, cm.Unmarshal(cfg))

	require.Equal(t, wantCfg, cfg)
}

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

func TestUnmarshalScalarErrorPropagation(t *testing.T) {
	type cfgWithFailing struct {
		Val failingScalarConfig `mapstructure:"val"`
	}

	cm := NewFromStringMap(map[string]any{"val": 42})
	var cfg cfgWithFailing
	err := cm.Unmarshal(&cfg)
	require.Error(t, err)
	require.ErrorContains(t, err, "always fails")
}

func TestUnmarshalNonApplicable(t *testing.T) {
	type cfgWithNonApplicable struct {
		Val nonApplicableScalarConfig `mapstructure:"val"`
	}

	cm := NewFromStringMap(map[string]any{"val": nil})
	var cfg cfgWithNonApplicable
	err := cm.Unmarshal(&cfg)
	require.NoError(t, err)
}
