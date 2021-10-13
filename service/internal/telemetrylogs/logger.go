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

package telemetrylogs // import "go.opentelemetry.io/collector/service/internal/telemetrylogs"

import (
	"flag"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config"
)

const (
	logLevelCfg   = "log-level"
	logProfileCfg = "log-profile"
	logFormatCfg  = "log-format"
)

var (
	defaultLogValue = "deprecated"
	// Command line pointer to logger level flag configuration.
	loggerLevelPtr   = &defaultLogValue
	loggerProfilePtr = &defaultLogValue
	loggerFormatPtr  = &defaultLogValue
)

// Flags adds flags related to service telemetry logs to the given flagset.
// Deprecated: keep this flag for preventing the breaking change. Use `service::telemetry::logs` in config instead.
func Flags(flags *flag.FlagSet) {
	loggerLevelPtr = flags.String(logLevelCfg, defaultLogValue, "Deprecated. Define the logging configuration as part of the configuration file, under the 'service' section.")

	loggerProfilePtr = flags.String(logProfileCfg, defaultLogValue, "Deprecated. Define the logging configuration as part of the configuration file, under the 'service' section.")

	// Note: we use "console" by default for more human-friendly mode of logging (tab delimited, formatted timestamps).
	loggerFormatPtr = flags.String(logFormatCfg, defaultLogValue, "Deprecated. Define the logging configuration as part of the configuration file, under the 'service' section.")
}

func NewLogger(cfg config.ServiceTelemetryLogs, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:       zap.NewAtomicLevelAt(cfg.Level),
		Development: cfg.Development,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         cfg.Encoding,
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if zapCfg.Encoding == "console" {
		// Human-readable timestamps for console format of logs.
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}
	logDeprecatedMessages(logger)
	return logger, nil
}

func logDeprecatedMessages(logger *zap.Logger) {
	if *loggerLevelPtr != defaultLogValue {
		logger.Warn("`log-level` command line option has been deprecated. Use `service::telemetry::logs` in config instead!")
	}

	if *loggerProfilePtr != defaultLogValue {
		logger.Warn("`log-profile` command line option has been deprecated. Use `service::telemetry::logs` in config instead!")
	}

	if *loggerFormatPtr != defaultLogValue {
		logger.Warn("`log-format` command line option has been deprecated. Use `service::telemetry::logs` in config instead!")
	}
}
