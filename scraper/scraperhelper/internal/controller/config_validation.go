// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"errors"
	"fmt"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

func validateControllerConfig(set *ControllerConfig) (errs error) {
	if set.CollectionInterval < 0 || (set.CollectionInterval == 0 && len(set.Controllers) == 0) {
		errs = multierr.Append(errs, errors.New(`"collection_interval": requires positive value, or zero when controllers is non-empty`))
	}
	if set.Timeout < 0 {
		errs = multierr.Append(errs, errors.New(`"timeout": requires positive value`))
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
