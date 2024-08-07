// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package grpclog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
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
			true,
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
					CallerInfo = entry.Caller.String()
				}
				return nil
			})

			// create new collector zap logger
			logger, err := test.cfg.Build(hook)
			assert.NoError(t, err)

			// create colGRPCLogger
			glogger := SetLogger(logger, test.cfg.Level.Level())
			assert.NotNil(t, glogger)
			alogger := grpclog.Component("channelz")
			glogger.Info(test.name)
			glogger.Warning(test.name)
			Info(alogger, nil, "Hello World")
			if test.name != "collector_debug_level_grpc_log_debug" { // As info logs will only be logged in the zapcore.DebugLevel config logger
				Warning(alogger, nil, "Hello World Warning") // and for the other 2 tests will only log warning logs so for testing infof we dont call this at last
			}
			if obsInfo && obsWarn {
				assert.Contains(t, CallerInfo, "grpclog/logger_test.go:83")
			} else {
				assert.Contains(t, CallerInfo, "grpclog/logger_test.go:85")
			}
			assert.Equal(t, obsInfo, test.infoLogged)
			assert.Equal(t, obsWarn, test.warnLogged)
		})
	}
}

// This is to emulate how logging happens in the channelz of grpc
type Entity interface {
	isEntity()
	fmt.Stringer
	id() int64
}

type Severity int

const (
	// CtUnknown indicates unknown severity of a trace event.
	CtUnknown Severity = iota
	// CtInfo indicates info level severity of a trace event.
	CtInfo
	// CtWarning indicates warning level severity of a trace event.
	CtWarning
	// CtError indicates error level severity of a trace event.
	CtError
)

type TraceEvent struct {
	Desc     string
	Severity Severity
	Parent   *TraceEvent
}

func AddTraceEvent(l grpclog.DepthLoggerV2, e Entity, depth int, desc *TraceEvent) {
	// Log only the trace description associated with the bottom most entity.
	d := fmt.Sprintf("[%s]%s", e, desc.Desc)
	switch desc.Severity {
	case CtUnknown, CtInfo:
		l.InfoDepth(depth+1, d)
	case CtWarning:
		l.WarningDepth(depth+1, d)
	case CtError:
		l.ErrorDepth(depth+1, d)
	}
}

// Info logs and adds a trace event if channelz is on.
func Info(l grpclog.DepthLoggerV2, e Entity, args ...any) {
	AddTraceEvent(l, e, 1, &TraceEvent{
		Desc:     fmt.Sprint(args...),
		Severity: 1,
	})
}

// Infof logs and adds a trace event if channelz is on.
func Infof(l grpclog.DepthLoggerV2, e Entity, format string, args ...any) {
	AddTraceEvent(l, e, 1, &TraceEvent{
		Desc:     fmt.Sprintf(format, args...),
		Severity: CtInfo,
	})
}

func Warning(l grpclog.DepthLoggerV2, e Entity, args ...any) {
	AddTraceEvent(l, e, 1, &TraceEvent{
		Desc:     fmt.Sprint(args...),
		Severity: CtWarning,
	})
}

// Warningf logs and adds a trace event if channelz is on.
func Warningf(l grpclog.DepthLoggerV2, e Entity, format string, args ...any) {
	AddTraceEvent(l, e, 1, &TraceEvent{
		Desc:     fmt.Sprintf(format, args...),
		Severity: CtWarning,
	})
}

// Error logs and adds a trace event if channelz is on.
func Error(l grpclog.DepthLoggerV2, e Entity, args ...any) {
	AddTraceEvent(l, e, 1, &TraceEvent{
		Desc:     fmt.Sprint(args...),
		Severity: CtError,
	})
}

// Errorf logs and adds a trace event if channelz is on.
func Errorf(l grpclog.DepthLoggerV2, e Entity, format string, args ...any) {
	AddTraceEvent(l, e, 1, &TraceEvent{
		Desc:     fmt.Sprintf(format, args...),
		Severity: CtError,
	})
}
