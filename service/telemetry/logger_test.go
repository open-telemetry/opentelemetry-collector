// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/config"
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
				type shutdownable interface {
					Shutdown(context.Context) error
				}
				if prov, ok := lp.(shutdownable); ok {
					require.NoError(t, prov.Shutdown(context.Background()))
				}
			}
		})
	}
}
