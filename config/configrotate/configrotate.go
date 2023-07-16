// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configrotate // import "go.opentelemetry.io/collector/config/configrotate"

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	// Enabled controls whether or not rotate logs
	Enabled bool `mapstructure:"enabled"`

	// MaxMegabytes is the maximum size in megabytes of the file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxMegabytes int `mapstructure:"max_megabytes"`

	// MaxDays is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxDays int `mapstructure:"max_days"`

	// MaxBackups is the maximum number of old log files to retain. The default
	// is to 100 files.
	MaxBackups int `mapstructure:"max_backups"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `mapstructure:"localtime"`
}

func NewDefaultRotateConfig() *Config {
	return &Config{
		Enabled:      true,
		MaxMegabytes: 100,
		MaxDays:      0,
		MaxBackups:   100,
		LocalTime:    false,
	}
}

func (cfg *Config) NewWriter(filename string) (io.WriteCloser, error) {
	if cfg.MaxMegabytes < 0 {
		return nil, fmt.Errorf("invalid MaxMegabytes %d", cfg.MaxMegabytes)
	}
	if cfg.MaxDays < 0 {
		return nil, fmt.Errorf("invalid MaxDays %d", cfg.MaxDays)
	}
	// #nosec G302 G304 -- filename is a trusted safe path, and should allow to be read by other users
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return file, nil
	}
	return cfg.newLumberjackWriter(filename), nil
}

func (cfg *Config) newLumberjackWriter(filename string) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxMegabytes,
		MaxAge:     cfg.MaxDays,
		MaxBackups: cfg.MaxBackups,
		LocalTime:  cfg.LocalTime,
	}
}
