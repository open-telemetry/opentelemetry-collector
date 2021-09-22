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

package telemetrylogs

import (
	"flag"
	"fmt"

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
	// Command line pointer to logger level flag configuration.
	loggerLevelPtr   *string
	loggerProfilePtr *string
	loggerFormatPtr  *string
)

// Flags adds flags related to service telemetry logs to the given flagset.
// Deprecated: keep this flag for preventing the breaking change. Use `service::telemetry::logs` in config instead.
func Flags(flags *flag.FlagSet) {
	loggerLevelPtr = flags.String(logLevelCfg, "deprecated", "Deprecated. Define the logging configuration as part of the configuration file, under the 'service' section.")

	loggerProfilePtr = flags.String(logProfileCfg, "deprecated", "Deprecated. Define the logging configuration as part of the configuration file, under the 'service' section.")

	// Note: we use "console" by default for more human-friendly mode of logging (tab delimited, formatted timestamps).
	loggerFormatPtr = flags.String(logFormatCfg, "deprecated", "Deprecated. Define the logging configuration as part of the configuration file, under the 'service' section.")
}

func NewLogger(cfg config.ServiceTelemetryLogs, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// Check flags in the same order as before, the default configuration starts from "prod" profile.
	if *loggerProfilePtr != "deprecated" {
		switch *loggerProfilePtr {
		case "dev":
			cfg.Level = zap.DebugLevel
			cfg.Development = true
			// Copied from NewDevelopmentConfig.
			zapCfg = &zap.Config{
				Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
				Development:      true,
				Encoding:         "console",
				EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
				OutputPaths:      []string{"stderr"},
				ErrorOutputPaths: []string{"stderr"},
			}
		case "prod":
			cfg.Level = zap.InfoLevel
			cfg.Development = false
		default:
			return nil, fmt.Errorf("invalid value %s for %s flag", *loggerProfilePtr, logProfileCfg)
		}
	}

	if *loggerFormatPtr != "deprecated" {
		cfg.Encoding = *loggerFormatPtr
	}

	if *loggerLevelPtr != "deprecated" {
		lvl, err := parseLogLevel(*loggerLevelPtr)
		if err != nil {
			return nil, err
		}
		cfg.Level = lvl
	}

	// Now set user configurations:
	zapCfg.Level.SetLevel(cfg.Level)
	zapCfg.Development = cfg.Development
	zapCfg.Encoding = cfg.Encoding

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
	if *loggerLevelPtr != "deprecated" {
		logger.Warn("`log-level` command line option has been deprecated. Use `service::telemetry::logs` in config instead!")
	}

	if *loggerProfilePtr != "deprecated" {
		logger.Warn("`log-profile` command line option has been deprecated. Use `service::telemetry::logs` in config instead!")
	}

	if *loggerFormatPtr != "deprecated" {
		logger.Warn("`log-format` command line option has been deprecated. Use `service::telemetry::logs` in config instead!")
	}
}

func parseLogLevel(level string) (zapcore.Level, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return lvl, fmt.Errorf(`invalid logger level: %q, valid values are "DEBUG", "INFO", "WARN", "ERROR", "DPANIC", "PANIC", "FATAL"`, lvl)
	}
	return lvl, nil
}
