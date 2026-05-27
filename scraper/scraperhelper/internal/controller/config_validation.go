// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"errors"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

var errNonPositiveInterval = errors.New("requires positive value")

func validateControllerConfig(set *ControllerConfig) (errs error) {
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
