// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controller // import "go.opentelemetry.io/collector/scraper/scraperhelper/internal/controller"

import (
	"errors"
	"fmt"

	"go.uber.org/multierr"
)

var errNonPositiveInterval = errors.New("requires positive value")

func validateControllerConfig(set *ControllerConfig) (errs error) {
	if set.CollectionInterval <= 0 {
		errs = multierr.Append(errs, fmt.Errorf(`"collection_interval": %w`, errNonPositiveInterval))
	}
	if set.Timeout < 0 {
		errs = multierr.Append(errs, fmt.Errorf(`"timeout": %w`, errNonPositiveInterval))
	}
	return errs
}
