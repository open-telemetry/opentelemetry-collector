// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grpclog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGRPCLogger(t *testing.T) {
	tests := []struct {
		cfg        zap.Config
		name       string
		infoLogged bool
		warnLogged bool
	}{
		{
			name: "collector_info_level_grpc_log_warn",
			cfg: zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
				Encoding: "console",
			},
			infoLogged: false,
			warnLogged: true,
		},
		{
			name: "collector_debug_level_grpc_log_debug",
			cfg: zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
				Encoding: "console",
			},
			infoLogged: true,
			warnLogged: true,
		},
		{
			name: "collector_warn_level_grpc_log_warn",
			cfg: zap.Config{
				Development: false, // this must set the grpc loggerV2 to loggerV2
				Level:       zap.NewAtomicLevelAt(zapcore.WarnLevel),
				Encoding:    "console",
			},
			infoLogged: false,
			warnLogged: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obsInfo, obsWarn := false, false
			hook := zap.Hooks(func(entry zapcore.Entry) error {
				switch entry.Level {
				case zapcore.InfoLevel:
					obsInfo = true
				case zapcore.WarnLevel:
					obsWarn = true
				}
				return nil
			})

			// create new collector zap logger
			logger, err := test.cfg.Build(hook)
			assert.NoError(t, err)

			// create colGRPCLogger
			glogger := SetLogger(logger, test.cfg.Level.Level())
			assert.NotNil(t, glogger)

			glogger.Info(test.name)
			glogger.Warning(test.name)

			assert.Equal(t, obsInfo, test.infoLogged)
			assert.Equal(t, obsWarn, test.warnLogged)
		})
	}
}
