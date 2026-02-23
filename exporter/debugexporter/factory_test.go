// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, componenttest.CheckConfigStruct(cfg))

	config := cfg.(*Config)
	assert.True(t, config.UseInternalLogger)
	// OutputPaths is nil by default (only used when UseInternalLogger is false,
	// where it defaults to ["stdout"] in createCustomLogger if not set)
	assert.Nil(t, config.OutputPaths)
}

func TestCreateMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	me, err := factory.CreateMetrics(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NotNil(t, me)
}

func TestCreateTraces(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	te, err := factory.CreateTraces(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NotNil(t, te)
}

func TestCreateLogs(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	te, err := factory.CreateLogs(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NotNil(t, te)
}

func TestCreateFactoryProfiles(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	te, err := factory.(xexporter.Factory).CreateProfiles(context.Background(), exportertest.NewNopSettings(factory.Type()), cfg)
	require.NoError(t, err)
	assert.NotNil(t, te)
}

func TestCreateCustomLogger(t *testing.T) {
	tests := []struct {
		name        string
		outputPaths []string
	}{
		{
			name:        "stdout",
			outputPaths: []string{"stdout"},
		},
		{
			name:        "stderr",
			outputPaths: []string{"stderr"},
		},
		{
			name:        "multiple paths",
			outputPaths: []string{"stdout", "stderr"},
		},
		{
			name:        "nil defaults to stdout",
			outputPaths: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				OutputPaths:        tt.outputPaths,
				SamplingInitial:    2,
				SamplingThereafter: 1,
			}
			logger := createCustomLogger(config)
			require.NotNil(t, logger)
			logger.Info("test message")
			// Sync may return an error for stdout/stderr in test environments
			_ = logger.Sync()
		})
	}
}

func TestCreateCustomLoggerWithFileOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows due to file locking issues with t.TempDir() cleanup")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Clean(filepath.Join(tmpDir, "debug.log"))

	config := &Config{
		OutputPaths:        []string{filePath},
		SamplingInitial:    2,
		SamplingThereafter: 1,
	}

	logger := createCustomLogger(config)
	require.NotNil(t, logger)

	logger.Info("test message to file")
	require.NoError(t, logger.Sync())

	// Verify file was created and contains content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test message to file")
}

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "use internal logger",
			config: &Config{
				UseInternalLogger:  true,
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
		},
		{
			name: "use custom logger with stdout",
			config: &Config{
				UseInternalLogger:  false,
				OutputPaths:        []string{"stdout"},
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
		},
		{
			name: "use custom logger with nil output paths defaults to stdout",
			config: &Config{
				UseInternalLogger:  false,
				OutputPaths:        nil,
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseLogger := zap.NewNop()
			logger := createLogger(tt.config, baseLogger)
			require.NotNil(t, logger)
			logger.Info("test message")
			_ = logger.Sync()
		})
	}
}

func TestCreateLoggerWithInternalLogger(t *testing.T) {
	config := &Config{
		UseInternalLogger:  true,
		SamplingInitial:    10,
		SamplingThereafter: 50,
		Verbosity:          configtelemetry.LevelDetailed,
	}

	baseLogger := zap.NewNop()
	logger := createLogger(config, baseLogger)
	require.NotNil(t, logger)

	logger.Info("test message")
	_ = logger.Sync()
}
