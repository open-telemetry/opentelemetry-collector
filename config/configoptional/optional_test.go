// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
)

type Config struct {
	Sub1 Optional[Sub] `mapstructure:"sub"`
}

type Sub struct {
	Foo string `mapstructure:"foo"`
}

var subFactory = NewFactory(func() Sub {
	return Sub{
		Foo: "foobar",
	}
})

func TestOptional(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		defaultCfg  Config
		expectedSub bool
		expectedFoo string
	}{
		{
			name: "none_no_config",
			defaultCfg: Config{
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
			defaultCfg: Config{
				Sub1: None[Sub](),
			},
			expectedSub: true,
			expectedFoo: "bar", // input overrides default
		},
		{
			name: "none_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config{
				Sub1: Default(subFactory),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
		{
			name: "default_no_config",
			defaultCfg: Config{
				Sub1: Default(subFactory),
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
			defaultCfg: Config{
				Sub1: Default(subFactory),
			},
			expectedSub: true,
			expectedFoo: "bar", // input overrides default
		},
		{
			name: "default_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config{
				Sub1: Default(subFactory),
			},
			expectedSub: true,
			expectedFoo: "foobar", // default applies
		},
		{
			name: "some_no_config",
			defaultCfg: Config{
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
			defaultCfg: Config{
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
			defaultCfg: Config{
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
