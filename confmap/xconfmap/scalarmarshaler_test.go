// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func (tms textMarshalerStruct) MarshalText() ([]byte, error) {
	return tms.data, nil
}

func (tms *textMarshalerStruct) UnmarshalText(data []byte) error {
	tms.data = data
	return nil
}

type textMarshalerStruct struct {
	id   int
	data []byte
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
	_ confmap.Unmarshaler = (*wrapperType[any])(nil)
	_ ScalarMarshaler     = wrapperType[any]{}
	_ ScalarUnmarshaler   = (*wrapperType[any])(nil)
)

type wrapperType[T any] struct {
	inner T `mapstructure:"-"`
}

func (wt *wrapperType[T]) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(&wt.inner, WithScalarUnmarshaler()); err != nil {
		return err
	}

	return nil
}

func (wt wrapperType[T]) Marshal(conf *confmap.Conf) error {
	if err := conf.Marshal(wt.inner, WithScalarMarshaler()); err != nil {
		return fmt.Errorf("failed to marshal wrapperType value: %w", err)
	}

	return nil
}

func (wt wrapperType[T]) MarshalScalar(sv ScalarValue) error {
	return sv.Marshal(wt.inner, WithScalarMarshaler())
}

func (wt *wrapperType[T]) UnmarshalScalar(val ScalarValue) error {
	var v T
	if err := val.Unmarshal(&v); err != nil {
		return fmt.Errorf("could not unmarshal scalar: %w", err)
	}

	wt.inner = v
	return nil
}

type testConfig struct {
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

func (cfg *testConfig) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(cfg, WithScalarUnmarshaler()); err != nil {
		return err
	}

	return nil
}

func TestMarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	wantCfg := &testConfig{}
	require.NoError(t, cm.Unmarshal(wantCfg, WithScalarUnmarshaler()))
	require.NoError(t, cm.Marshal(wantCfg, WithScalarMarshaler()))

	conf := confmap.New()
	cfg := &testConfig{
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

	require.NoError(t, conf.Marshal(cfg, WithScalarMarshaler()))
	require.EqualValues(t, cm.ToStringMap(), conf.ToStringMap())
}

// failingScalarMarshaler always returns an error from MarshalScalar.
type failingScalarMarshaler struct{}

func (f failingScalarMarshaler) MarshalScalar(_ ScalarValue) error {
	return errors.New("marshal always fails")
}

// TestMarshalScalarErrorPropagation verifies that an error returned by
// MarshalScalar surfaces as an error from confmap.Marshal.
func TestMarshalScalarErrorPropagation(t *testing.T) {
	type cfgWithFailing struct {
		Val failingScalarMarshaler `mapstructure:"val"`
	}

	cfg := cfgWithFailing{Val: failingScalarMarshaler{}}
	conf := confmap.New()
	err := conf.Marshal(&cfg, WithScalarMarshaler())
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
	conf := confmap.New()
	require.NoError(t, conf.Marshal(cfg, WithScalarMarshaler()))

	m := conf.ToStringMap()
	require.Equal(t, 42, m["impl"], "implementing field should be encoded as scalar")
	require.Equal(t, 99, m["plain"], "plain field should be encoded as scalar")
	// NonImplWrapperType has no exported fields, so mapstructure encodes it as an empty map.
	_, ok := m["non_impl"]
	require.True(t, ok, "non-implementing field should still appear in output")
}
