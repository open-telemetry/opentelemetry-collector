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
	"go.opentelemetry.io/collector/featuregate"
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

	assert.Panics(t, func() {
		opt := None[int]()
		_ = opt.GetOrInsertDefault()
	})

	assert.Panics(t, func() {
		var opt Optional[WithEnabled]
		_ = opt.GetOrInsertDefault()
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

	var zeroVal Sub
	ret := none.GetOrInsertDefault()
	require.True(t, none.HasValue())
	assert.Equal(t, &zeroVal, ret)
}

func TestNone(t *testing.T) {
	none := None[Sub]()
	require.False(t, none.HasValue())
	require.Nil(t, none.Get())

	var zeroVal Sub
	ret := none.GetOrInsertDefault()
	require.True(t, none.HasValue())
	assert.Equal(t, &zeroVal, ret)
}

func ExampleNone() {
	type Person struct {
		Name string
		Age  int
	}

	opt := None[Person]()

	// A None has no value.
	fmt.Println(opt.HasValue())
	fmt.Println(opt.Get())

	// GetOrInsertDefault places the zero value
	// and returns it, allowing you to modify it.
	opt.GetOrInsertDefault().Name = "John Doe"
	fmt.Println(opt.HasValue())
	fmt.Println(opt.Get())

	// Output:
	// false
	// <nil>
	// true
	// &{John Doe 0}
}

func TestSome(t *testing.T) {
	some := Some(Sub{
		Foo: "foobar",
	})
	require.True(t, some.HasValue())
	retGet := some.Get()
	assert.Equal(t, "foobar", retGet.Foo)

	retGetOrInsertDefault := some.GetOrInsertDefault()
	require.True(t, some.HasValue())
	assert.Equal(t, retGet, retGetOrInsertDefault)
}

func ExampleSome() {
	type Person struct {
		Name string
		Age  int
	}

	opt := Some(Person{
		Name: "John Doe",
		Age:  42,
	})

	// A Some has a value.
	fmt.Println(opt.HasValue())
	fmt.Println(opt.Get())

	// GetOrInsertDefault only returns a reference
	// to the inner value without modifying it.
	opt.GetOrInsertDefault().Name = "Jane Doe"
	fmt.Println(opt.HasValue())
	fmt.Println(opt.Get())

	// Output:
	// true
	// &{John Doe 42}
	// true
	// &{Jane Doe 42}
}

func TestDefault(t *testing.T) {
	defaultSub := Default(subDefault)
	require.False(t, defaultSub.HasValue())
	require.Nil(t, defaultSub.Get())

	ret := defaultSub.GetOrInsertDefault()
	require.True(t, defaultSub.HasValue())
	assert.Equal(t, &subDefault, ret)
}

func ExampleDefault() {
	type Person struct {
		Name string
		Age  int
	}

	opt := Default(Person{
		Name: "John Doe",
		Age:  42,
	})

	// A Default has no value.
	fmt.Println(opt.HasValue())
	fmt.Println(opt.Get())

	// GetOrInsertDefault places the default value
	// and returns it, allowing you to modify it.
	opt.GetOrInsertDefault().Age = 38
	fmt.Println(opt.HasValue())
	fmt.Println(opt.Get())

	// Output:
	// false
	// <nil>
	// true
	// &{John Doe 38}
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

func TestAddFieldEnabledFeatureGate(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		defaultCfg  Config[Sub]
		expectedSub bool
		expectedFoo string
	}{
		{
			name: "none_with_enabled_true",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": true,
					"foo":     "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: true,
			expectedFoo: "bar",
		},
		{
			name: "none_with_enabled_false",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
					"foo":     "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: false,
		},
		{
			name: "none_with_enabled_false_no_other_config",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: None[Sub](),
			},
			expectedSub: false,
		},
		{
			name: "default_with_enabled_true",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": true,
					"foo":     "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: true,
			expectedFoo: "bar",
		},
		{
			name: "default_with_enabled_false",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
					"foo":     "bar",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: false,
		},
		{
			name: "default_with_enabled_false_no_other_config",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(subDefault),
			},
			expectedSub: false,
		},
		{
			name: "some_with_enabled_true",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": true,
					"foo":     "baz",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Some(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: true,
			expectedFoo: "baz",
		},
		{
			name: "some_with_enabled_false",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
					"foo":     "baz",
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Some(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: false,
		},
		{
			name: "some_with_enabled_false_no_other_config",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Some(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: false,
		},
	}

	oldVal := addEnabledFieldFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(addEnabledFieldFeatureGateID, true))
	defer func() { require.NoError(t, featuregate.GlobalRegistry().Set(addEnabledFieldFeatureGateID, oldVal)) }()

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

func TestEnabledFalseResetsValue(t *testing.T) {
	oldVal := addEnabledFieldFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(addEnabledFieldFeatureGateID, true))
	defer func() { require.NoError(t, featuregate.GlobalRegistry().Set(addEnabledFieldFeatureGateID, oldVal)) }()

	cfg := Config[Sub]{Sub1: Some(Sub{Foo: "initial"})}
	require.True(t, cfg.Sub1.HasValue())

	cm := confmap.NewFromStringMap(map[string]any{
		"sub": map[string]any{"enabled": false, "foo": "ignored"},
	})
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Equal(t, None[Sub](), cfg.Sub1)
}

func TestUnmarshalErrorEnabledInvalidType(t *testing.T) {
	oldVal := addEnabledFieldFeatureGate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(addEnabledFieldFeatureGateID, true))
	defer func() { require.NoError(t, featuregate.GlobalRegistry().Set(addEnabledFieldFeatureGateID, oldVal)) }()

	cm := confmap.NewFromStringMap(map[string]any{
		"sub": map[string]any{
			"enabled": "something",
			"foo":     "bar",
		},
	})
	cfg := Config[Sub]{
		Sub1: None[Sub](),
	}
	err := cm.Unmarshal(&cfg)
	require.ErrorContains(t, err, "unexpected type string for 'enabled': got 'something' value expected 'true' or 'false'")
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
	require.NoError(t, xconfmap.Validate(hasNested{
		CouldBe: Default(invalid{}),
	}))
	require.Error(t, xconfmap.Validate(hasNested{
		CouldBe: Some(invalid{}),
	}))
}

type validatedConfig struct {
	Default Optional[optionalConfig] `mapstructure:"default"`
	Some    Optional[someConfig]     `mapstructure:"some"`
}

var _ xconfmap.Validator = (*optionalConfig)(nil)

type optionalConfig struct {
	StringVal string `mapstructure:"string_val"`
}

func (n optionalConfig) Validate() error {
	if n.StringVal == "invalid" {
		return errors.New("field `string_val` cannot be set to `invalid`")
	}

	return nil
}

type someConfig struct {
	Nested Optional[optionalConfig] `mapstructure:"nested"`
}

func newDefaultValidatedConfig() validatedConfig {
	return validatedConfig{
		Default: Default(optionalConfig{StringVal: "valid"}),
	}
}

func newInvalidDefaultConfig() validatedConfig {
	return validatedConfig{
		Default: Default(optionalConfig{StringVal: "invalid"}),
	}
}

func TestOptionalFileValidate(t *testing.T) {
	cases := []struct {
		name    string
		variant string
		cfg     func() validatedConfig
		err     error
	}{
		{
			name:    "valid default with just key set and no subfields",
			variant: "implicit",
			cfg:     newDefaultValidatedConfig,
		},
		{
			name:    "valid default with keys set in default",
			variant: "explicit",
			cfg:     newDefaultValidatedConfig,
		},
		{
			name:    "invalid config",
			variant: "invalid",
			cfg:     newDefaultValidatedConfig,
			err:     errors.New("default: field `string_val` cannot be set to `invalid`\nsome: nested: field `string_val` cannot be set to `invalid`"),
		},
		{
			name:    "invalid default throws an error",
			variant: "implicit",
			cfg:     newInvalidDefaultConfig,
			err:     errors.New("default: field `string_val` cannot be set to `invalid`"),
		},
		{
			name:    "invalid default does not throw an error when key is not set",
			variant: "no_default",
			cfg:     newInvalidDefaultConfig,
		},
		{
			name:    "invalid default invalid default does not throw an error when the value is overridden",
			variant: "explicit",
			cfg:     newInvalidDefaultConfig,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := confmaptest.LoadConf(fmt.Sprintf("testdata/validate_%s.yaml", tt.variant))
			require.NoError(t, err)

			cfg := tt.cfg()

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
