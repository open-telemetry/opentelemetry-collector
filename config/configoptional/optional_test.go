// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

type Config[T any] struct {
	Sub1 Optional[T] `mapstructure:"sub"`
}

type Sub struct {
	Foo string `mapstructure:"foo"`
}

func ptr[T any](v T) *T {
	return &v
}

func TestPanicsNestedOptional(t *testing.T) {
	assert.Panics(t, func() {
		_ = Some(Some(Sub{}))
	})

	assert.Panics(t, func() {
		_ = Some(Default(Sub{}))
	})

	assert.Panics(t, func() {
		_ = Default(1)
	})

	assert.Panics(t, func() {
		_ = Some(ptr(Some(Sub{})))
	})

	assert.Panics(t, func() {
		_ = Some(ptr(ptr(Some(Sub{}))))
	})

	assert.Panics(t, func() {
		_ = None[Optional[Sub]]()
	})

	assert.NotPanics(t, func() {
		_ = Some(1)
	})

	assert.NotPanics(t, func() {
		_ = None[int]()
	})
}

func TestNoneZeroVal(t *testing.T) {
	var none Optional[Sub]
	require.False(t, none.Enabled)
	require.Nil(t, none.Get())
}

func TestNone(t *testing.T) {
	none := None[Sub]()
	require.False(t, none.Enabled)
	require.Nil(t, none.Get())
}

func TestSome(t *testing.T) {
	some := Some(Sub{
		Foo: "foobar",
	})
	require.True(t, some.Enabled)
	require.NotEqual(t, 1, *some.Get())
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
			name: "none_with_config_enabled_true",
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
			expectedFoo: "bar", // input overrides default
		},
		{
			name: "none_with_config_enabled_false",
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
			name: "none_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
		{
			name: "default_no_config",
			defaultCfg: Config[Sub]{
				Sub1: Default(Sub{
					Foo: "foobar",
				}),
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
				Sub1: Default(Sub{
					Foo: "foobar",
				}),
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
				Sub1: Default(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
		{
			name: "default_with_config_no_foo_enabled_false",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": false,
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(Sub{
					Foo: "foobar",
				}),
			},
			expectedSub: false,
		},
		{
			name: "default_with_config_no_foo_enabled_true",
			config: map[string]any{
				"sub": map[string]any{
					"enabled": true,
				},
			},
			defaultCfg: Config[Sub]{
				Sub1: Default(Sub{
					Foo: "foobar",
				}),
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
			require.Equal(t, test.expectedSub, cfg.Sub1.Enabled)
			if test.expectedSub {
				require.Equal(t, test.expectedFoo, cfg.Sub1.Get().Foo)
			}
		})
	}
}

func TestUnmarshalEnabledType(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"sub": map[string]any{
			"enabled": "foo",
		},
	})
	var cfg Config[Sub]
	err := cm.Unmarshal(&cfg)
	require.Error(t, err)
	assert.ErrorContains(t, err, "configoptional: expected bool for 'enabled' field, got string")
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
	assert.True(t, cfg.Sub1.Enabled)
	assert.Equal(t, "bar", (*cfg.Sub1.Get()).Foo)
}

type MyIntConfig struct {
	Val int `mapstructure:"my_int"`
}
type MyConfig struct {
	Optional[MyIntConfig] `mapstructure:",squash"`
}

func TestSquashedOptional(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"my_int": 42,
	})

	cfg := MyConfig{
		Some(MyIntConfig{1}),
	}

	err := cm.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 42, cfg.Get().Val)
}
