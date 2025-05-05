// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap/zapcore"
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
			wantCoreType: "*componentattribute.wrapperCoreWithAttributes",
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

		testCoreType(t, tt.wantCoreType)
	}
}
