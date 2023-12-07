// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scraperhelper // import "go.opentelemetry.io/collector/receiver/scraperhelper"

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

var (
	errNonPositiveInterval = errors.New("requires positive value")
)

// ScraperControllerSettings defines common settings for a scraper controller
// configuration. Scraper controller receivers can embed this struct, instead
// of receiver.Settings, and extend it with more fields if needed.
type ScraperControllerSettings struct {
	// CollectionInterval sets the how frequently the scraper
	// should be called and used as the context timeout
	// to ensure that scrapers don't exceed the interval.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	// InitialDelay sets the initial start delay for the scraper,
	// any non positive value is assumed to be immediately.
	InitialDelay time.Duration `mapstructure:"initial_delay"`
	// Timeout is an optional value used to set scraper's context deadline.
	Timeout time.Duration `mapstructure:"timeout"`
}

// NewDefaultScraperControllerSettings returns default scraper controller
// settings with a collection interval of one minute.
func NewDefaultScraperControllerSettings(component.Type) ScraperControllerSettings {
	return ScraperControllerSettings{
		CollectionInterval: time.Minute,
		InitialDelay:       time.Second,
		Timeout:            0,
	}
}

func (set *ScraperControllerSettings) Validate() (errs error) {
	if set.CollectionInterval <= 0 {
		errs = multierr.Append(errs, fmt.Errorf(`"collection_interval": %w`, errNonPositiveInterval))
	}
	if set.Timeout < 0 {
		errs = multierr.Append(errs, fmt.Errorf(`"timeout": %w`, errNonPositiveInterval))
	}
	return errs
}
