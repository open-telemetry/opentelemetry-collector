// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
)

type shutdownable interface {
	Shutdown(context.Context) error
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name         string
		wantCoreType any
		wantErr      error
		cfg          Config
	}{
		{
			name:         "no log config",
			cfg:          Config{},
			wantErr:      errors.New("no encoder name specified"),
			wantCoreType: nil,
		},
		{
			name: "log config with no processors",
			cfg: Config{
				Logs: LogsConfig{
					Level:             zapcore.DebugLevel,
					Development:       true,
					Encoding:          "console",
					DisableCaller:     true,
					DisableStacktrace: true,
					InitialFields:     map[string]any{"fieldKey": "filed-value"},
				},
			},
			wantCoreType: "*componentattribute.consoleCoreWithAttributes",
		},
		{
			name: "log config with processors",
			cfg: Config{
				Logs: LogsConfig{
					Level:             zapcore.DebugLevel,
					Development:       true,
					Encoding:          "console",
					DisableCaller:     true,
					DisableStacktrace: true,
					InitialFields:     map[string]any{"fieldKey": "filed-value"},
					Processors: []config.LogRecordProcessor{
						{
							Batch: &config.BatchLogRecordProcessor{
								Exporter: config.LogRecordExporter{
									Console: config.Console{},
								},
							},
						},
					},
				},
			},
			wantCoreType: "*componentattribute.otelTeeCoreWithAttributes",
		},
		{
			name: "log config with sampling",
			cfg: Config{
				Logs: LogsConfig{
					Level:       zapcore.InfoLevel,
					Development: false,
					Encoding:    "console",
					Sampling: &LogsSamplingConfig{
						Enabled:    true,
						Tick:       10 * time.Second,
						Initial:    10,
						Thereafter: 100,
					},
					OutputPaths:       []string{"stderr"},
					ErrorOutputPaths:  []string{"stderr"},
					DisableCaller:     false,
					DisableStacktrace: false,
					InitialFields:     map[string]any(nil),
				},
			},
			wantCoreType: "*componentattribute.consoleCoreWithAttributes",
		},
		{
			name: "log config with rotation",
			cfg: Config{
				Logs: LogsConfig{
					Level:             zapcore.InfoLevel,
					Development:       false,
					Encoding:          "json",
					OutputPaths:       []string{filepath.Join(t.TempDir(), "test-rotate.log")},
					ErrorOutputPaths:  []string{"stderr"},
					DisableCaller:     false,
					DisableStacktrace: false,
					Rotation: &LogsRotationConfig{
						Enabled:      true,
						MaxMegabytes: 1,
						MaxBackups:   2,
						MaxAge:       1,
						Compress:     false,
					},
				},
			},
			wantCoreType: "*componentattribute.consoleCoreWithAttributes",
		},
	}
	for _, tt := range tests {
		testCoreType := func(t *testing.T, wantCoreType any) {
			sdk, _ := config.NewSDK(config.WithOpenTelemetryConfiguration(config.OpenTelemetryConfiguration{LoggerProvider: &config.LoggerProvider{
				Processors: tt.cfg.Logs.Processors,
			}}))

			l, lp, err := newLogger(Settings{SDK: &sdk}, tt.cfg)
			if tt.wantErr != nil {
				require.ErrorContains(t, err, tt.wantErr.Error())
				require.Nil(t, wantCoreType)
			} else {
				require.NoError(t, err)
				gotType := reflect.TypeOf(l.Core()).String()
				require.Equal(t, wantCoreType, gotType)
				if prov, ok := lp.(shutdownable); ok {
					require.NoError(t, prov.Shutdown(context.Background()))
				}
			}
		}

		testCoreType(t, tt.wantCoreType)
	}
}

func TestNewLoggerWithResource(t *testing.T) {
	observerCore, observedLogs := observer.New(zap.InfoLevel)

	set := Settings{
		ZapOptions: []zap.Option{
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				// Combine original core and observer core to capture everything
				return zapcore.NewTee(core, observerCore)
			}),
		},
	}

	cfg := Config{
		Logs: LogsConfig{
			Level:    zapcore.InfoLevel,
			Encoding: "json",
		},
		Resource: map[string]*string{
			"myfield": ptr("myvalue"),
		},
	}

	mylogger, _, _ := newLogger(set, cfg)

	mylogger.Info("Test log message")
	require.Len(t, observedLogs.All(), 1)

	entry := observedLogs.All()[0]
	assert.Equal(t, "resource", entry.Context[0].Key)
	dict := entry.Context[0].Interface.(zapcore.ObjectMarshaler)
	enc := zapcore.NewMapObjectEncoder()
	require.NoError(t, dict.MarshalLogObject(enc))
	require.Equal(t, "myvalue", enc.Fields["myfield"])
}

func TestNewLoggerWithRotateEnabled(t *testing.T) {
	observerCore, observedLogs := observer.New(zap.InfoLevel)

	set := Settings{
		ZapOptions: []zap.Option{
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				// Combine original core and observer core to capture everything
				return zapcore.NewTee(core, observerCore)
			}),
		},
	}

	cfg := Config{
		Logs: LogsConfig{
			Level:       zapcore.InfoLevel,
			Encoding:    "json",
			OutputPaths: []string{filepath.Join(t.TempDir(), "test-rotate.log")},
			Rotation: &LogsRotationConfig{
				Enabled:      true,
				MaxMegabytes: 1,
				Compress:     false,
			},
		},
	}

	mylogger, _, err := newLogger(set, cfg)
	require.NoError(t, err)

	// Ensure proper cleanup of lumberjack logger
	if ljLogger != nil {
		defer ljLogger.Close()
	}

	mylogger.Info("Test log message")
	require.Len(t, observedLogs.All(), 1)

	require.NotNil(t, ljLogger)
}

func TestNewLogger_RotateFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	tempFile := "test.log"
	logFileFullPath := filepath.Join(tempDir, tempFile)

	observerCore, _ := observer.New(zap.InfoLevel)
	set := Settings{
		ZapOptions: []zap.Option{
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				// Combine original core and observer core to capture everything
				return zapcore.NewTee(core, observerCore)
			}),
		},
	}

	cfg := Config{
		Logs: LogsConfig{
			Level:       zapcore.InfoLevel,
			Encoding:    "json",
			OutputPaths: []string{logFileFullPath},
			Rotation: &LogsRotationConfig{
				Enabled:      true,
				MaxMegabytes: 1, // Rotate after ~1MB
				Compress:     false,
			},
		},
	}

	logger, _, err := newLogger(set, cfg)
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Ensure proper cleanup of lumberjack logger
	if ljLogger != nil {
		defer ljLogger.Close()
	}
	// Write ~1.2MB log data
	line := strings.Repeat("abcdefghijmnewbigfilewritingdatatobigdatafile", 200) // ~10KB
	for i := 0; i < 200; i++ {
		logger.Info(fmt.Sprintf("Line %d: %s", i, line))
	}

	err = logger.Sync()
	require.NoError(t, err)
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	// We expect two files: test.log and test.log.<timestamp>
	require.Len(t, files, 2)

	defer ljLogger.Close()
	cntTempFile := 0
	for _, file := range files {
		fileName := file.Name()
		if fileName == tempFile {
			cntTempFile++
			continue
		}
		// Can't validate timestamp without known format, skip this part
	}
	assert.Equal(t, 1, cntTempFile)
}

func TestGetFirstFileOutputPath(t *testing.T) {
	tests := []struct {
		name        string
		logsCfg     LogsConfig
		expectedLog string
	}{
		{
			name:        "Empty OutputPaths",
			logsCfg:     LogsConfig{OutputPaths: []string{}},
			expectedLog: "",
		},
		{
			name:        "Only stdout",
			logsCfg:     LogsConfig{OutputPaths: []string{"stdout"}},
			expectedLog: "",
		},
		{
			name:        "Only stderr",
			logsCfg:     LogsConfig{OutputPaths: []string{"stderr"}},
			expectedLog: "",
		},
		{
			name:        "Only console",
			logsCfg:     LogsConfig{OutputPaths: []string{"console"}},
			expectedLog: "",
		},
		{
			name:        "Keywords only",
			logsCfg:     LogsConfig{OutputPaths: []string{"stdout", "stderr", "console"}},
			expectedLog: "",
		},
		{
			name:        "Single file path",
			logsCfg:     LogsConfig{OutputPaths: []string{"/var/log/test.log"}},
			expectedLog: "/var/log/test.log",
		},
		{
			name:        "File path first, then keywords",
			logsCfg:     LogsConfig{OutputPaths: []string{"/var/log/app.log", "stdout", "stderr"}},
			expectedLog: "/var/log/app.log",
		},
		{
			name:        "Keywords first, then file path",
			logsCfg:     LogsConfig{OutputPaths: []string{"stdout", "stderr", "/var/log/system.log"}},
			expectedLog: "/var/log/system.log",
		},
		{
			name:        "File path in the middle of keywords",
			logsCfg:     LogsConfig{OutputPaths: []string{"stdout", "/var/log/middle.log", "stderr"}},
			expectedLog: "/var/log/middle.log",
		},
		{
			name:        "Multiple file paths",
			logsCfg:     LogsConfig{OutputPaths: []string{"/first.log", "/second.log"}},
			expectedLog: "/first.log",
		},
		{
			name:        "File path resembling a keyword",
			logsCfg:     LogsConfig{OutputPaths: []string{"stdout.log"}},
			expectedLog: "stdout.log",
		},
		{
			name:        "Mixed case keyword (treated as file)",
			logsCfg:     LogsConfig{OutputPaths: []string{"Stdout", "/var/log/app.log"}},
			expectedLog: "Stdout", // Current implementation is case-sensitive for keywords
		},
		{
			name:        "Empty string in paths",
			logsCfg:     LogsConfig{OutputPaths: []string{"", "/var/log/app.log"}},
			expectedLog: "", // Empty string is not a keyword, so it's returned
		},
		{
			name:        "File path with spaces (if valid on OS)",
			logsCfg:     LogsConfig{OutputPaths: []string{"/my logs/app.log"}},
			expectedLog: "/my logs/app.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedLog, getFirstFileOutputPath(tt.logsCfg))
		})
	}
}

func TestOTLPLogExport(t *testing.T) {
	version := "1.2.3"
	service := "test-service"
	testAttribute := "test-attribute"
	testValue := "test-value"
	receivedLogs := 0
	totalLogs := 10

	// Create a backend to receive the logs and assert the content
	srv := createBackend("/v1/logs", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		defer request.Body.Close()

		// Unmarshal the protobuf body into logs
		req := plogotlp.NewExportRequest()
		err = req.UnmarshalProto(body)
		assert.NoError(t, err)

		logs := req.Logs()
		rl := logs.ResourceLogs().At(0)

		resourceAttrs := rl.Resource().Attributes().AsRaw()
		assert.Equal(t, resourceAttrs[string(semconv.ServiceNameKey)], service)
		assert.Equal(t, resourceAttrs[string(semconv.ServiceVersionKey)], version)
		assert.Equal(t, resourceAttrs[testAttribute], testValue)

		// Check that the resource attributes are not duplicated in the log records
		sl := rl.ScopeLogs().At(0)
		logRecord := sl.LogRecords().At(0)
		attrs := logRecord.Attributes().AsRaw()
		assert.NotContains(t, attrs, string(semconv.ServiceNameKey))
		assert.NotContains(t, attrs, string(semconv.ServiceVersionKey))
		assert.NotContains(t, attrs, testAttribute)

		receivedLogs++

		writer.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	processors := []config.LogRecordProcessor{
		{
			Simple: &config.SimpleLogRecordProcessor{
				Exporter: config.LogRecordExporter{
					OTLP: &config.OTLP{
						Endpoint: ptr(srv.URL),
						Protocol: ptr("http/protobuf"),
						Insecure: ptr(true),
					},
				},
			},
		},
	}

	cfg := Config{
		Logs: LogsConfig{
			Level:       zapcore.DebugLevel,
			Development: true,
			Encoding:    "json",
			Processors:  processors,
		},
	}

	sdk, _ := config.NewSDK(
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				LoggerProvider: &config.LoggerProvider{
					Processors: processors,
				},
				Resource: &config.Resource{
					SchemaUrl: ptr(""),
					Attributes: []config.AttributeNameValue{
						{Name: string(semconv.ServiceNameKey), Value: service},
						{Name: string(semconv.ServiceVersionKey), Value: version},
						{Name: testAttribute, Value: testValue},
					},
				},
			},
		),
	)

	l, lp, err := newLogger(Settings{SDK: &sdk}, cfg)
	require.NoError(t, err)
	require.NotNil(t, l)
	require.NotNil(t, lp)

	defer func() {
		if prov, ok := lp.(shutdownable); ok {
			require.NoError(t, prov.Shutdown(context.Background()))
		}
	}()

	// Reset counter for each test case
	receivedLogs = 0

	// Generate some logs to send to the backend
	for i := 0; i < totalLogs; i++ {
		l.Info("Test log message")
	}

	// Ensure the correct number of logs were received
	require.Equal(t, totalLogs, receivedLogs)
}

func createBackend(endpoint string, handler func(writer http.ResponseWriter, request *http.Request)) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, handler)
	return httptest.NewServer(mux)
}
