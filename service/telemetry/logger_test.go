// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
)

type shutdownable interface {
	Shutdown(context.Context) error
}

func setGate(t *testing.T, gate *featuregate.Gate, value bool) {
	initialValue := gate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), value))
	t.Cleanup(func() {
		_ = featuregate.GlobalRegistry().Set(gate.ID(), initialValue)
	})
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name            string
		wantCoreType    any
		wantCoreTypeRfc any
		wantErr         error
		cfg             Config
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
			wantCoreType:    "*zapcore.ioCore",
			wantCoreTypeRfc: "*componentattribute.consoleCoreWithAttributes",
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
			wantCoreType:    "*zapcore.levelFilterCore",
			wantCoreTypeRfc: "*componentattribute.otelTeeCoreWithAttributes",
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
			wantCoreType:    "*zapcore.sampler",
			wantCoreTypeRfc: "*componentattribute.wrapperCoreWithAttributes",
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
				type shutdownable interface {
					Shutdown(context.Context) error
				}
				if prov, ok := lp.(shutdownable); ok {
					require.NoError(t, prov.Shutdown(context.Background()))
				}
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			setGate(t, telemetry.NewPipelineTelemetryGate, false)
			testCoreType(t, tt.wantCoreType)
		})
		t.Run(tt.name+" (pipeline telemetry on)", func(t *testing.T) {
			setGate(t, telemetry.NewPipelineTelemetryGate, true)
			testCoreType(t, tt.wantCoreTypeRfc)
		})
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
	require.Equal(t, "myvalue", entry.Context[0].String)
	require.Equal(t, "myfield", entry.Context[0].Key)
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
		assert.Equal(t, resourceAttrs[semconv.AttributeServiceName], service)
		assert.Equal(t, resourceAttrs[semconv.AttributeServiceVersion], version)
		assert.Equal(t, resourceAttrs[testAttribute], testValue)

		// Check that the resource attributes are not duplicated in the log records
		sl := rl.ScopeLogs().At(0)
		logRecord := sl.LogRecords().At(0)
		attrs := logRecord.Attributes().AsRaw()
		assert.NotContains(t, attrs, semconv.AttributeServiceName)
		assert.NotContains(t, attrs, semconv.AttributeServiceVersion)
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
						{Name: semconv.AttributeServiceName, Value: service},
						{Name: semconv.AttributeServiceVersion, Value: version},
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
