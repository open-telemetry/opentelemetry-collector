// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package optional

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

func TestOptional(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		defaultCfg  Config
		expectedSub bool
		expectedFoo string
	}{
		{
			name: "no_default_no_config",
			defaultCfg: Config{
				Sub1: None[Sub](),
			},
			expectedSub: false,
		},
		{
			name: "no_default_with_config",
			config: map[string]any{
				"sub": map[string]any{
					"foo": "bar",
				},
			},
			defaultCfg: Config{
				Sub1: None[Sub](),
			},
			expectedSub: true,
			expectedFoo: "bar",
		},
		{
			name: "with_default_no_config",
			defaultCfg: Config{
				Sub1: WithDefault(func() Sub {
					return Sub{
						Foo: "foobar",
					}
				}),
			},
			expectedSub: false,
		},
		{
			name: "with_default_with_config",
			config: map[string]any{
				"sub": map[string]any{
					"foo": "bar",
				},
			},
			defaultCfg: Config{
				Sub1: WithDefault(func() Sub {
					return Sub{
						Foo: "foobar",
					}
				}),
			},
			expectedSub: true,
			expectedFoo: "bar", // input overrides default
		},
		{
			// this test fails, because "sub:" is considered null value by mapstructure
			// and no additional processing happens for it, including custom unmarshaler.
			// https://github.com/go-viper/mapstructure/blob/0382e5b7e3987443c91311b7fdb60b92c69a47bf/mapstructure.go#L445
			name: "with_default_with_config_no_foo",
			config: map[string]any{
				"sub": nil,
			},
			defaultCfg: Config{
				Sub1: WithDefault(func() Sub {
					return Sub{
						Foo: "foobar",
					}
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
				require.Equal(t, test.expectedFoo, cfg.Sub1.Value().Foo)
			}
		})
	}
}
