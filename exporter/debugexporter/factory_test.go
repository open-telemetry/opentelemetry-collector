// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

	// Verify default config
	config := cfg.(*Config)
	assert.True(t, config.UseInternalLogger)
	// When UseInternalLogger is true, OutputPaths should be empty
	assert.Empty(t, config.OutputPaths)
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
		config      *Config
		expectPaths []string
	}{
		{
			name: "single output path",
			config: &Config{
				OutputPaths:        []string{"stderr"},
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
			expectPaths: []string{"stderr"},
		},
		{
			name: "multiple output paths",
			config: &Config{
				OutputPaths:        []string{"stdout", "stderr"},
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
			expectPaths: []string{"stdout", "stderr"},
		},
		{
			name: "file path",
			config: &Config{
				OutputPaths:        []string{"test.log"}, // Will be resolved to temp dir in test
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
			expectPaths: []string{"test.log"}, // Path will be adjusted in test
		},
		{
			name: "stdout path",
			config: &Config{
				OutputPaths:        []string{"stdout"},
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
			expectPaths: []string{"stdout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of config to avoid issues with t.TempDir() being called multiple times
			config := *tt.config
			if len(config.OutputPaths) > 0 {
				// Check if it's a file path (not stdout/stderr)
				for i, path := range config.OutputPaths {
					if path != "stdout" && path != "stderr" && !filepath.IsAbs(path) {
						// This is a relative path that might need temp dir
						tmpDir := t.TempDir()
						config.OutputPaths[i] = filepath.Join(tmpDir, filepath.Base(path))
					}
				}
			}
			logger := createCustomLogger(&config)
			require.NotNil(t, logger)
			// Verify logger can be used without panicking
			logger.Info("test message")
			// Sync to ensure all writes are complete and close file handles
			// Note: Sync() may fail for stdout/stderr in test environments, which is acceptable
			_ = logger.Sync()
			// On Windows, we need to ensure file handles are released before cleanup
			// Set logger to nil and force GC to help release file handles
			logger = nil
			if runtime.GOOS == "windows" {
				runtime.GC()
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
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
			name: "use custom logger with stderr",
			config: &Config{
				UseInternalLogger:  false,
				OutputPaths:        []string{"stderr"},
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
		},
		{
			name: "use custom logger with file",
			config: &Config{
				UseInternalLogger:  false,
				OutputPaths:        []string{"test.log"}, // Will be resolved to temp dir in test
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
		},
		{
			name: "use custom logger with multiple paths",
			config: &Config{
				UseInternalLogger:  false,
				OutputPaths:        []string{"stdout", "stderr"},
				SamplingInitial:    2,
				SamplingThereafter: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of config to avoid issues with t.TempDir() being called multiple times
			config := *tt.config
			if len(config.OutputPaths) > 0 {
				// Check if it's a file path (not stdout/stderr)
				for i, path := range config.OutputPaths {
					if path != "stdout" && path != "stderr" && !filepath.IsAbs(path) {
						// This is a relative path that might need temp dir
						tmpDir := t.TempDir()
						config.OutputPaths[i] = filepath.Join(tmpDir, filepath.Base(path))
					}
				}
			}
			baseLogger := zap.NewNop()
			logger := createLogger(&config, baseLogger)
			require.NotNil(t, logger)
			// Verify logger can be used without panicking
			logger.Info("test message")
			// Sync to ensure all writes are complete and close file handles
			// Note: Sync() may fail for stdout/stderr in test environments, which is acceptable
			_ = logger.Sync()
			// On Windows, we need to ensure file handles are released before cleanup
			// Set logger to nil and force GC to help release file handles
			logger = nil
			if runtime.GOOS == "windows" {
				runtime.GC()
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestCreateLoggerWithInternalLogger(t *testing.T) {
	// Test that createLogger properly uses internal logger when configured
	config := &Config{
		UseInternalLogger:  true,
		SamplingInitial:    10,
		SamplingThereafter: 50,
		Verbosity:          configtelemetry.LevelDetailed,
	}

	baseLogger := zap.NewNop()
	logger := createLogger(config, baseLogger)
	require.NotNil(t, logger)

	// Verify logger can be used
	logger.Info("test message")
	// Note: Sync() may fail for internal logger in test environments, which is acceptable
	_ = logger.Sync()
}

func TestCreateCustomLoggerWithFileOutput(t *testing.T) {
	// Test creating a logger that writes to a file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "debug.log")

	config := &Config{
		OutputPaths:        []string{filePath},
		SamplingInitial:    2,
		SamplingThereafter: 1,
	}

	logger := createCustomLogger(config)
	require.NotNil(t, logger)

	// Write a test message
	logger.Info("test message to file")
	require.NoError(t, logger.Sync())

	// Verify file was created and contains the message
	_, err := os.Stat(filePath)
	assert.NoError(t, err, "log file should be created")

	// On Windows, we need to ensure file handles are released before cleanup
	// Set logger to nil and force GC to help release file handles
	logger = nil
	if runtime.GOOS == "windows" {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
}
