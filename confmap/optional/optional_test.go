package optional

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

type Config struct {
	Sub1 Optional[Sub] `mapstructure:"sub"`
}

type Sub struct {
	Foo string `mapstructure:"foo"`
}

func TestConfigValidateNoBackends(t *testing.T) {
	tests := []struct {
		name        string
		config      string
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
			config: `
					sub:
					  foo: bar
					`,
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
			config: `
					sub:
					  foo: bar
					`,
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
			config: `
					sub:
					`,
			defaultCfg: Config{
				Sub1: WithDefault(func() Sub {
					return Sub{
						Foo: "foobar",
					}
				}),
			},
			expectedSub: true,
			expectedFoo: "barbar", // default applies
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.defaultCfg
			conf := strToConf(t, test.config)
			require.NoError(t, conf.Unmarshal(&cfg))
			require.Equal(t, test.expectedSub, cfg.Sub1.HasValue())
			if test.expectedSub {
				require.Equal(t, test.expectedFoo, cfg.Sub1.Value().Foo)
			}
		})
	}
}

func strToConf(t *testing.T, config string) *confmap.Conf {
	config = stripWhitespacePrefix(config)
	d := t.TempDir()
	f := filepath.Join(d, "config.yaml")
	require.NoError(t, os.WriteFile(f, []byte(config), 0o644))
	cm, err := confmaptest.LoadConf(f)
	require.NoError(t, err)
	return cm
}

// stripWhitespacePrefix finds the first non-blank line,
// detects how much whitespace is in front of it, and removes
// that much whitespace from the beginning of all lines.
func stripWhitespacePrefix(s string) string {
	lines := strings.Split(s, "\n")
	var prefix string

	// Find the first non-blank line
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonSpace := strings.IndexFunc(line, func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		prefix = string([]rune(line)[:nonSpace])
		break
	}

	// Remove the prefix from all lines
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		result = append(result, strings.TrimPrefix(line, prefix))
	}

	return strings.Join(result, "\n")
}
