// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config[T any] struct {
	Sub1 Optional[T] `mapstructure:"sub"`
}

type Sub struct {
	Foo string `mapstructure:"foo"`
}

type WithEnabled struct {
	Enabled bool `mapstructure:"enabled"`
}

type WithEnabledOmitEmpty struct {
	Enabled bool `mapstructure:"enabled,omitempty"`
}

type OmitEmpty struct {
	Foo int `mapstructure:",omitempty"`
}

type NoMapstructure struct {
	Foo string
}

var subDefault = Sub{
	Foo: "foobar",
}

func ptr[T any](v T) *T {
	return &v
}

func TestDefaultPanics(t *testing.T) {
	assert.Panics(t, func() {
		_ = Default(1)
	})

	assert.Panics(t, func() {
		_ = Default(ptr(1))
	})

	assert.Panics(t, func() {
		_ = Default(WithEnabled{})
	})

	assert.Panics(t, func() {
		_ = Default(WithEnabledOmitEmpty{})
	})

	assert.Panics(t, func() {
		_ = Some(WithEnabled{})
	})

	assert.Panics(t, func() {
		_ = None[WithEnabled]()
	})

	assert.NotPanics(t, func() {
		_ = Default(NoMapstructure{})
	})

	assert.NotPanics(t, func() {
		_ = Default(OmitEmpty{})
	})

	assert.NotPanics(t, func() {
		_ = Default(subDefault)
	})

	assert.NotPanics(t, func() {
		_ = Default(ptr(subDefault))
	})
}

func TestEqualityDefault(t *testing.T) {
	defaultOne := Default(subDefault)
	defaultTwo := Default(subDefault)
	assert.Equal(t, defaultOne, defaultTwo)
}

func TestNoneZeroVal(t *testing.T) {
	var none Optional[Sub]
	require.False(t, none.HasValue())
	require.Nil(t, none.Get())
}

func TestNone(t *testing.T) {
	none := None[Sub]()
	require.False(t, none.HasValue())
	require.Nil(t, none.Get())
}

func TestSome(t *testing.T) {
	some := Some(Sub{
		Foo: "foobar",
	})
	require.True(t, some.HasValue())
	assert.Equal(t, "foobar", some.Get().Foo)
}

func TestDefault(t *testing.T) {
	defaultSub := Default(&subDefault)
	require.False(t, defaultSub.HasValue())
	require.Nil(t, defaultSub.Get())
}

func TestUnmarshalOptional(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		defaultCfg  Config[Sub]
		expectedSub bool
		expectedFoo string
	}{
		{
			name: "none_no_config",
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: false,
		},
		{
			name: "none_with_config",
			config: map[string]any{
				"sub": map[string]any{
					"foo": "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: true,
			expectedFoo: "bar", // input overrides default
		},
		{
			// nil is treated as an empty map because of the hooks we use.
			name: "none_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: false,
		},
		{
			name: "none_with_config_no_foo_empty_map",
			config: map[string]any{
				"sub": map[string]any{},
			},
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: true,
			expectedFoo: "",
		},
		{
			name: "default_no_config",
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: false,
		},
		{
			name: "default_with_config",
			config: map[string]any{
				"sub": map[string]any{
					"foo": "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: true,
			expectedFoo: "bar", // input overrides default
		},
		{
			name: "default_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
		{
			name: "default_with_config_no_foo_empty_map",
			config: map[string]any{
				"sub": map[string]any{},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
		{
			name: "some_no_config",
			defaultCfg: Config[Sub]{
				Sub1: Some(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: true,
			expectedFoo: "foobar", // value is not modified
		},
		{
			name: "some_with_config",
			config: map[string]any{
				"sub": map[string]any{
					"foo": "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Some(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: true,
			expectedFoo: "bar", // input overrides previous value
		},
		{
			name: "some_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config[Sub]{
				Sub1: Some(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.defaultCfg
			conf := confmap.NewFromStringMap(test.config)
			require.NoError(t, conf.Unmarshal(&cfg))
			require.Equal(t, test.expectedSub, cfg.Sub1.HasValue())
			if test.expectedSub {
				require.Equal(t, test.expectedFoo, cfg.Sub1.Get().Foo)
			}
		})
	}
}

func TestUnmarshalErrorEnabledField(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"enabled": true,
	})
	// Use zero value to avoid panic on constructor.
	var none Optional[WithEnabled]
	require.Error(t, cm.Unmarshal(&none))
}

func TestUnmarshalConfigPointer(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"sub": map[string]any{
			"foo": "bar",
		},
	})

	var cfg Config[*Sub]
	err := cm.Unmarshal(&cfg)
	require.NoError(t, err)
	assert.True(t, cfg.Sub1.HasValue())
	assert.Equal(t, "bar", (*cfg.Sub1.Get()).Foo)
}

func TestUnmarshalErr(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"field": "value",
	})

	cfg := Config[Sub]{
		Sub1: Default(subDefault),
	}

	assert.False(t, cfg.Sub1.HasValue())

	err := cm.Unmarshal(&cfg)
	require.Error(t, err)
	require.ErrorContains(t, err, "has invalid keys: field")
	assert.False(t, cfg.Sub1.HasValue())
}

type MyIntConfig struct {
	Val int `mapstructure:"my_int"`
}
type MyConfig struct {
	Optional[MyIntConfig] `mapstructure:",squash"`
}

var myIntDefault = MyIntConfig{
	Val: 1,
}

func TestSquashedOptional(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"my_int": 42,
	})

	cfg := MyConfig{
		Default(myIntDefault),
	}

	err := cm.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.True(t, cfg.HasValue())
	assert.Equal(t, 42, cfg.Get().Val)
}

func confFromYAML(t *testing.T, yaml string) *confmap.Conf {
	t.Helper()
	cm, err := confmap.NewRetrievedFromYAML([]byte(yaml))
	require.NoError(t, err)
	conf, err := cm.AsConf()
	require.NoError(t, err)
	return conf
}

func TestComparePointerUnmarshal(t *testing.T) {
	tests := []struct {
		yaml string
	}{
		{yaml: ""},
		{yaml: "sub: "},
		{yaml: "sub: null"},
		{yaml: "sub: {}"},
		{yaml: "sub: {foo: bar}"},
	}
	for _, test := range tests {
		t.Run(test.yaml, func(t *testing.T) {
			var optCfg Config[Sub]
			conf := confFromYAML(t, test.yaml)
			optErr := conf.Unmarshal(&optCfg)
			require.NoError(t, optErr)

			var ptrCfg struct {
				Sub1 *Sub `mapstructure:"sub"`
			}
			ptrErr := conf.Unmarshal(&ptrCfg)
			require.NoError(t, ptrErr)

			assert.Equal(t, optCfg.Sub1.Get(), ptrCfg.Sub1)
		})
	}
}

func TestOptionalMarshal(t *testing.T) {
	tests := []struct {
		name     string
		value    Config[Sub]
		expected map[string]any
	}{
		{
			name:     "none (zero value)",
			value:    Config[Sub]{},
			expected: map[string]any{"sub": nil},
		},
		{
			name:     "none",
			value:    Config[Sub]{Sub1: None[Sub]()},
			expected: map[string]any{"sub": nil},
		},
		{
			name:     "default",
			value:    Config[Sub]{Sub1: Default(subDefault)},
			expected: map[string]any{"sub": nil},
		},
		{
			name: "some",
			value: Config[Sub]{Sub1: Some(Sub{
				Foo: "bar",
			})},
			expected: map[string]any{
				"sub": map[string]any{
					"foo": "bar",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conf := confmap.New()
			require.NoError(t, conf.Marshal(test.value))
			assert.Equal(t, test.expected, conf.ToStringMap())
		})
	}
}

func TestComparePointerMarshal(t *testing.T) {
	type Wrap[T any] struct {
		// Note: passes without requiring "squash".
		Sub1 T `mapstructure:"sub"`
	}

	type WrapOmitEmpty[T any] struct {
		// Note: passes without requiring "squash", except with Default-flavored Optional values.
		Sub1 T `mapstructure:"sub,omitempty"`
	}

	tests := []struct {
		pointer       *Sub
		optional      Optional[Sub]
		skipOmitEmpty bool
	}{
		{pointer: nil, optional: None[Sub]()},
		{pointer: nil, optional: Default(subDefault), skipOmitEmpty: true}, // does not work with omitempty
		{pointer: &Sub{Foo: "bar"}, optional: Some(Sub{Foo: "bar"})},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v vs %v", test.pointer, test.optional), func(t *testing.T) {
			wrapPointer := Wrap[*Sub]{Sub1: test.pointer}
			confPointer := confmap.NewFromStringMap(nil)
			require.NoError(t, confPointer.Marshal(wrapPointer))

			wrapOptional := Wrap[Optional[Sub]]{Sub1: test.optional}
			confOptional := confmap.NewFromStringMap(nil)
			require.NoError(t, confOptional.Marshal(wrapOptional))

			assert.Equal(t, confPointer.ToStringMap(), confOptional.ToStringMap())
		})

		if test.skipOmitEmpty {
			continue
		}
		t.Run(fmt.Sprintf("%v vs %v (omitempty)", test.pointer, test.optional), func(t *testing.T) {
			wrapPointer := WrapOmitEmpty[*Sub]{Sub1: test.pointer}
			confPointer := confmap.NewFromStringMap(nil)
			require.NoError(t, confPointer.Marshal(wrapPointer))

			wrapOptional := WrapOmitEmpty[Optional[Sub]]{Sub1: test.optional}
			confOptional := confmap.NewFromStringMap(nil)
			require.NoError(t, confOptional.Marshal(wrapOptional))

			assert.Equal(t, confPointer.ToStringMap(), confOptional.ToStringMap())
		})
	}
}

type invalid struct{}

func (invalid) Validate() error {
	return errors.New("invalid")
}

var _ xconfmap.Validator = invalid{}

type hasNested struct {
	CouldBe Optional[invalid]
}

func TestOptionalValidate(t *testing.T) {
	require.NoError(t, xconfmap.Validate(hasNested{
		CouldBe: None[invalid](),
	}))
	require.Error(t, xconfmap.Validate(hasNested{
		CouldBe: Default(invalid{}),
	}))
	require.Error(t, xconfmap.Validate(hasNested{
		CouldBe: Some(invalid{}),
	}))
}

var _ xconfmap.Validator = (*validatedConfig)(nil)

type validatedConfig struct {
	Default0         Optional[nestedConfig] `mapstructure:"default0"`
	Default1         Optional[nestedConfig] `mapstructure:"default1"`
	Some             Optional[someConfig]   `mapstructure:"some"`
	ActuallyRequired Optional[nestedConfig] `mapstructure:"actually_required"`
}

func (v validatedConfig) Validate() error {
	if !v.ActuallyRequired.HasValue() {
		return errors.New("field `actually_required` must be set")
	}

	return nil
}

var _ xconfmap.Validator = (*nestedConfig)(nil)

type nestedConfig struct {
	Required     string `mapstructure:"required"`
	NotRequired0 string `mapstructure:"not_required0"`
	NotRequired1 string `mapstructure:"not_required1"`
}

func (n nestedConfig) Validate() error {
	if n.Required == "" {
		return errors.New("field `required` must be set")
	}

	if n.NotRequired0 == "no" || n.NotRequired1 == "no" {
		return errors.New("non-required fields cannot be set to 'no'")
	}

	return nil
}

var _ xconfmap.Validator = (*someConfig)(nil)

type someConfig struct {
	Required string                 `mapstructure:"required"`
	Nested   Optional[nestedConfig] `mapstructure:"nested"`
}

func (s someConfig) Validate() error {
	if s.Required == "" {
		return errors.New("field `required` must be set")
	}

	return nil
}

func TestOptionalFileValidate(t *testing.T) {
	cases := []struct {
		name    string
		variant string
		cfg     validatedConfig
		err     error
	}{
		{
			name:    "valid config",
			variant: "valid",
			cfg: validatedConfig{
				Default0: Default(nestedConfig{Required: "valid value"}),
				Default1: Default(nestedConfig{Required: "valid value"}),
			},
		},
		{
			name:    "invalid config",
			variant: "missing_required",
			cfg: validatedConfig{
				Default0: Default(nestedConfig{Required: "valid value"}),
				Default1: Default(nestedConfig{Required: "valid value"}),
			},
			err: errors.New("field `actually_required` must be set\nsome: field `required` must be set"),
		},
		{
			name:    "invalid default",
			variant: "valid",
			cfg: validatedConfig{
				Default0: Default(nestedConfig{Required: "valid value", NotRequired0: "no", NotRequired1: "no"}),
				Default1: Default(nestedConfig{Required: "valid value", NotRequired0: "no", NotRequired1: "no"}),
			},
			err: errors.New("default0: non-required fields cannot be set to 'no'\ndefault1: non-required fields cannot be set to 'no'"),
		},
		{
			name:    "invalid and valid default",
			variant: "valid_with_extra",
			cfg: validatedConfig{
				Default0: Default(nestedConfig{Required: "valid value", NotRequired0: "no", NotRequired1: "no"}),
				Default1: Default(nestedConfig{Required: "valid value", NotRequired0: "no", NotRequired1: "no"}),
			},
			err: errors.New("default0: non-required fields cannot be set to 'no'"),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(fmt.Sprintf("testdata/validate_%s.yaml", tt.variant))
			require.NoError(t, err)

			cfg := tt.cfg

			err = conf.Unmarshal(&cfg)
			require.NoError(t, err)

			err = xconfmap.Validate(cfg)
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.err.Error())
			}
		})
	}
}

func TestDefaultValueNoUnmarshaling(t *testing.T) {
	cfg := validatedConfig{
		Some:     Some(someConfig{}),
		Default0: Default(nestedConfig{Required: "valid value", NotRequired0: "no", NotRequired1: "no"}),
		Default1: Default(nestedConfig{Required: "valid value", NotRequired0: "no", NotRequired1: "no"}),
	}

	err := xconfmap.Validate(cfg)
	require.EqualError(t, err, "field `actually_required` must be set\ndefault0: non-required fields cannot be set to 'no'\ndefault1: non-required fields cannot be set to 'no'\nsome: field `required` must be set")
}
