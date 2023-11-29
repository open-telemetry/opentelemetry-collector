// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package common // import "go.opentelemetry.io/collector/exporter/internal/common"

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/internal/otlptext"
)

var onceWarnLogLevel sync.Once

type Common struct {
	Verbosity          configtelemetry.Level
	WarnLogLevel       bool
	LogLevel           zapcore.Level
	SamplingInitial    int
	SamplingThereafter int
}

func CreateTracesExporter(ctx context.Context, set exporter.CreateSettings, config component.Config, c *Common) (exporter.Traces, error) {
	exporterLogger := c.createLogger(set.TelemetrySettings.Logger)
	s := newLoggingExporter(exporterLogger, c.Verbosity)
	return exporterhelper.NewTracesExporter(ctx, set, config,
		s.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func CreateMetricsExporter(ctx context.Context, set exporter.CreateSettings, config component.Config, c *Common) (exporter.Metrics, error) {
	exporterLogger := c.createLogger(set.TelemetrySettings.Logger)
	s := newLoggingExporter(exporterLogger, c.Verbosity)
	return exporterhelper.NewMetricsExporter(ctx, set, config,
		s.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func CreateLogsExporter(ctx context.Context, set exporter.CreateSettings, config component.Config, c *Common) (exporter.Logs, error) {
	exporterLogger := c.createLogger(set.TelemetrySettings.Logger)
	s := newLoggingExporter(exporterLogger, c.Verbosity)
	return exporterhelper.NewLogsExporter(ctx, set, config,
		s.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithShutdown(otlptext.LoggerSync(exporterLogger)),
	)
}

func (c *Common) createLogger(logger *zap.Logger) *zap.Logger {
	if c.WarnLogLevel {
		onceWarnLogLevel.Do(func() {
			logger.Warn(
				"'loglevel' option is deprecated in favor of 'verbosity'. Set 'verbosity' to equivalent value to preserve behavior.",
				zap.Stringer("loglevel", c.LogLevel),
				zap.Stringer("equivalent verbosity level", c.Verbosity),
			)
		})
	}

	core := zapcore.NewSamplerWithOptions(
		logger.Core(),
		1*time.Second,
		c.SamplingInitial,
		c.SamplingThereafter,
	)

	return zap.New(core)
}
