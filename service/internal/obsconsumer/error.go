// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"
	"errors"
)

const (
	componentErrorTypeCancelled        = "cancelled"
	componentErrorTypeDeadlineExceeded = "deadline_exceeded"
	componentErrorTypeUnspecified      = "unspecified"
)

func classifyError(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return componentErrorTypeCancelled
	case errors.Is(err, context.DeadlineExceeded):
		return componentErrorTypeDeadlineExceeded
	default:
		return componentErrorTypeUnspecified
	}
}
