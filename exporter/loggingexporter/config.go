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
	"fmt"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
)

var (
	// supportedLevels in this exporter's configuration.
	// configtelemetry.LevelNone and other future values are not supported.
	supportedLevels map[configtelemetry.Level]struct{} = map[configtelemetry.Level]struct{}{
		configtelemetry.LevelBasic:    {},
		configtelemetry.LevelNormal:   {},
		configtelemetry.LevelDetailed: {},
	}
)

// Config defines configuration for logging exporter.
type Config struct {
	// LogLevel defines log level of the logging exporter; options are debug, info, warn, error.
	// Deprecated: Use `Verbosity` instead.
	LogLevel zapcore.Level `mapstructure:"loglevel,omitempty"`

	// Verbosity defines the logging exporter verbosity.
	Verbosity configtelemetry.Level `mapstructure:"verbosity,omitempty"`

	// SamplingInitial defines how many samples are initially logged during each second.
	SamplingInitial int `mapstructure:"sampling_initial"`

	// SamplingThereafter defines the sampling rate after the initial samples are logged.
	SamplingThereafter int `mapstructure:"sampling_thereafter"`

	// warnLogLevel is set on unmarshaling to warn users about `loglevel` usage.
	warnLogLevel bool
}

var _ component.Config = (*Config)(nil)
var _ confmap.Unmarshaler = (*Config)(nil)

func mapLevel(level zapcore.Level) (configtelemetry.Level, error) {
	switch level {
	case zapcore.DebugLevel:
		return configtelemetry.LevelDetailed, nil
	case zapcore.InfoLevel:
		return configtelemetry.LevelNormal, nil
	case zapcore.WarnLevel, zapcore.ErrorLevel,
		zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		// Anything above info is mapped to 'basic' level.
		return configtelemetry.LevelBasic, nil
	default:
		return configtelemetry.LevelNone, fmt.Errorf("log level %q is not supported", level)
	}
}

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	if conf.IsSet("loglevel") && conf.IsSet("verbosity") {
		return fmt.Errorf("'loglevel' and 'verbosity' are incompatible. Use only 'verbosity' instead")
	}

	if err := conf.Unmarshal(cfg, confmap.WithErrorUnused()); err != nil {
		return err
	}

	if conf.IsSet("loglevel") {
		verbosity, err := mapLevel(cfg.LogLevel)
		if err != nil {
			return fmt.Errorf("failed to map 'loglevel': %w", err)
		}

		// 'verbosity' is unset but 'loglevel' is set.
		// Override default verbosity.
		cfg.Verbosity = verbosity
		cfg.warnLogLevel = true
	}

	return nil
}

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if _, ok := supportedLevels[cfg.Verbosity]; !ok {
		return fmt.Errorf("verbosity level %q is not supported", cfg.Verbosity)
	}

	return nil
}
