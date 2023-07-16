// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configrotate"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

const (
	backupTimeFormat = "2006-01-02T15-04-05.000"
)

func TestTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		success bool
	}{
		{
			name: "Valid config",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
				},
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "127.0.0.1:3333",
				},
			},
			success: true,
		},
		{
			name: "Invalid config",
			cfg: &Config{
				Logs: LogsConfig{
					Level: zapcore.DebugLevel,
				},
				Metrics: MetricsConfig{
					Level:   configtelemetry.LevelBasic,
					Address: "127.0.0.1:3333",
				},
			},
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFactory()
			set := Settings{ZapOptions: []zap.Option{}}
			logger, err := f.CreateLogger(context.Background(), set, tt.cfg)
			if tt.success {
				assert.NoError(t, err)
				assert.NotNil(t, logger)
			} else {
				assert.Error(t, err)
				assert.Nil(t, logger)
			}
		})
	}
}

func TestSampledLogger(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "Default sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Encoding: "console",
				},
			},
		},
		{
			name: "Custom sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
					Sampling: &LogsSamplingConfig{
						Enabled:    true,
						Tick:       1 * time.Second,
						Initial:    100,
						Thereafter: 100,
					},
				},
			},
		},
		{
			name: "Disable sampling",
			cfg: &Config{
				Logs: LogsConfig{
					Level:    zapcore.DebugLevel,
					Encoding: "console",
					Sampling: &LogsSamplingConfig{
						Enabled: false,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFactory()
			ctx := context.Background()
			set := Settings{ZapOptions: []zap.Option{}}
			logger, err := f.CreateLogger(ctx, set, tt.cfg)
			assert.NoError(t, err)
			assert.NotNil(t, logger)
		})
	}
}

type loggerBuildTest struct {
	name string
	cfg  LogsConfig
	opts []zap.Option
}

func normalLoggerConfig() LogsConfig {
	return LogsConfig{
		Level:       zapcore.InfoLevel,
		Development: false,
		Encoding:    "console",
		Sampling: &LogsSamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Rotation: &configrotate.Config{
			Enabled:      true,
			MaxMegabytes: 123,
			MaxDays:      14,
			MaxBackups:   132,
			LocalTime:    false,
		},
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableCaller:     false,
		DisableStacktrace: false,
		InitialFields:     map[string]any(nil),
	}
}

func TestLoggerBuild(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	var tests []loggerBuildTest

	cfg := normalLoggerConfig()
	cfg.Rotation = nil
	tests = append(tests, loggerBuildTest{
		name: "non rotation config",
		cfg:  cfg,
	})

	cfg = normalLoggerConfig()
	tests = append(tests, loggerBuildTest{
		name: "logger config with normal rotation config",
		cfg:  cfg,
	})

	cfg = normalLoggerConfig()
	cfg.OutputPaths = append(cfg.OutputPaths, path.Join(tempDir, "test.log"))
	cfg.ErrorOutputPaths = append(cfg.ErrorOutputPaths, (&url.URL{
		Scheme: "file",
		Host:   "localhost",
		Path:   filepath.Join(tempDir, "test.err.log"),
	}).String())
	tests = append(tests, loggerBuildTest{
		name: "rotation config with file",
		cfg:  cfg,
	})

	cfg = normalLoggerConfig()
	cfg.Encoding = "json"
	tests = append(tests, loggerBuildTest{
		name: "rotation config with json encoding",
		cfg:  cfg,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newLogger(tt.cfg, tt.opts)
			assert.NoError(t, err)
		})
	}

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	if runtime.GOOS == "windows" {
		time.Sleep(1 * time.Second)
	}
}

func TestWrongLoggerBuild(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	var tests []loggerBuildTest

	cfg := normalLoggerConfig()
	cfg.OutputPaths = append(cfg.OutputPaths, (&url.URL{
		Scheme:   "file",
		Path:     path.Join(tempDir, "test.err.log"),
		RawQuery: "error=1",
	}).String())
	tests = append(tests, loggerBuildTest{
		name: "rotation config with dirty URL",
		cfg:  cfg,
	})

	cfg = normalLoggerConfig()
	// query should be empty
	cfg.OutputPaths = append(cfg.OutputPaths, "test.error.log?path=test")
	tests = append(tests, loggerBuildTest{
		name: "rotation config with dirty URL.",
		cfg:  cfg,
	})

	cfg = normalLoggerConfig()
	// non-existent schema
	cfg.OutputPaths = append(cfg.OutputPaths, "none:test")
	tests = append(tests, loggerBuildTest{
		name: "rotation config with non-existent schema as outputPath.",
		cfg:  cfg,
	})

	cfg = normalLoggerConfig()
	cfg.Encoding = "jsonabcdefg"
	tests = append(tests, loggerBuildTest{
		name: "rotation config with invalid encoding",
		cfg:  cfg,
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newLogger(tt.cfg, tt.opts)
			assert.Error(t, err)
		})
	}

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	if runtime.GOOS == "windows" {
		time.Sleep(1 * time.Second)
	}
}

func TestRotateFile(t *testing.T) {
	t.Skip("Skip this test because due to the goroutine leak of lumberjack")
	// zap doesn't close the output file even after sleeping for 5s.
	// This is not caused by lumberjack.
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}

	t.Parallel()
	tempDir := t.TempDir()
	tempFile := "test.log"
	tempFileNameLen := 4
	tempFileExtLen := 3
	cfg := normalLoggerConfig()
	// The default sampling will drop some duplicated logs
	cfg.Sampling = nil
	cfg.OutputPaths = []string{path.Join(tempDir, tempFile)}
	cfg.Rotation.MaxMegabytes = 1
	logger, err := newLogger(cfg, nil)
	assert.NoError(t, err)

	// write logs of about 1.1MB.
	tsBeforeLogging := time.Now()
	for i := 0; i < 1100; i++ {
		logger.Error(strings.Repeat("abcdefghij", 100))
	}
	err = logger.Sync()
	assert.NoError(t, err)
	files, err := os.ReadDir(tempDir)
	assert.NoError(t, err)
	assert.Len(t, files, 2)

	cntTempFile := 0
	for _, file := range files {
		fileName := file.Name()
		if fileName == tempFile {
			cntTempFile++
			continue
		}

		tsString := fileName[tempFileNameLen+1 : len(fileName)-tempFileExtLen-1]
		ts, err := time.Parse(backupTimeFormat, tsString)
		assert.NoError(t, err)
		assert.LessOrEqual(t, tsBeforeLogging, ts)
		assert.LessOrEqual(t, ts, time.Now())
	}
	assert.Equal(t, cntTempFile, 1)
}

func TestLargeOldFile(t *testing.T) {
	t.Skip("Skip this test because due to the goroutine leak of lumberjack")
	// zap doesn't close the output file even after sleeping for 5s.
	// This is not caused by lumberjack.
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}

	t.Parallel()
	tempDir := t.TempDir()
	tempFile := "test.log"
	tempPath := path.Join(tempDir, tempFile)

	// #nosec G304 -- tempPath is a trusted safe path
	f, err := os.Create(tempPath)
	assert.NoError(t, err)
	num, err := f.WriteString(strings.Repeat("abcdefghij", 10000000))
	assert.NoError(t, err)
	assert.Equal(t, num, 100000000)
	assert.NoError(t, f.Close())
	stat, err := os.Stat(tempPath)
	assert.NoError(t, err)

	cfg := normalLoggerConfig()
	cfg.OutputPaths = []string{tempPath}
	cfg.Rotation.MaxMegabytes = 1
	logger, err := newLogger(cfg, nil)
	assert.NoError(t, err)

	logger.Error("test log")
	err = logger.Sync()

	assert.NoError(t, err)
	files, err := os.ReadDir(tempDir)
	assert.NoError(t, err)
	assert.Len(t, files, 2)

	cntTempFile := 0
	for _, file := range files {
		fileName := file.Name()
		if fileName == tempFile {
			cntTempFile++
			continue
		}
		newStat, err := file.Info()
		assert.NoError(t, err)
		assert.Equal(t, stat.Size(), newStat.Size())
		assert.True(t, os.SameFile(stat, newStat))
	}
	assert.Equal(t, cntTempFile, 1)
}

func TestRelativePath(t *testing.T) {
	t.Skip("Skip this test because due to the goroutine leak of lumberjack")
	// zap doesn't close the output file even after sleeping for 5s.
	// This is not caused by lumberjack.
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}

	tempDir := t.TempDir()
	dir, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		assert.NoError(t, os.Chdir(dir))
	})
	tempFile := "test.log"

	cfg := normalLoggerConfig()
	cfg.OutputPaths = []string{tempFile}
	logger, err := newLogger(cfg, nil)
	assert.NoError(t, err)

	logger.Error("test log")
	assert.NoError(t, logger.Sync())

	stat, err := os.Stat(path.Join(tempDir, tempFile))
	assert.NoError(t, err)
	assert.False(t, stat.IsDir())
}

func TestInvalidUrls(t *testing.T) {
	tests := []struct {
		Name   string
		URL    string
		ErrMsg string
	}{
		{
			Name:   "Url includes user",
			URL:    "file://user:pass@localhost/path",
			ErrMsg: "user and password not allowed with file URLs",
		},
		{
			Name:   "Url includes fragments",
			URL:    "file://localhost/path#fragment",
			ErrMsg: "fragments not allowed with file URLs",
		},
		{
			Name:   "Url includes query",
			URL:    "file://localhost/path?k=v",
			ErrMsg: "query parameters not allowed with file URLs",
		},
		{
			Name:   "Url includes ports",
			URL:    "file://localhost:8080/path",
			ErrMsg: "ports not allowed with file URLs",
		},
		{
			Name:   "Url includes host ranther than localhost",
			URL:    "file://example.com/path",
			ErrMsg: "file URLs must leave host empty or use localhost",
		},
		{
			Name:   "Url cannot be parsed: invalid control character in Url",
			URL:    "file://localhost/\x00path",
			ErrMsg: "invalid control character in URL",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			cfg := normalLoggerConfig()
			cfg.ErrorOutputPaths = []string{tt.URL}
			_, err := newLogger(cfg, nil)
			assert.ErrorContains(t, err, tt.ErrMsg)
		})
	}

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	if runtime.GOOS == "windows" {
		time.Sleep(1 * time.Second)
	}
}
