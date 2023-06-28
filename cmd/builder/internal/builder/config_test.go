// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/cmd/builder/internal/config"
)

func TestParseModules(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{{
			GoMod: "github.com/org/repo v0.1.2",
		}},
	}

	// test
	err := cfg.ParseModules()
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "github.com/org/repo v0.1.2", cfg.Extensions[0].GoMod)
	assert.Equal(t, "github.com/org/repo", cfg.Extensions[0].Import)
	assert.Equal(t, "repo", cfg.Extensions[0].Name)
}

func TestRelativePath(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{{
			GoMod: "some-module",
			Path:  "./some-module",
		}},
	}

	// test
	err := cfg.ParseModules()
	assert.NoError(t, err)

	// verify
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(cfg.Extensions[0].Path, cwd))
}

func TestModuleFromCore(t *testing.T) {
	// prepare
	cfg := Config{
		Extensions: []Module{ // see issue-12
			{
				Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
				GoMod:  "go.opentelemetry.io/collector v0.0.0",
			},
			{
				Import: "go.opentelemetry.io/collector/receiver/otlpreceiver",
				GoMod:  "go.opentelemetry.io/collector v0.0.0",
			},
		},
	}

	// test
	err := cfg.ParseModules()
	assert.NoError(t, err)

	// verify
	assert.True(t, strings.HasPrefix(cfg.Extensions[0].Name, "otlpreceiver"))
}

func TestInvalidModule(t *testing.T) {
	type invalidModuleTest struct {
		cfg Config
		err error
	}
	// prepare
	configurations := []invalidModuleTest{
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Extensions: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrInvalidGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Receivers: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrInvalidGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Exporters: []Module{{
					Import: "invali",
				}},
			},
			err: ErrInvalidGoMod,
		},
		{
			cfg: Config{
				Logger: zap.NewNop(),
				Processors: []Module{{
					Import: "invalid",
				}},
			},
			err: ErrInvalidGoMod,
		},
	}

	for _, test := range configurations {
		assert.True(t, errors.Is(test.cfg.Validate(), test.err))
	}
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	require.NoError(t, cfg.ParseModules())
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
	require.NoError(t, cfg.Validate())
	assert.False(t, cfg.Distribution.DebugCompilation)
}

func TestNewBuiltinConfig(t *testing.T) {
	k := koanf.New(".")

	require.NoError(t, k.Load(config.DefaultProvider(), yaml.Parser()))

	cfg := Config{Logger: zaptest.NewLogger(t)}

	require.NoError(t, k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{Tag: "mapstructure"}))
	assert.NoError(t, cfg.ParseModules())
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())

	// Unlike the config initialized in NewDefaultConfig(), we expect
	// the builtin default to be practically useful, so there must be
	// a set of modules present.
	assert.NotZero(t, len(cfg.Receivers))
	assert.NotZero(t, len(cfg.Exporters))
	assert.NotZero(t, len(cfg.Extensions))
	assert.NotZero(t, len(cfg.Processors))
}

func TestSkipGoValidation(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			Go: "invalid/go/binary/path",
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
}

func TestSkipGoInitialization(t *testing.T) {
	cfg := Config{
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.NoError(t, cfg.SetGoPath())
	assert.Zero(t, cfg.Distribution.Go)
}

func TestDebugOptionSetConfig(t *testing.T) {
	cfg := Config{
		Distribution: Distribution{
			DebugCompilation: true,
		},
		SkipCompilation: true,
		SkipGetModules:  true,
	}
	assert.NoError(t, cfg.Validate())
	assert.True(t, cfg.Distribution.DebugCompilation)
}
