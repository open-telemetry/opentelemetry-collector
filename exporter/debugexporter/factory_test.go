// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
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
	require.NotNil(t, te)
}

func TestDebugExporter_CustomLoggerOutputPaths(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_output_*.log")
	require.NoError(t, err)
	logFile := tmpFile.Name()
	t.Cleanup(func() { os.Remove(logFile) })

	cfg := &Config{
		Verbosity:          configtelemetry.LevelDetailed,
		SamplingInitial:    2,
		SamplingThereafter: 1,
		UseInternalLogger:  false,
		OutputPaths:        []string{logFile},
	}

	// Use a nop logger for the base logger, as it will not be used.
	baseLogger := zap.NewNop()
	exporterLogger, closer := createLogger(cfg, baseLogger)
	debug := newDebugExporter(exporterLogger, cfg.Verbosity)
	defer func() {
		err := closer()
		require.NoError(t, err, "closer should not return an error")
	}()

	// Actually emit a log line via the debug exporter.
	testMsg := "test debug exporter output path"
	// Use the debug exporter's pushLogs to trigger output.
	ld := generateTestLogsWithMessage(testMsg)
	_ = debug.pushLogs(context.Background(), ld)

	// Ensure the log file contains the test message.
	require.Eventually(t, func() bool {
		//nolint:gosec // G304: Potential file inclusion via variable
		data, err := os.ReadFile(logFile)
		return err == nil && strings.Contains(string(data), testMsg)
	}, 2*time.Second, 100*time.Millisecond, "expected log message not found in log file")
	require.NoError(t, tmpFile.Close())
}

func TestCreateCustomLoggerWithMultipleOutputs(t *testing.T) {
	// Create temporary files for testing
	tmpFile1, err := os.CreateTemp(t.TempDir(), "test_output1_*.log")
	require.NoError(t, err)
	logFile1 := tmpFile1.Name()
	t.Cleanup(func() { os.Remove(logFile1) })

	tmpFile2, err := os.CreateTemp(t.TempDir(), "test_output2_*.log")
	require.NoError(t, err)
	logFile2 := tmpFile2.Name()
	t.Cleanup(func() { os.Remove(logFile2) })

	cfg := &Config{
		Verbosity:          configtelemetry.LevelDetailed,
		SamplingInitial:    2,
		SamplingThereafter: 1,
		UseInternalLogger:  false,
		OutputPaths:        []string{logFile1, logFile2, "stdout"}, // Test multiple file outputs plus stdout
	}

	// Create custom logger with multiple outputs
	logger, closer := createCustomLogger(cfg)
	require.NotNil(t, logger)
	require.NotNil(t, closer)

	// Test that the logger works
	logger.Info("test message for multiple outputs")

	// Test that closer function works without errors
	err = closer()
	require.NoError(t, err, "closer function should not return an error")

	// Verify that calling closer multiple times doesn't cause issues
	err = closer()
	require.NoError(t, err, "calling closer multiple times should not return an error")

	require.NoError(t, tmpFile1.Close())
	require.NoError(t, tmpFile2.Close())
}

func TestCreateCustomLoggerFileCleanup(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_output_*.log")
	require.NoError(t, err)
	logFile := tmpFile.Name()
	t.Cleanup(func() { os.Remove(logFile) })

	cfg := &Config{
		Verbosity:          configtelemetry.LevelDetailed,
		SamplingInitial:    2,
		SamplingThereafter: 1,
		UseInternalLogger:  false,
		OutputPaths:        []string{logFile},
	}

	// Create custom logger
	logger, closer := createCustomLogger(cfg)
	require.NotNil(t, logger)
	require.NotNil(t, closer)

	// Write some logs
	logger.Info("test message before close")

	// Test proper cleanup by calling closer
	err = closer()
	require.NoError(t, err, "closer function should work correctly")

	require.NoError(t, tmpFile.Close())
}

func TestShutdownLoggerSyncError(t *testing.T) {
	// Define test cases for each exporter type
	testCases := []struct {
		name       string
		createFunc func(context.Context, exporter.Settings, component.Config) (component.Component, error)
	}{
		{
			name: "traces",
			createFunc: func(ctx context.Context, settings exporter.Settings, cfg component.Config) (component.Component, error) {
				return createTraces(ctx, settings, cfg)
			},
		},
		{
			name: "metrics",
			createFunc: func(ctx context.Context, settings exporter.Settings, cfg component.Config) (component.Component, error) {
				return createMetrics(ctx, settings, cfg)
			},
		},
		{
			name: "logs",
			createFunc: func(ctx context.Context, settings exporter.Settings, cfg component.Config) (component.Component, error) {
				return createLogs(ctx, settings, cfg)
			},
		},
		{
			name: "profiles",
			createFunc: func(ctx context.Context, settings exporter.Settings, cfg component.Config) (component.Component, error) {
				factory := NewFactory()
				return factory.(xexporter.Factory).CreateProfiles(ctx, settings, cfg)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original function and restore it after test
			originalFunc := createLoggerFunc
			defer func() { createLoggerFunc = originalFunc }()

			// Override the logger creation function to inject failing logger
			createLoggerFunc = func(_ *Config, _ *zap.Logger) (*zap.Logger, func() error) {
				return createFailingSyncLogger(t), func() error { return nil }
			}

			cfg := &Config{
				Verbosity:          configtelemetry.LevelDetailed,
				SamplingInitial:    2,
				SamplingThereafter: 1,
				UseInternalLogger:  false,
				OutputPaths:        []string{"stdout"},
			}

			// Create actual exporter with the failing logger
			settings := exportertest.NewNopSettings(componentType)
			exporter, err := tc.createFunc(context.Background(), settings, cfg)
			require.NoError(t, err, "Exporter creation should succeed")
			require.NotNil(t, exporter)

			// Test that shutdown returns the sync error from the specified line in factory.go
			err = exporter.Shutdown(context.Background())
			require.Error(t, err, "Expected sync error to be returned from %s shutdown", tc.name)
			assert.Contains(t, err.Error(), "intentional sync failure", "Expected specific sync error")
		})
	}
}

func TestCreateCustomLoggerZapOpenError(t *testing.T) {
	// Test the error path on line 181 where zap.Open fails
	testCases := []struct {
		name        string
		outputPaths []string
	}{
		{
			name:        "invalid_path_nonexistent_directory",
			outputPaths: []string{"/nonexistent/directory/that/cannot/be/created/file.log"},
		},
		{
			name:        "multiple_paths_with_invalid",
			outputPaths: []string{"stdout", "/nonexistent/directory/file.log"}, // First succeeds, second fails
		},
		{
			name:        "invalid_null_character_path",
			outputPaths: []string{"file\x00with\x00null.log"}, // Path with null characters
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Verbosity:          configtelemetry.LevelDetailed,
				SamplingInitial:    2,
				SamplingThereafter: 1,
				UseInternalLogger:  false,
				OutputPaths:        tc.outputPaths,
			}

			// Test that createCustomLogger panics when zap.Open fails on line 181
			require.Panics(t, func() {
				createCustomLogger(cfg)
			}, "Expected createCustomLogger to panic when zap.Open fails")

			// Verify the panic contains error information
			defer func() {
				if r := recover(); r != nil {
					assert.NotNil(t, r, "Expected panic to contain error information")
					t.Logf("Successfully caught panic from zap.Open error: %v", r)
				}
			}()

			// This should panic, but we need to call it in a way that doesn't stop the test
			func() {
				defer func() {
					if r := recover(); r != nil {
						assert.NotNil(t, r, "Expected panic to contain error information")
						t.Logf("Successfully caught panic from zap.Open error: %v", r)
					}
				}()
				createCustomLogger(cfg)
			}()
		})
	}
}

// failingSyncer is a custom WriteSyncer that always fails on Sync() for testing
type failingSyncer struct{}

func (f *failingSyncer) Write(p []byte) (n int, err error) {
	// Allow writes to succeed so we can test sync failure specifically
	return len(p), nil
}

func (f *failingSyncer) Sync() error {
	// Return a custom error that won't be filtered out by knownSyncError
	return errors.New("intentional sync failure for testing")
}

// createFailingSyncLogger creates a zap logger that will fail on Sync() for testing
func createFailingSyncLogger(_ *testing.T) *zap.Logger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.LevelKey = ""
	encoderConfig.TimeKey = ""

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	syncer := &failingSyncer{}

	core := zapcore.NewCore(encoder, syncer, zap.InfoLevel)
	logger := zap.New(core)

	return logger
}

// generateTestLogsWithMessage creates a logs.Logs with a log record containing the given message.
func generateTestLogsWithMessage(msg string) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr(msg)
	return ld
}
