// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"bytes"
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

var _ confmap.Unmarshaler = (*wrapperType[any])(nil)
var _ ScalarMarshaler = wrapperType[any]{}
var _ ScalarUnmarshaler = (*wrapperType[any])(nil)

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

func (wt wrapperType[T]) MarshalScalar(in *string) (any, error) {
	if in != nil {
		return *in, nil
	}

	return wt.inner, nil
}

func (wt wrapperType[T]) GetScalarValue() (any, error) {
	return wt.inner, nil
}

func (wt *wrapperType[T]) UnmarshalScalar(val any) error {
	v, ok := val.(T)

	if !ok {
		return fmt.Errorf("could not unmarshal scalar: val is %T, not %T", val, v)
	}

	wt.inner = v
	return nil
}

func (wt *wrapperType[T]) ScalarType() any {
	return wt.inner
}

type testConfig struct {
	// Handled by confmap, treated as string
	Tma         textMarshalerAlias                            `mapstructure:"tma"`
	Ntma        nonTextMarshalerAlias                         `mapstructure:"ntma"`
	Nonimplint  NonImplWrapperType[int]                       `mapstructure:"non_impl_int"`
	Nonimplstr  NonImplWrapperType[string]                    `mapstructure:"non_impl_str"`
	Nonimpltms  NonImplWrapperType[textMarshalerStruct]       `mapstructure:"non_impl_text_marshaler_struct"`
	Nonimplntms NonImplWrapperType[nonTextMarshalerStruct]    `mapstructure:"non_impl_non_text_marshaler_struct"`
	Implint     wrapperType[int]                              `mapstructure:"impl_int"`
	Implstr     wrapperType[string]                           `mapstructure:"impl_str"`
	Impltms     wrapperType[textMarshalerStruct]              `mapstructure:"impl_tms"`
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
		Impltms:     wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 0, data: []byte{80}}},
		Implntms:    wrapperType[nonTextMarshalerStruct]{inner: nonTextMarshalerStruct{id: 2, data: []byte{80}}},
		Recursive:   wrapperType[wrapperType[textMarshalerStruct]]{inner: wrapperType[textMarshalerStruct]{inner: textMarshalerStruct{id: 2, data: []byte{80}}}},
	}

	require.NoError(t, conf.Marshal(cfg, WithScalarMarshaler()))
	require.EqualValues(t, cm.ToStringMap(), conf.ToStringMap())
}
