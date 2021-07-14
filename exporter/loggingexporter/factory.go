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

package loggingexporter

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr                   = "logging"
	defaultSamplingInitial    = 2
	defaultSamplingThereafter = 500
)

// NewFactory creates a factory for Logging exporter
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithTraces(createTracesExporter),
		exporterhelper.WithMetrics(createMetricsExporter),
		exporterhelper.WithLogs(createLogsExporter))
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings:   config.NewExporterSettings(config.NewID(typeStr)),
		LogLevel:           "info",
		SamplingInitial:    defaultSamplingInitial,
		SamplingThereafter: defaultSamplingThereafter,
	}
}

func createTracesExporter(_ context.Context, set component.ExporterCreateSettings, config config.Exporter) (component.TracesExporter, error) {
	cfg := config.(*Config)

	exporterLogger, err := createLogger(cfg)
	if err != nil {
		return nil, err
	}

	return newTracesExporter(config, cfg.LogLevel, exporterLogger, set)
}

func createMetricsExporter(_ context.Context, set component.ExporterCreateSettings, config config.Exporter) (component.MetricsExporter, error) {
	cfg := config.(*Config)

	exporterLogger, err := createLogger(cfg)
	if err != nil {
		return nil, err
	}

	return newMetricsExporter(config, cfg.LogLevel, exporterLogger, set)
}

func createLogsExporter(_ context.Context, set component.ExporterCreateSettings, config config.Exporter) (component.LogsExporter, error) {
	cfg := config.(*Config)

	exporterLogger, err := createLogger(cfg)
	if err != nil {
		return nil, err
	}

	return newLogsExporter(config, cfg.LogLevel, exporterLogger, set)
}

func createLogger(cfg *Config) (*zap.Logger, error) {
	var level zapcore.Level
	err := (&level).UnmarshalText([]byte(cfg.LogLevel))
	if err != nil {
		return nil, err
	}

	// We take development config as the base since it matches the purpose
	// of logging exporter being used for debugging reasons (so e.g. console encoder)
	conf := zap.NewDevelopmentConfig()
	conf.Level = zap.NewAtomicLevelAt(level)
	conf.Sampling = &zap.SamplingConfig{
		Initial:    cfg.SamplingInitial,
		Thereafter: cfg.SamplingThereafter,
	}

	logginglogger, err := conf.Build()
	if err != nil {
		return nil, err
	}
	return logginglogger, nil
}
