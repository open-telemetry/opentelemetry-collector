// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	internalTelemetry "go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/service/internal/componentattribute"
	"go.opentelemetry.io/collector/service/telemetry"
)

const (
	version       = "1.2.3"
	service       = "test-service"
	testAttribute = "test-attribute"
	testValue     = "test-value"
)

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name         string
		wantCoreType any
		wantErr      error
		cfg          Config
	}{
		{
			name:    "no log config",
			cfg:     Config{},
			wantErr: errors.New("no encoder name specified"),
		},
		{
			name: "log config with invalid processors",
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
									OTLP: &config.OTLP{}, // missing required fields
								},
							},
						},
					},
				},
			},
			wantErr: errors.New("no valid log exporter"),
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildInfo := component.BuildInfo{}
			_, provider, err := NewFactory().CreateLogger(context.Background(), telemetry.LoggerSettings{
				Settings: telemetry.Settings{BuildInfo: buildInfo},
			}, &tt.cfg)
			if tt.wantErr != nil {
				require.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				require.NoError(t, provider.Shutdown(context.Background()))
			}
		})
	}
}

func TestCreateLoggerWithResource(t *testing.T) {
	tests := []struct {
		name           string
		buildInfo      component.BuildInfo
		resourceConfig map[string]*string
		wantFields     map[string]string
	}{
		{
			name: "auto-populated fields only",
			buildInfo: component.BuildInfo{
				Command: "mycommand",
				Version: "1.0.0",
			},
			resourceConfig: map[string]*string{},
			wantFields: map[string]string{
				string(semconv.ServiceNameKey):       "mycommand",
				string(semconv.ServiceVersionKey):    "1.0.0",
				string(semconv.ServiceInstanceIDKey): "",
			},
		},
		{
			name: "override service.name",
			buildInfo: component.BuildInfo{
				Command: "mycommand",
				Version: "1.0.0",
			},
			resourceConfig: map[string]*string{
				string(semconv.ServiceNameKey): ptr("custom-service"),
			},
			wantFields: map[string]string{
				string(semconv.ServiceNameKey):       "custom-service",
				string(semconv.ServiceVersionKey):    "1.0.0",
				string(semconv.ServiceInstanceIDKey): "",
			},
		},
		{
			name: "override service.version",
			buildInfo: component.BuildInfo{
				Command: "mycommand",
				Version: "1.0.0",
			},
			resourceConfig: map[string]*string{
				string(semconv.ServiceVersionKey): ptr("2.0.0"),
			},
			wantFields: map[string]string{
				string(semconv.ServiceNameKey):       "mycommand",
				string(semconv.ServiceVersionKey):    "2.0.0",
				string(semconv.ServiceInstanceIDKey): "",
			},
		},
		{
			name: "custom field with auto-populated",
			buildInfo: component.BuildInfo{
				Command: "mycommand",
				Version: "1.0.0",
			},
			resourceConfig: map[string]*string{
				"custom.field": ptr("custom-value"),
			},
			wantFields: map[string]string{
				string(semconv.ServiceNameKey):       "mycommand",
				string(semconv.ServiceVersionKey):    "1.0.0",
				string(semconv.ServiceInstanceIDKey): "", // Just check presence
				"custom.field":                       "custom-value",
			},
		},
		{
			name:           "resource with no attributes",
			buildInfo:      component.BuildInfo{},
			resourceConfig: nil,
			wantFields: map[string]string{
				// A random UUID is injected for service.instance.id by default
				string(semconv.ServiceInstanceIDKey): "", // Just check presence
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, observedLogs := observer.New(zapcore.DebugLevel)
			set := telemetry.LoggerSettings{
				Settings: telemetry.Settings{BuildInfo: tt.buildInfo},
				ZapOptions: []zap.Option{
					// Redirect logs to the observer core
					zap.WrapCore(func(zapcore.Core) zapcore.Core { return core }),
				},
			}
			cfg := &Config{
				Logs: LogsConfig{
					Level:    zapcore.InfoLevel,
					Encoding: "json",
				},
				Resource: tt.resourceConfig,
			}

			logger, loggerProvider, err := createLogger(t.Context(), set, cfg)
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, loggerProvider.Shutdown(t.Context()))
			}()

			logger.Info("Test log message")
			require.Len(t, observedLogs.All(), 1)

			entry := observedLogs.All()[0]
			if tt.wantFields == nil {
				assert.Empty(t, entry.Context)
				return
			}

			assert.Equal(t, "resource", entry.Context[0].Key)
			dict := entry.Context[0].Interface.(zapcore.ObjectMarshaler)
			enc := zapcore.NewMapObjectEncoder()
			require.NoError(t, dict.MarshalLogObject(enc))

			// Verify all expected fields
			for k, v := range tt.wantFields {
				if k == string(semconv.ServiceInstanceIDKey) {
					// For service.instance.id just verify it exists since it's auto-generated
					assert.Contains(t, enc.Fields, k)
				} else {
					assert.Equal(t, v, enc.Fields[k])
				}
			}
		})
	}
}

func TestLogger_OTLP(t *testing.T) {
	// Create a backend to receive the logs and assert the content
	receivedLogs := 0
	logger := newOTLPLogger(t, zapcore.InfoLevel, func(req plogotlp.ExportRequest) {
		logs := req.Logs()
		rl := logs.ResourceLogs().At(0)

		resourceAttrs := rl.Resource().Attributes().AsRaw()
		assert.Equal(t, service, resourceAttrs[string(semconv.ServiceNameKey)])
		assert.Equal(t, version, resourceAttrs[string(semconv.ServiceVersionKey)])
		assert.Equal(t, testValue, resourceAttrs[testAttribute])

		// Check that the resource attributes are not duplicated in the log records
		sl := rl.ScopeLogs().At(0)
		logRecord := sl.LogRecords().At(0)
		attrs := logRecord.Attributes().AsRaw()
		assert.NotContains(t, attrs, string(semconv.ServiceNameKey))
		assert.NotContains(t, attrs, string(semconv.ServiceVersionKey))
		assert.NotContains(t, attrs, testAttribute)

		receivedLogs++
	})

	const totalLogs = 10
	for range totalLogs {
		logger.Info("Test log message")
	}

	// Ensure the correct number of logs were received
	require.Equal(t, totalLogs, receivedLogs)
}

func newOTLPLogger(t *testing.T, level zapcore.Level, handler func(plogotlp.ExportRequest)) *zap.Logger {
	srv := createLogsBackend(t, "/v1/logs", handler)
	t.Cleanup(srv.Close)

	processors := []config.LogRecordProcessor{{
		Simple: &config.SimpleLogRecordProcessor{
			Exporter: config.LogRecordExporter{
				OTLP: &config.OTLP{
					Endpoint: ptr(srv.URL),
					Protocol: ptr("http/protobuf"),
					Insecure: ptr(true),
				},
			},
		},
	}}

	cfg := &Config{
		Logs: LogsConfig{
			Level:      level,
			Encoding:   "json",
			Processors: processors,
			// OutputPaths is empty, so logs are only
			// written to the OTLP processor
		},
		Resource: map[string]*string{
			string(semconv.ServiceNameKey):    ptr(service),
			string(semconv.ServiceVersionKey): ptr(version),
			testAttribute:                     ptr(testValue),
		},
	}

	logger, shutdown, err := createLogger(t.Context(), telemetry.LoggerSettings{}, cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, shutdown.Shutdown(context.WithoutCancel(t.Context())))
	})
	return logger
}

func createLogsBackend(t *testing.T, endpoint string, handler func(plogotlp.ExportRequest)) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(_ http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		defer request.Body.Close()

		// Unmarshal the protobuf body into logs
		req := plogotlp.NewExportRequest()
		err = req.UnmarshalProto(body)
		assert.NoError(t, err)

		handler(req)
	})
	return httptest.NewServer(mux)
}

func TestLogAttributeInjection(t *testing.T) {
	core, consoleLogs := observer.New(zapcore.DebugLevel)

	var otlpLogs []plogotlp.ExportRequest
	srv := createLogsBackend(t, "/v1/logs", func(req plogotlp.ExportRequest) {
		otlpLogs = append(otlpLogs, req)
	})
	t.Cleanup(srv.Close)

	set := telemetry.LoggerSettings{
		ZapOptions: []zap.Option{
			// Redirect console logs to the observer core
			zap.WrapCore(func(zapcore.Core) zapcore.Core { return core }),
		},
	}
	cfg := &Config{
		Resource: map[string]*string{
			"service.instance.id": nil,
			"service.name":        nil,
			"service.version":     nil,
		},
		Logs: LogsConfig{
			Encoding: "json",
			Processors: []config.LogRecordProcessor{{
				Simple: &config.SimpleLogRecordProcessor{
					Exporter: config.LogRecordExporter{
						OTLP: &config.OTLP{
							// Send OTLP logs to the mock backend
							Endpoint: ptr(srv.URL),
							Protocol: ptr("http/protobuf"),
							Insecure: ptr(true),
						},
					},
				},
			}},
		},
	}

	sourceLogger, loggerProvider, err := createLogger(t.Context(), set, cfg)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, loggerProvider.Shutdown(t.Context()))
	}()

	ts := componenttest.NewNopTelemetrySettings()
	ts.Logger = sourceLogger
	ts = componentattribute.TelemetrySettingsWithAttributes(ts, attribute.NewSet(
		attribute.String("injected1", "val"),
		attribute.String("injected2", "val"),
	))
	ts.Logger = ts.Logger.With(zap.String("after", "val"))

	fields, scope := checkScopes(t, ts.Logger, consoleLogs, &otlpLogs)
	assert.JSONEq(t, `{"injected1":"val","injected2":"val","after":"val","manual":"val"}`, fields)
	assert.JSONEq(t, `{"injected1":"val","injected2":"val"}`, scope)

	ts = internalTelemetry.DropInjectedAttributes(ts, "injected1")

	fields, scope = checkScopes(t, ts.Logger, consoleLogs, &otlpLogs)
	assert.JSONEq(t, `{"injected2":"val","after":"val","manual":"val"}`, fields)
	assert.JSONEq(t, `{"injected2":"val"}`, scope)
}

func checkScopes(t *testing.T, logger *zap.Logger, consoleLogs *observer.ObservedLogs, otlpLogs *[]plogotlp.ExportRequest) (string, string) {
	logger.Info("Test log message", zap.String("manual", "val"))

	require.Len(t, consoleLogs.All(), 1)
	log := consoleLogs.TakeAll()[0]
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
	fieldsBuf, err := enc.EncodeEntry(log.Entry, log.Context)
	require.NoError(t, err)
	fieldsStr := strings.TrimSuffix(fieldsBuf.String(), "\n")

	require.Len(t, *otlpLogs, 1)
	req := (*otlpLogs)[0]
	*otlpLogs = nil
	rls := req.Logs().ResourceLogs()
	require.Equal(t, 1, rls.Len())
	sls := rls.At(0).ScopeLogs()
	require.Equal(t, 1, sls.Len())
	attrs := sls.At(0).Scope().Attributes()
	scopeBuf, err := json.Marshal(attrs.AsRaw())
	require.NoError(t, err)
	scopeStr := string(scopeBuf)

	return fieldsStr, scopeStr
}
