// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package xscraperhelper provides utilities for scrapers.
package xscraperhelper // import "go.opentelemetry.io/collector/scraper/scraperhelper/xscraperhelper"

import (
	"errors"
	"fmt"
	"time"
)

var errNonPositiveInterval = errors.New("requires positive value")

// ControllerConfig defines common settings for a scraper controller
// configuration. Scraper controller receivers can embed this struct, instead
// of receiver.Settings, and extend it with more fields if needed.
type ControllerConfig struct {
	// CollectionInterval sets how frequently the scraper
	// should be called and used as the context timeout
	// to ensure that scrapers don't exceed the interval.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	// InitialDelay sets the initial start delay for the scraper,
	// any non positive value is assumed to be immediately.
	InitialDelay time.Duration `mapstructure:"initial_delay"`
	// Timeout is an optional value used to set scraper's context deadline.
	Timeout time.Duration `mapstructure:"timeout"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultControllerConfig returns default scraper controller
// settings with a collection interval of one minute.
func NewDefaultControllerConfig() ControllerConfig {
	return ControllerConfig{
		CollectionInterval: time.Minute,
		InitialDelay:       time.Second,
		Timeout:            0,
	}
}

func (set *ControllerConfig) Validate() error {
	var errs []error

	if set.CollectionInterval <= 0 {
		errs = append(errs, fmt.Errorf(`"collection_interval": %w`, errNonPositiveInterval))
	}
	if set.Timeout < 0 {
		errs = append(errs, fmt.Errorf(`"timeout": %w`, errNonPositiveInterval))
	}
	return errors.Join(errs...)
}
