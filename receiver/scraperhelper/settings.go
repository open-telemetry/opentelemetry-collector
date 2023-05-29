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
	ErrNonPositiveInterval    = errors.New("requires positive value")
	ErrTimeoutExceedsInterval = errors.New("timeout value exceeds collection interval")
)

// ScraperControllerSettings defines common settings for a scraper controller
// configuration. Scraper controller receivers can embed this struct, instead
// of receiver.Settings, and extend it with more fields if needed.
type ScraperControllerSettings struct {
	// CollectionInterval sets the how frequently the scraper
	// should be called and used as the context timeout
	// to ensure that scrapers don't exceed the interval.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	InitialDelay       time.Duration `mapstructure:"intial_delay"`
	// Timeout is used to set scraper's context deadline, it must be within
	// the range of (0, CollectionInterval]
	Timeout time.Duration `mapstructure:"timeout"`
}

// NewDefaultScraperControllerSettings returns default scraper controller
// settings with a collection interval of one minute.
func NewDefaultScraperControllerSettings(component.Type) ScraperControllerSettings {
	return ScraperControllerSettings{
		CollectionInterval: time.Minute,
		Timeout:            time.Minute,
	}
}

func (set *ScraperControllerSettings) Validate() (errs error) {
	if set.CollectionInterval < 1 {
		errs = multierr.Append(errs, fmt.Errorf(`"collection_interval": %w`, ErrNonPositiveInterval))
	}
	if set.Timeout < 1 {
		errs = multierr.Append(errs, fmt.Errorf(`"timeout": %w`, ErrNonPositiveInterval))
	}
	if set.Timeout > set.CollectionInterval {
		errs = multierr.Append(errs, ErrTimeoutExceedsInterval)
	}
	return errs
}
