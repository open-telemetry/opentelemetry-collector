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
		name                string
		cfg                 zap.Config
		infoLogged          bool
		warnLogged          bool
		checkCallerLocation bool
	}{
		{
			"collector_info_level_grpc_log_warn",
			zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
				Encoding: "console",
			},
			false,
			true,
			false,
		},
		{
			"collector_debug_level_grpc_log_debug",
			zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
				Encoding: "console",
			},
			true,
			true,
			true,
		},
		{
			"collector_warn_level_grpc_log_warn",
			zap.Config{
				Development: false, // this must set the grpc loggerV2 to loggerV2
				Level:       zap.NewAtomicLevelAt(zapcore.WarnLevel),
				Encoding:    "console",
			},
			false,
			true,
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obsInfo, obsWarn := false, false
			CallerInfo := ""
			hook := zap.Hooks(func(entry zapcore.Entry) error {
				switch entry.Level {
				case zapcore.InfoLevel:
					obsInfo = true
				case zapcore.WarnLevel:
					obsWarn = true
				}
				if test.checkCallerLocation {
					CallerInfo = entry.Caller.TrimmedPath()
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
			// create a wrapper function to test the caller location
			if test.checkCallerLocation {
				wrapper := func() {
					glogger.Info("test message")
				}
				wrapper1 := func() {
					wrapper()
				}
				wrapper2 := func() {
					wrapper1()
				}
				wrapper3 := func() {
					wrapper2()
				}
				wrapper3()
				assert.Contains(t, CallerInfo, "grpclog/logger_test.go:95") // change line number to the line where wrapper3() is called in case of making changes
			}
			assert.Equal(t, obsInfo, test.infoLogged)
			assert.Equal(t, obsWarn, test.warnLogged)
		})
	}
}
