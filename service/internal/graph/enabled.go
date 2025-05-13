// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type enabledInstrument interface {
	Enabled(context.Context) bool
}

func isEnabled(inst metric.Int64Counter) bool {
	_, ok := inst.(enabledInstrument)
	return ok
}
