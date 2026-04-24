// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grpclog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
)

func TestGRPCLogger(t *testing.T) {
	tests := []struct {
		name       string
		cfg        zap.Config
		infoLogged bool
		warnLogged bool
	}{
		{
			"collector_info_level_grpc_log_warn",
			zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.InfoLevel),
				Encoding: "console",
			},
			false,
			true,
		},
		{
			"collector_debug_level_grpc_log_debug",
			zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
				Encoding: "console",
			},
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
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obsInfo, obsWarn := false, false
			callerInfo := ""
			hook := zap.Hooks(func(entry zapcore.Entry) error {
				switch entry.Level {
				case zapcore.InfoLevel:
					obsInfo = true
				case zapcore.WarnLevel:
					obsWarn = true
				}
				callerInfo = entry.Caller.String()
				return nil
			})

			// create new collector zap logger
			logger, err := test.cfg.Build(hook)
			require.NoError(t, err)

			// create GRPCLogger
			glogger := SetLogger(logger)
			assert.NotNil(t, glogger)
			// grpc does not usually call the logger directly, but through various wrappers that add extra depth
			component := &mockComponent{logger: grpclog.Component("channelz")}
			component.Info(test.name)
			component.Warning(test.name)
			assert.Equal(t, obsInfo, test.infoLogged)
			assert.Equal(t, obsWarn, test.warnLogged)
			// match the file name and line number of Warning() call above
			assert.Contains(t, callerInfo, "internal/grpclog/logger_test.go:77")
		})
	}
}

func TestBenignClientCloseWarningsSuppressed(t *testing.T) {
	benignMessages := []string{
		"transport: http2Server.HandleStreams failed to read frame: EOF",
		"transport: http2Server.HandleStreams failed to read frame: read tcp: connection reset by peer",
		"[transport] connection reset by peer",
	}

	for _, msg := range benignMessages {
		t.Run(msg, func(t *testing.T) {
			warnLogged := false
			hook := zap.Hooks(func(entry zapcore.Entry) error {
				if entry.Level == zapcore.WarnLevel {
					warnLogged = true
				}
				return nil
			})

			cfg := zap.Config{
				Level:    zap.NewAtomicLevelAt(zapcore.WarnLevel),
				Encoding: "console",
			}
			logger, err := cfg.Build(hook)
			require.NoError(t, err)

			SetLogger(logger)
			component := &mockComponent{logger: grpclog.Component("transport")}
			component.Warning(msg)

			assert.False(t, warnLogged, "benign client-close message should be suppressed: %s", msg)
		})
	}
}

func TestNonBenignWarningsPreserved(t *testing.T) {
	warnLogged := false
	hook := zap.Hooks(func(entry zapcore.Entry) error {
		if entry.Level == zapcore.WarnLevel {
			warnLogged = true
		}
		return nil
	})

	cfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(zapcore.WarnLevel),
		Encoding: "console",
	}
	logger, err := cfg.Build(hook)
	require.NoError(t, err)

	SetLogger(logger)
	component := &mockComponent{logger: grpclog.Component("channelz")}
	component.Warning("some unexpected grpc error unrelated to client close")
	assert.True(t, warnLogged, "non-benign warning should still be logged at warn level")
}

func TestIsBenignClientCloseMessage(t *testing.T) {
	assert.True(t, isBenignClientCloseMessage("HandleStreams failed to read frame: EOF"))
	assert.True(t, isBenignClientCloseMessage("read tcp: connection reset by peer"))
	assert.False(t, isBenignClientCloseMessage("unexpected internal error"))
	assert.False(t, isBenignClientCloseMessage("dial tcp: connection refused"))
}

type mockComponent struct {
	logger grpclog.DepthLoggerV2
}

func (c *mockComponent) Info(args ...any) {
	c.logger.Info(args...)
}

func (c *mockComponent) Warning(args ...any) {
	c.logger.Warning(args...)
}
