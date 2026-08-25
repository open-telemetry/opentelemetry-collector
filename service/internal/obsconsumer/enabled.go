// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type enabledInstrument interface {
	Enabled(context.Context) bool
}

func isEnabled(ctx context.Context, inst metric.Int64Counter) bool {
	ei, ok := inst.(enabledInstrument)
	return !ok || ei.Enabled(ctx)
}
