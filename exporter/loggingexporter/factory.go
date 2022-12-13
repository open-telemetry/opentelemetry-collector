// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loggingexporter // import "go.opentelemetry.io/collector/exporter/loggingexporter"

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
)

const (
	// The value of "type" key in configuration.
	typeStr                   = "logging"
	defaultSamplingInitial    = 2
	defaultSamplingThereafter = 500
)

var onceWarnLogLevel sync.Once

// NewFactory creates a factory for Logging exporter
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, component.StabilityLevelDevelopment),
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelDevelopment),
		exporter.WithLogs(createLogsExporter, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		LogLevel:           zapcore.InfoLevel,
		Verbosity:          configtelemetry.LevelNormal,
		SamplingInitial:    defaultSamplingInitial,
		SamplingThereafter: defaultSamplingThereafter,
	}
}

func createTracesExporter(ctx context.Context, set exporter.CreateSettings, config component.Config) (exporter.Traces, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.TelemetrySettings.Logger)
	s := newLoggingExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewTracesExporter(ctx, set, cfg,
		s.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable Timeout/RetryOnFailure and SendingQueue
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{Enabled: false}),
		exporterhelper.WithShutdown(loggerSync(exporterLogger)),
	)
}

func createMetricsExporter(ctx context.Context, set exporter.CreateSettings, config component.Config) (exporter.Metrics, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.TelemetrySettings.Logger)
	s := newLoggingExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewMetricsExporter(ctx, set, cfg,
		s.pushMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable Timeout/RetryOnFailure and SendingQueue
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{Enabled: false}),
		exporterhelper.WithShutdown(loggerSync(exporterLogger)),
	)
}

func createLogsExporter(ctx context.Context, set exporter.CreateSettings, config component.Config) (exporter.Logs, error) {
	cfg := config.(*Config)
	exporterLogger := createLogger(cfg, set.TelemetrySettings.Logger)
	s := newLoggingExporter(exporterLogger, cfg.Verbosity)
	return exporterhelper.NewLogsExporter(ctx, set, cfg,
		s.pushLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		// Disable Timeout/RetryOnFailure and SendingQueue
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(exporterhelper.RetrySettings{Enabled: false}),
		exporterhelper.WithQueue(exporterhelper.QueueSettings{Enabled: false}),
		exporterhelper.WithShutdown(loggerSync(exporterLogger)),
	)
}

func createLogger(cfg *Config, logger *zap.Logger) *zap.Logger {
	if cfg.warnLogLevel {
		onceWarnLogLevel.Do(func() {
			logger.Warn(
				"'loglevel' option is deprecated in favor of 'verbosity'. Set 'verbosity' to equivalent value to preserve behavior.",
				zap.Stringer("loglevel", cfg.LogLevel),
				zap.Stringer("equivalent verbosity level", cfg.Verbosity),
			)
		})
	}

	core := zapcore.NewSamplerWithOptions(
		logger.Core(),
		1*time.Second,
		cfg.SamplingInitial,
		cfg.SamplingThereafter,
	)

	return zap.New(core)
}
