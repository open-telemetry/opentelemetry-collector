// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"errors"
	"net/http"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/internal/memorylimiter"
)

func Error(w http.ResponseWriter, err error) {
	// non-retryable status
	status := http.StatusBadRequest
	if !consumererror.IsPermanent(err) {
		// retryable status
		if errors.Is(err, memorylimiter.ErrDataRefused) {
			status = http.StatusTooManyRequests
		} else {
			status = http.StatusServiceUnavailable
		}
	}
	http.Error(w, err.Error(), status)
}
