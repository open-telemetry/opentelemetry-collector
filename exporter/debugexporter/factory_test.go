// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package debugexporter

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
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

// generateTestLogsWithMessage creates a logs.Logs with a log record containing the given message.
func generateTestLogsWithMessage(msg string) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr(msg)
	return ld
}
