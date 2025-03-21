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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	semconv "go.opentelemetry.io/collector/semconv/v1.18.0"
)

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
			wantCoreType: "*zapcore.ioCore",
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
			wantCoreType: "*zapcore.levelFilterCore",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdk, _ := config.NewSDK(config.WithOpenTelemetryConfiguration(config.OpenTelemetryConfiguration{LoggerProvider: &config.LoggerProvider{
				Processors: tt.cfg.Logs.Processors,
			}}))

			l, lp, err := newLogger(Settings{SDK: &sdk}, tt.cfg)
			if tt.wantErr != nil {
				require.ErrorContains(t, err, tt.wantErr.Error())
				require.Nil(t, tt.wantCoreType)
			} else {
				require.NoError(t, err)
				gotType := reflect.TypeOf(l.Core()).String()
				require.Equal(t, tt.wantCoreType, gotType)
				if prov, ok := lp.(shutdownable); ok {
					require.NoError(t, prov.Shutdown(context.Background()))
				}
			}
		})
	}
}

func TestOTELZapIntegration(t *testing.T) {
	version := "1.2.3"
	service := "test-service"
	testAttribute := "test-attribute"

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
		sl := rl.ScopeLogs().At(0)
		logRecord := sl.LogRecords().At(0)
		attrs := logRecord.Attributes().AsRaw()

		assert.Contains(t, attrs, semconv.AttributeServiceName)
		assert.Contains(t, attrs, semconv.AttributeServiceVersion)
		assert.Contains(t, attrs, testAttribute)
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

	sdk, _ := config.NewSDK(
		config.WithOpenTelemetryConfiguration(
			config.OpenTelemetryConfiguration{
				LoggerProvider: &config.LoggerProvider{
					Processors: processors,
				},
			},
		),
	)

	cfg := Config{
		Logs: LogsConfig{
			Level:       zapcore.DebugLevel,
			Development: true,
			Encoding:    "json",
			Processors:  processors,
		},
		Resource: map[string]*string{
			semconv.AttributeServiceName:    ptr(service),
			semconv.AttributeServiceVersion: ptr(version),
			testAttribute:                   ptr("test-value"),
		},
	}
	l, lp, err := newLogger(Settings{SDK: &sdk}, cfg)
	require.NoError(t, err)
	require.NotNil(t, l)
	require.NotNil(t, lp)

	defer func() {
		if prov, ok := lp.(shutdownable); ok {
			require.NoError(t, prov.Shutdown(context.Background()))
		}
	}()

	// Generate some logs to send to the backend
	for i := 0; i < totalLogs; i++ {
		l.Info("Test log message")
	}

	// Ensure the correct number of logs were received
	assert.Equal(t, totalLogs, receivedLogs)
}

type shutdownable interface {
	Shutdown(context.Context) error
}

func createBackend(endpoint string, handler func(writer http.ResponseWriter, request *http.Request)) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, handler)

	srv := httptest.NewServer(mux)

	return srv
}
