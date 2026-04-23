// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional

import (
	"encoding"
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

var _ encoding.TextUnmarshaler = (*textLevel)(nil)

type textLevel string

const (
	textLevelHigh textLevel = "high"
	textLevelLow  textLevel = "low"
	textLevelNone textLevel = "none"
)

func (l *textLevel) UnmarshalText(text []byte) error {
	switch textLevel(text) {
	case textLevelHigh, textLevelLow, textLevelNone:
		*l = textLevel(text)
		return nil
	default:
		return fmt.Errorf("unknown textLevel %q", string(text))
	}
}

var _ confmap.Unmarshaler = (*customUnmarshalerStruct)(nil)

type customUnmarshalerStruct struct {
	Val string
}

func (c *customUnmarshalerStruct) Unmarshal(conf *confmap.Conf) error {
	m := conf.ToStringMap()
	if v, ok := m["val"]; ok {
		c.Val = fmt.Sprintf("%v", v)
	}
	return nil
}

var subDefault = Sub{
	Foo: "foobar",
}

func ptr[T any](v T) *T {
	return &v
}

func TestDefaultPanics(t *testing.T) {
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
			require.NoError(t, conf.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
			require.Equal(t, test.expectedSub, cfg.Sub1.HasValue())
			if test.expectedSub {
				require.Equal(t, test.expectedFoo, cfg.Sub1.Get().Foo)
			}
		})
	}
}

func TestUnmarshalOptionalWithoutScalarUnmarshalerOption(t *testing.T) {
	config := map[string]any{
		"sub": map[string]any{"foo": "bar"},
	}
	defaultCfg := Config[Sub]{Sub1: Default(subDefault)}
	expectedFoo := "bar"

	cfg := defaultCfg
	conf := confmap.NewFromStringMap(config)
	require.NoError(t, conf.Unmarshal(&cfg))
	require.True(t, cfg.Sub1.HasValue())
	require.Equal(t, expectedFoo, cfg.Sub1.Get().Foo)
}

func TestUnmarshalErrorEnabledField(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"enabled": true,
	})
	// Use zero value to avoid panic on constructor.
	var none Optional[WithEnabled]
	require.Error(t, cm.Unmarshal(&none, xconfmap.WithScalarUnmarshaler()))
}

func TestUnmarshalConfigPointer(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"sub": map[string]any{
			"foo": "bar",
		},
	})

	var cfg Config[*Sub]
	err := cm.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler())
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

	err := cm.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler())
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

type MyListConfig struct {
	Val []string `mapstructure:"my_strs"`
}
type TestConfig struct {
	List Optional[MyListConfig] `mapstructure:"list"`
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

	err := cm.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler())
	require.NoError(t, err)

	assert.True(t, cfg.HasValue())
	assert.Equal(t, 42, cfg.Get().Val)
}

func TestListOptional(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"list": map[string]any{
			"my_strs": []string{"a", "b", "c"},
		},
	})

	cfg := TestConfig{
		List: Default(MyListConfig{Val: []string{"default"}}),
	}

	err := cm.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler())
	require.NoError(t, err)

	require.True(t, cfg.List.HasValue())
	require.Equal(t, []string{"a", "b", "c"}, cfg.List.Get().Val)
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
			optErr := conf.Unmarshal(&optCfg, xconfmap.WithScalarUnmarshaler())
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
			require.NoError(t, conf.Marshal(test.value, xconfmap.WithScalarMarshaler()))
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
			require.NoError(t, confPointer.Marshal(wrapPointer, xconfmap.WithScalarMarshaler()))

			wrapOptional := Wrap[Optional[Sub]]{Sub1: test.optional}
			confOptional := confmap.NewFromStringMap(nil)
			require.NoError(t, confOptional.Marshal(wrapOptional, xconfmap.WithScalarMarshaler()))

			assert.Equal(t, confPointer.ToStringMap(), confOptional.ToStringMap())
		})

		if test.skipOmitEmpty {
			continue
		}
		t.Run(fmt.Sprintf("%v vs %v (omitempty)", test.pointer, test.optional), func(t *testing.T) {
			wrapPointer := WrapOmitEmpty[*Sub]{Sub1: test.pointer}
			confPointer := confmap.NewFromStringMap(nil)
			require.NoError(t, confPointer.Marshal(wrapPointer, xconfmap.WithScalarMarshaler()))

			wrapOptional := WrapOmitEmpty[Optional[Sub]]{Sub1: test.optional}
			confOptional := confmap.NewFromStringMap(nil)
			require.NoError(t, confOptional.Marshal(wrapOptional, xconfmap.WithScalarMarshaler()))

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

func TestUnmarshalScalar(t *testing.T) {
	type IntConfig struct {
		Val Optional[int] `mapstructure:"val"`
	}
	type StrConfig struct {
		Val Optional[string] `mapstructure:"val"`
	}

	t.Run("int", func(t *testing.T) {
		tests := []struct {
			name         string
			config       map[string]any
			initial      IntConfig
			expectHasVal bool
			expectVal    int
		}{
			// Present scalar value overrides all initial flavors.
			{name: "none_with_value", config: map[string]any{"val": 42},
				initial: IntConfig{Val: None[int]()}, expectHasVal: true, expectVal: 42},
			{name: "default_with_value", config: map[string]any{"val": 42},
				initial: IntConfig{Val: Default(5)}, expectHasVal: true, expectVal: 42},
			{name: "some_with_value", config: map[string]any{"val": 42},
				initial: IntConfig{Val: Some(1)}, expectHasVal: true, expectVal: 42},
			// Absent key leaves the Optional unchanged.
			{name: "none_absent_key", config: map[string]any{},
				initial: IntConfig{Val: None[int]()}, expectHasVal: false},
			{name: "default_absent_key", config: map[string]any{},
				initial: IntConfig{Val: Default(5)}, expectHasVal: false}, // Default.HasValue() == false
			{name: "some_absent_key", config: map[string]any{},
				initial: IntConfig{Val: Some(3)}, expectHasVal: true, expectVal: 3},
			// Null (as a nil map) explicitly clears to None.
			{name: "none_null_map", config: map[string]any{"val": map[string]any(nil)},
				initial: IntConfig{Val: None[int]()}, expectHasVal: false},
			{name: "default_null_map", config: map[string]any{"val": map[string]any(nil)},
				initial: IntConfig{Val: Default(5)}, expectHasVal: false},
			{name: "some_null_map", config: map[string]any{"val": map[string]any(nil)},
				initial: IntConfig{Val: Some(3)}, expectHasVal: false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.initial
				conf := confmap.NewFromStringMap(tc.config)
				require.NoError(t, conf.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
				require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
				if tc.expectHasVal {
					require.Equal(t, tc.expectVal, *cfg.Val.Get())
				} else {
					require.Nil(t, cfg.Val.Get())
				}
			})
		}
	})

	t.Run("slice_of_ints", func(t *testing.T) {
		type SliceIntConfig struct {
			Val Optional[[]int] `mapstructure:"val"`
		}
		tests := []struct {
			name         string
			config       map[string]any
			initial      SliceIntConfig
			expectHasVal bool
			expectVal    []int
		}{
			{name: "none_with_value", config: map[string]any{"val": []any{1, 2, 3}},
				initial: SliceIntConfig{Val: None[[]int]()}, expectHasVal: true, expectVal: []int{1, 2, 3}},
			{name: "default_with_value", config: map[string]any{"val": []any{4, 5}},
				initial: SliceIntConfig{Val: Default([]int{1})}, expectHasVal: true, expectVal: []int{4, 5}},
			{name: "some_with_value", config: map[string]any{"val": []any{7}},
				initial: SliceIntConfig{Val: Some([]int{1, 2})}, expectHasVal: true, expectVal: []int{7}},
			{name: "none_absent_key", config: map[string]any{},
				initial: SliceIntConfig{Val: None[[]int]()}, expectHasVal: false},
			{name: "default_absent_key", config: map[string]any{},
				initial: SliceIntConfig{Val: Default([]int{1})}, expectHasVal: false},
			{name: "some_absent_key", config: map[string]any{},
				initial: SliceIntConfig{Val: Some([]int{1, 2})}, expectHasVal: true, expectVal: []int{1, 2}},
			{name: "none_null", config: map[string]any{"val": map[string]any(nil)},
				initial: SliceIntConfig{Val: None[[]int]()}, expectHasVal: false},
			{name: "default_null", config: map[string]any{"val": map[string]any(nil)},
				initial: SliceIntConfig{Val: Default([]int{1})}, expectHasVal: false},
			{name: "some_null", config: map[string]any{"val": map[string]any(nil)},
				initial: SliceIntConfig{Val: Some([]int{1, 2})}, expectHasVal: false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.initial
				conf := confmap.NewFromStringMap(tc.config)
				require.NoError(t, conf.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
				require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
				if tc.expectHasVal {
					require.Equal(t, tc.expectVal, *cfg.Val.Get())
				} else {
					require.Nil(t, cfg.Val.Get())
				}
			})
		}
	})

	t.Run("slice_of_optional_ints", func(t *testing.T) {
		type SliceOptIntConfig struct {
			Val []Optional[int] `mapstructure:"val"`
		}
		tests := []struct {
			name      string
			config    map[string]any
			initial   SliceOptIntConfig
			expectVal []Optional[int]
		}{
			{
				name:      "with_values",
				config:    map[string]any{"val": []any{1, 2, 3}},
				initial:   SliceOptIntConfig{},
				expectVal: []Optional[int]{Some(1), Some(2), Some(3)},
			},
			{
				name:      "absent_key",
				config:    map[string]any{},
				initial:   SliceOptIntConfig{},
				expectVal: nil,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.initial
				conf := confmap.NewFromStringMap(tc.config)
				require.NoError(t, conf.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
				require.Equal(t, tc.expectVal, cfg.Val)
			})
		}
	})

	t.Run("slice_of_text_unmarshalers", func(t *testing.T) {
		type SliceTextConfig struct {
			Val Optional[[]textLevel] `mapstructure:"val"`
		}
		tests := []struct {
			name         string
			config       map[string]any
			initial      SliceTextConfig
			expectHasVal bool
			expectVal    []textLevel
		}{
			{name: "none_with_values", config: map[string]any{"val": []any{"high", "low"}},
				initial: SliceTextConfig{Val: None[[]textLevel]()}, expectHasVal: true,
				expectVal: []textLevel{textLevelHigh, textLevelLow}},
			{name: "default_with_values", config: map[string]any{"val": []any{"none"}},
				initial: SliceTextConfig{Val: Default([]textLevel{textLevelHigh})}, expectHasVal: true,
				expectVal: []textLevel{textLevelNone}},
			{name: "some_with_values", config: map[string]any{"val": []any{"low", "high"}},
				initial: SliceTextConfig{Val: Some([]textLevel{textLevelNone})}, expectHasVal: true,
				expectVal: []textLevel{textLevelLow, textLevelHigh}},
			{name: "absent_key", config: map[string]any{},
				initial: SliceTextConfig{Val: None[[]textLevel]()}, expectHasVal: false},
			{name: "null", config: map[string]any{"val": map[string]any(nil)},
				initial: SliceTextConfig{Val: Some([]textLevel{textLevelHigh})}, expectHasVal: false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.initial
				conf := confmap.NewFromStringMap(tc.config)
				require.NoError(t, conf.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
				require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
				if tc.expectHasVal {
					require.Equal(t, tc.expectVal, *cfg.Val.Get())
				} else {
					require.Nil(t, cfg.Val.Get())
				}
			})
		}
	})

	t.Run("slice_of_confmap_unmarshalers", func(t *testing.T) {
		type SliceUnmarshalerConfig struct {
			Val Optional[[]customUnmarshalerStruct] `mapstructure:"val"`
		}
		tests := []struct {
			name         string
			config       map[string]any
			initial      SliceUnmarshalerConfig
			expectHasVal bool
			expectVal    []customUnmarshalerStruct
		}{
			{
				name: "none_with_values",
				config: map[string]any{"val": []any{
					map[string]any{"val": "a"},
					map[string]any{"val": "b"},
				}},
				initial:      SliceUnmarshalerConfig{Val: None[[]customUnmarshalerStruct]()},
				expectHasVal: true,
				expectVal:    []customUnmarshalerStruct{{Val: "a"}, {Val: "b"}},
			},
			{name: "absent_key", config: map[string]any{},
				initial: SliceUnmarshalerConfig{Val: None[[]customUnmarshalerStruct]()}, expectHasVal: false},
			{name: "null", config: map[string]any{"val": map[string]any(nil)},
				initial:      SliceUnmarshalerConfig{Val: Some([]customUnmarshalerStruct{{Val: "x"}})},
				expectHasVal: false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.initial
				conf := confmap.NewFromStringMap(tc.config)
				require.NoError(t, conf.Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
				require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
				if tc.expectHasVal {
					require.Equal(t, tc.expectVal, *cfg.Val.Get())
				} else {
					require.Nil(t, cfg.Val.Get())
				}
			})
		}
	})
}

func TestScalarMarshalingRoundTrip(t *testing.T) {
	type strWrapper struct {
		Val Optional[string] `mapstructure:"val"`
	}

	tests := []struct {
		name              string
		initial           Optional[string]
		expectAfterHasVal bool
		expectAfterVal    string
	}{
		{name: "none", initial: None[string](), expectAfterHasVal: false},
		// Default marshals as nil -> round-trips to None (not Default).
		{name: "default", initial: Default("hello"), expectAfterHasVal: false},
		{name: "some", initial: Some("hello"), expectAfterHasVal: true, expectAfterVal: "hello"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf := confmap.New()
			require.NoError(t, conf.Marshal(strWrapper{Val: tc.initial}, xconfmap.WithScalarMarshaler()))

			var result strWrapper
			require.NoError(t, conf.Unmarshal(&result, xconfmap.WithScalarUnmarshaler()))

			require.Equal(t, tc.expectAfterHasVal, result.Val.HasValue())
			if tc.expectAfterHasVal {
				require.Equal(t, tc.expectAfterVal, *result.Val.Get())
			} else {
				require.Nil(t, result.Val.Get())
			}
		})
	}
}

func TestScalarNoPanic(t *testing.T) {
	assert.NotPanics(t, func() { _ = Default(42) })
	assert.NotPanics(t, func() { _ = Default("hello") })
	assert.NotPanics(t, func() { _ = Default(3.14) })
	assert.NotPanics(t, func() { _ = None[int]() })
	assert.NotPanics(t, func() { _ = None[string]() })
	assert.NotPanics(t, func() { _ = None[float64]() })
	assert.NotPanics(t, func() { _ = Some(42) })
	assert.NotPanics(t, func() { _ = Some("hello") })
	assert.NotPanics(t, func() { _ = Some(3.14) })

	intDefault := Default(42)
	assert.False(t, intDefault.HasValue())
	assert.Nil(t, intDefault.Get())

	strDefault := Default("hello")
	assert.False(t, strDefault.HasValue())
	assert.Nil(t, strDefault.Get())

	floatDefault := Default(3.14)
	assert.False(t, floatDefault.HasValue())
	assert.Nil(t, floatDefault.Get())

	intNone := None[int]()
	assert.False(t, intNone.HasValue())
	assert.Nil(t, intNone.Get())

	intSome := Some(42)
	assert.True(t, intSome.HasValue())
	assert.Equal(t, ptr(42), intSome.Get())

	strSome := Some("hello")
	assert.True(t, strSome.HasValue())
	assert.Equal(t, ptr("hello"), strSome.Get())

	floatSome := Some(3.14)
	assert.True(t, floatSome.HasValue())
	assert.Equal(t, ptr(3.14), floatSome.Get())
}

// TestUnmarshalFromYAML verifies unmarshal behavior using real YAML input loaded
// from a testdata file. Each test case selects a sub-key from the file via
// (*confmap.Conf).Sub and confirms that YAML representations (null, empty map,
// absent key, explicit value) produce the expected Optional state.
func TestUnmarshalFromYAML(t *testing.T) {
	allConf, err := confmaptest.LoadConf("testdata/unmarshal.yaml")
	require.NoError(t, err)

	sub := func(t *testing.T, key string) *confmap.Conf {
		t.Helper()
		c, err := allConf.Sub(key)
		require.NoError(t, err)
		return c
	}

	t.Run("struct", func(t *testing.T) {
		tests := []struct {
			name         string
			key          string
			initial      Config[Sub]
			expectHasVal bool
			expectFoo    string
		}{
			// None: value present -> Some with provided value.
			{name: "none/with_value", key: "struct_with_value",
				initial: Config[Sub]{Sub1: None[Sub]()}, expectHasVal: true, expectFoo: "bar"},
			// None: null -> stays None.
			{name: "none/null", key: "struct_null",
				initial: Config[Sub]{Sub1: None[Sub]()}, expectHasVal: false},
			// None: empty map -> Some with zero value.
			{name: "none/empty", key: "struct_empty",
				initial: Config[Sub]{Sub1: None[Sub]()}, expectHasVal: true, expectFoo: ""},
			// None: absent key -> stays None.
			{name: "none/absent", key: "struct_absent",
				initial: Config[Sub]{Sub1: None[Sub]()}, expectHasVal: false},
			// Default: value present -> Some, input value overrides default.
			{name: "default/with_value", key: "struct_with_value",
				initial: Config[Sub]{Sub1: Default(subDefault)}, expectHasVal: true, expectFoo: "bar"},
			// Default: null -> Some, default value applies.
			{name: "default/null", key: "struct_null",
				initial: Config[Sub]{Sub1: Default(subDefault)}, expectHasVal: true, expectFoo: "foobar"},
			// Default: empty map -> Some, default value applies.
			{name: "default/empty", key: "struct_empty",
				initial: Config[Sub]{Sub1: Default(subDefault)}, expectHasVal: true, expectFoo: "foobar"},
			// Default: absent key -> stays None (HasValue false).
			{name: "default/absent", key: "struct_absent",
				initial: Config[Sub]{Sub1: Default(subDefault)}, expectHasVal: false},
			// Some: null -> keeps existing value.
			{name: "some/null", key: "struct_null",
				initial: Config[Sub]{Sub1: Some(Sub{Foo: "foobar"})}, expectHasVal: true, expectFoo: "foobar"},
			// Some: value present -> input value overrides existing.
			{name: "some/with_value", key: "struct_with_value",
				initial: Config[Sub]{Sub1: Some(Sub{Foo: "foobar"})}, expectHasVal: true, expectFoo: "bar"},
			// Some: absent key -> unchanged.
			{name: "some/absent", key: "struct_absent",
				initial: Config[Sub]{Sub1: Some(Sub{Foo: "foobar"})}, expectHasVal: true, expectFoo: "foobar"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := tc.initial
				require.NoError(t, sub(t, tc.key).Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
				require.Equal(t, tc.expectHasVal, cfg.Sub1.HasValue())
				if tc.expectHasVal {
					require.Equal(t, tc.expectFoo, cfg.Sub1.Get().Foo)
				}
			})
		}
	})

	t.Run("scalar", func(t *testing.T) {
		type IntConfig struct {
			Val Optional[int] `mapstructure:"val"`
		}
		type StrConfig struct {
			Val Optional[string] `mapstructure:"val"`
		}
		type SliceIntConfig struct {
			Val Optional[[]int] `mapstructure:"val"`
		}

		t.Run("int", func(t *testing.T) {
			tests := []struct {
				name         string
				key          string
				initial      IntConfig
				expectHasVal bool
				expectVal    int
			}{
				// Present value overrides all initial flavors.
				{name: "none/with_value", key: "int_with_value",
					initial: IntConfig{Val: None[int]()}, expectHasVal: true, expectVal: 42},
				{name: "default/with_value", key: "int_with_value",
					initial: IntConfig{Val: Default(5)}, expectHasVal: true, expectVal: 42},
				{name: "some/with_value", key: "int_with_value",
					initial: IntConfig{Val: Some(1)}, expectHasVal: true, expectVal: 42},
				// Null explicitly clears to None.
				{name: "some/null", key: "int_null",
					initial: IntConfig{Val: Some(3)}, expectHasVal: false},
				{name: "default/null", key: "int_null",
					initial: IntConfig{Val: Default(5)}, expectHasVal: false},
				// Absent key leaves Optional unchanged.
				{name: "none/absent", key: "int_absent",
					initial: IntConfig{Val: None[int]()}, expectHasVal: false},
				{name: "some/absent", key: "int_absent",
					initial: IntConfig{Val: Some(3)}, expectHasVal: true, expectVal: 3},
			}
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					cfg := tc.initial
					require.NoError(t, sub(t, tc.key).Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
					require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
					if tc.expectHasVal {
						require.Equal(t, tc.expectVal, *cfg.Val.Get())
					} else {
						require.Nil(t, cfg.Val.Get())
					}
				})
			}
		})

		t.Run("string", func(t *testing.T) {
			tests := []struct {
				name         string
				key          string
				initial      StrConfig
				expectHasVal bool
				expectVal    string
			}{
				{name: "none/with_value", key: "str_with_value",
					initial: StrConfig{Val: None[string]()}, expectHasVal: true, expectVal: "hello"},
				{name: "default/with_value", key: "str_with_value",
					initial: StrConfig{Val: Default("default")}, expectHasVal: true, expectVal: "hello"},
				{name: "some/with_value", key: "str_with_value",
					initial: StrConfig{Val: Some("old")}, expectHasVal: true, expectVal: "hello"},
				{name: "none/null", key: "int_null",
					initial: StrConfig{Val: None[string]()}, expectHasVal: false},
				{name: "some/null", key: "int_null",
					initial: StrConfig{Val: Some("old")}, expectHasVal: false},
			}
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					cfg := tc.initial
					require.NoError(t, sub(t, tc.key).Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
					require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
					if tc.expectHasVal {
						require.Equal(t, tc.expectVal, *cfg.Val.Get())
					} else {
						require.Nil(t, cfg.Val.Get())
					}
				})
			}
		})

		t.Run("slice_of_ints", func(t *testing.T) {
			tests := []struct {
				name         string
				key          string
				initial      SliceIntConfig
				expectHasVal bool
				expectVal    []int
			}{
				{name: "none/with_values", key: "slice_with_values",
					initial: SliceIntConfig{Val: None[[]int]()}, expectHasVal: true, expectVal: []int{1, 2, 3}},
				{name: "default/with_values", key: "slice_with_values",
					initial: SliceIntConfig{Val: Default([]int{9})}, expectHasVal: true, expectVal: []int{1, 2, 3}},
				{name: "some/null", key: "int_null",
					initial: SliceIntConfig{Val: Some([]int{1, 2})}, expectHasVal: false},
				{name: "none/absent", key: "int_absent",
					initial: SliceIntConfig{Val: None[[]int]()}, expectHasVal: false},
				{name: "some/absent", key: "int_absent",
					initial: SliceIntConfig{Val: Some([]int{1, 2})}, expectHasVal: true, expectVal: []int{1, 2}},
			}
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					cfg := tc.initial
					require.NoError(t, sub(t, tc.key).Unmarshal(&cfg, xconfmap.WithScalarUnmarshaler()))
					require.Equal(t, tc.expectHasVal, cfg.Val.HasValue())
					if tc.expectHasVal {
						require.Equal(t, tc.expectVal, *cfg.Val.Get())
					} else {
						require.Nil(t, cfg.Val.Get())
					}
				})
			}
		})
	})
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
