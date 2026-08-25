// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package refconsumer // import "go.opentelemetry.io/collector/service/internal/refconsumer"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func NewMetrics(cons consumer.Metrics) consumer.Metrics {
	return refMetrics{
		consumer: cons,
	}
}

type refMetrics struct {
	consumer consumer.Metrics
}

// ConsumeMetrics measures telemetry before calling ConsumeMetrics because the data may be mutated downstream
func (c refMetrics) ConsumeMetrics(ctx context.Context, ld pmetric.Metrics) error {
	if pref.MarkPipelineOwnedMetrics(ld) {
		defer pref.UnrefMetrics(ld)
	}
	return c.consumer.ConsumeMetrics(ctx, ld)
}

func (c refMetrics) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
