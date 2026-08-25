// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type disabledCounter struct {
	metric.Int64Counter
}

func newDisabledCounter(embedded metric.Int64Counter) *disabledCounter {
	return &disabledCounter{Int64Counter: embedded}
}

func (m *disabledCounter) Enabled(context.Context) bool {
	return false
}

func (m *disabledCounter) Add(ctx context.Context, value int64, opts ...metric.AddOption) {
	m.Int64Counter.Add(ctx, value, opts...)
}
