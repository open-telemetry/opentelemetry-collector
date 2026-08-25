// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetrytest // import "go.opentelemetry.io/collector/internal/telemetry/telemetrytest"

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type mockInjectorCore struct {
	zapcore.Core
	dropped *[]string
}

func (mic mockInjectorCore) DropInjectedAttributes(droppedAttrs ...string) zapcore.Core {
	*mic.dropped = append(*mic.dropped, droppedAttrs...)
	return mic
}

func MockInjectorLogger(logger *zap.Logger, dropped *[]string) *zap.Logger {
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return mockInjectorCore{
			Core:    c,
			dropped: dropped,
		}
	}))
}
