// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	tests := []struct {
		filename             string
		cfg                  *Config
		expectedUnmarshalErr string
		expectedValidateErr  string
	}{
		{
			filename: "config_verbosity.yaml",
			cfg: &Config{
				Verbosity:          configtelemetry.LevelDetailed,
				SamplingInitial:    10,
				SamplingThereafter: 50,
				UseInternalLogger:  false,
				OutputPaths:        []string{"stdout"},
				QueueConfig:        configoptional.Default(exporterhelper.NewDefaultQueueConfig()),
			},
		},
		{
			filename: "config_output_paths.yaml",
			cfg: &Config{
				Verbosity:          configtelemetry.LevelBasic,
				SamplingInitial:    2,
				SamplingThereafter: 1,
				UseInternalLogger:  false,
				OutputPaths:        []string{"stderr"},
				QueueConfig:        configoptional.Default(exporterhelper.NewDefaultQueueConfig()),
			},
		},
		{
			filename:            "config_output_paths_empty.yaml",
			expectedValidateErr: "output_paths must not be empty when use_internal_logger is false",
		},
		{
			filename:             "config_verbosity_typo.yaml",
			expectedUnmarshalErr: "has invalid keys: verBosity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", tt.filename))
			require.NoError(t, err)
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			err = cm.Unmarshal(&cfg)
			if tt.expectedUnmarshalErr != "" {
				require.ErrorContains(t, err, tt.expectedUnmarshalErr)
				return
			}
			require.NoError(t, err)

			cfgCasted := cfg.(*Config)
			err = cfgCasted.Validate()
			if tt.expectedValidateErr != "" {
				require.ErrorContains(t, err, tt.expectedValidateErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.cfg, cfg)
		})
	}
}

func Test_UnmarshalMarshalled(t *testing.T) {
	for name, tc := range map[string]struct {
		inCfg          *Config
		expectedConfig *Config
		expectedErr    string
	}{
		"Base": {
			inCfg:          &Config{},
			expectedConfig: &Config{},
		},
		"VerbositySpecified": {
			inCfg: &Config{
				Verbosity: configtelemetry.LevelDetailed,
			},
			expectedConfig: &Config{
				Verbosity: configtelemetry.LevelDetailed,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			conf := confmap.New()
			err := conf.Marshal(tc.inCfg)
			require.NoError(t, err)

			raw := conf.ToStringMap()

			conf = confmap.NewFromStringMap(raw)

			outCfg := &Config{}

			err = conf.Unmarshal(outCfg)

			if tc.expectedErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedConfig, outCfg)
				return
			}
			require.Error(t, err)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr string
	}{
		{
			name: "verbosity none",
			cfg: &Config{
				Verbosity: configtelemetry.LevelNone,
			},
			expectedErr: "verbosity level \"None\" is not supported",
		},
		{
			name: "verbosity detailed",
			cfg: &Config{
				Verbosity:         configtelemetry.LevelDetailed,
				UseInternalLogger: true, // Default behavior
			},
		},
		{
			name: "empty output_paths when use_internal_logger is false",
			cfg: &Config{
				UseInternalLogger: false,
				OutputPaths:       []string{},
			},
			expectedErr: "output_paths must not be empty when use_internal_logger is false",
		},
		{
			name: "valid output_paths when use_internal_logger is false",
			cfg: &Config{
				UseInternalLogger: false,
				OutputPaths:       []string{"stdout"},
			},
		},
		{
			name: "nil output_paths when use_internal_logger is false (defaults to stdout)",
			cfg: &Config{
				UseInternalLogger: false,
				OutputPaths:       nil,
			},
		},
		{
			name: "output_paths set when use_internal_logger is true (not allowed)",
			cfg: &Config{
				UseInternalLogger: true,
				OutputPaths:       []string{"stderr"},
			},
			expectedErr: "output_paths is not supported when use_internal_logger is true",
		},
		{
			name: "nil output_paths when use_internal_logger is true (allowed)",
			cfg: &Config{
				UseInternalLogger: true,
				OutputPaths:       nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
