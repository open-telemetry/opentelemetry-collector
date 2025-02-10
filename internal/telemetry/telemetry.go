package telemetry

import (
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func LoggerWithout(ts component.TelemetrySettings, fields ...string) *zap.Logger {
	type coreWithout interface {
		Without(fields ...string) zapcore.Core
	}
	if _, ok := ts.Logger.Core().(coreWithout); !ok {
		return ts.Logger
	}
	return ts.Logger.WithOptions(
		zap.WrapCore(func(from zapcore.Core) zapcore.Core {
			return from.(coreWithout).Without(fields...)
		}),
	)
}
