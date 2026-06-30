// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configretry // import "go.opentelemetry.io/collector/config/configretry"

import (
	"errors"
)

func validateMaxElapsedTime(bs *BackOffConfig) error {
	if bs.MaxElapsedTime > 0 {
		if bs.MaxElapsedTime < bs.InitialInterval {
			return errors.New("'max_elapsed_time' must not be less than 'initial_interval'")
		}
		if bs.MaxElapsedTime < bs.MaxInterval {
			return errors.New("'max_elapsed_time' must not be less than 'max_interval'")
		}
	}
	return nil
}
