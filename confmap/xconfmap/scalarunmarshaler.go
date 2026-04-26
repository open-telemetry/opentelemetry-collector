// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap // import "go.opentelemetry.io/collector/confmap/xconfmap"

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal"
)

func WithScalarUnmarshaler() confmap.UnmarshalOption {
	return internal.UnmarshalOptionFunc(func(uo *internal.UnmarshalOptions) {
		uo.AdditionalDecodeHookFuncs = append(uo.AdditionalDecodeHookFuncs, scalarunmarshalerHookFunc(uo))
	})
}

// ScalarUnmarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar
// value.
//
// This should be used for types like `Wrapper[T]` where T is a scalar type, and
// the wrapper type needs to implement custom logic for unmarshaling from a
// scalar value (e.g. `5` for `Wrapper[int]`) into the wrapper type (e.g.
// `Wrapper[int]{inner: 5}`).
type ScalarUnmarshaler interface {
	// UnmarshalScalar allows a type to unmarshal itself from a scalar value.
	UnmarshalScalar(ScalarValue) error
}

type ScalarValue interface {
	GetRaw() any

	Unmarshal(result any, opts ...internal.UnmarshalOption) error

	Marshal(result any, opts ...internal.MarshalOption) error

	// Seal the interface so it can't be implemented outside this package.
	_unexported()
}

// scalarunmarshalerHookFunc handles decoding for types implementing the
// ScalarUnmarshaler interface.
func scalarunmarshalerHookFunc(opts *internal.UnmarshalOptions) mapstructure.DecodeHookFuncValue {
	return safeWrapDecodeHookFunc(func(from, to reflect.Value) (any, error) {
		if !to.CanAddr() {
			return from.Interface(), nil
		}

		toPtr := to.Addr().Interface()

		unmarshaler, ok := toPtr.(ScalarUnmarshaler)
		if !ok {
			return from.Interface(), nil
		}

		if to.Addr().IsNil() {
			unmarshaler = reflect.New(to.Type()).Interface().(ScalarUnmarshaler)
		}

		// Non-nil maps shouldn't be handled by this hook as they indicate
		// struct-typed input.
		if from.Kind() == reflect.Map && !from.IsNil() {
			return from.Interface(), nil
		}

		sv := scalarValue{
			val:              from.Interface(),
			unmarshalOptions: opts,
		}

		if err := unmarshaler.UnmarshalScalar(&sv); err != nil {
			return nil, err
		}

		return unmarshaler, nil
	})
}

var _ ScalarValue = (*scalarValue)(nil)

type scalarValue struct {
	val any

	unmarshalOptions *internal.UnmarshalOptions
	marshalOptions   *internal.MarshalOptions
}

func (s *scalarValue) GetRaw() any {
	return s.val
}

func (s *scalarValue) Unmarshal(result any, opts ...internal.UnmarshalOption) error {
	settings := internal.ApplyUnmarshalOptions(s.unmarshalOptions, opts)
	return internal.Decode(s.val, result, *settings, false)
}

func (s *scalarValue) Marshal(value any, opts ...internal.MarshalOption) error {
	if value == nil {
		return nil
	}

	settings := internal.ApplyMarshalOptions(s.marshalOptions, opts)
	data, err := internal.Encode(value, *settings)
	if err != nil {
		return err
	}
	s.val = data

	return nil
}

func (s *scalarValue) _unexported() {}

// safeWrapDecodeHookFunc wraps a DecodeHookFuncValue to ensure fromVal is a valid `reflect.Value`
// object and therefore it is safe to call `reflect.Value` methods on fromVal.
//
// Use this only if the hook does not need to be called on untyped nil values.
// Typed nil values are safe to call and will be passed to the hook.
// See https://github.com/golang/go/issues/51649
func safeWrapDecodeHookFunc(
	f mapstructure.DecodeHookFuncValue,
) mapstructure.DecodeHookFuncValue {
	return func(fromVal, toVal reflect.Value) (any, error) {
		if !fromVal.IsValid() {
			return nil, nil
		}
		return f(fromVal, toVal)
	}
}
