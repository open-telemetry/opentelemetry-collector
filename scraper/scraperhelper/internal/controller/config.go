// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

var errNonPositiveInterval = errors.New("requires positive value")

// ControllerConfig defines common settings for a scraper controller
// configuration. Scraper controller receivers can embed this struct, instead
// of receiver.Settings, and extend it with more fields if needed.
type ControllerConfig struct {
	// CollectionInterval sets how frequently the scraper
	// should be called and used as the context timeout
	// to ensure that scrapers don't exceed the interval.
	// Must be positive, or zero to disable timer-based scraping.
	// If zero, at least one controller extension must be configured.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	// InitialDelay sets the initial start delay for the scraper,
	// any non positive value is assumed to be immediately.
	InitialDelay time.Duration `mapstructure:"initial_delay"`
	// Timeout is an optional value used to set scraper's context deadline.
	Timeout time.Duration `mapstructure:"timeout"`
	// Controllers is a list of extension IDs that control when scrapes occur.
	// When specified, extensions can trigger scrapes based on external events.
	// If Controllers is non-empty, CollectionInterval may be zero to disable
	// timer-based scraping entirely.
	Controllers []component.ID `mapstructure:"controllers"`
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

func (set *ControllerConfig) Validate() (errs error) {
	if set.CollectionInterval < 0 || (set.CollectionInterval == 0 && len(set.Controllers) == 0) {
		errs = multierr.Append(errs, fmt.Errorf(`"collection_interval": %w`, errNonPositiveInterval))
	}
	if set.Timeout < 0 {
		errs = multierr.Append(errs, fmt.Errorf(`"timeout": %w`, errNonPositiveInterval))
	}
	seen := make(map[component.ID]int, len(set.Controllers))
	for _, id := range set.Controllers {
		seen[id]++
		if seen[id] == 2 {
			errs = multierr.Append(errs, fmt.Errorf(`"controllers": duplicate extension ID %q`, id))
		}
	}
	return errs
}
