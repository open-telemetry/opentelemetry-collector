// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Metrics is an interface that receives pmetric.Metrics, processes it
// as needed, and sends it to the next processing node if any or to the destination.
//
// Deprecated: use the cmetric subpackage instead.
type Metrics interface {
	baseConsumer
	// ConsumeMetrics receives pmetric.Metrics for consumption.
	ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error
}

// ConsumeMetricsFunc is a helper function that is similar to ConsumeMetrics.
//
// Deprecated: use the cmetric subpackage instead.
type ConsumeMetricsFunc func(ctx context.Context, md pmetric.Metrics) error

// ConsumeMetrics calls f(ctx, md).
//
// Deprecated: use the cmetric subpackage instead.
func (f ConsumeMetricsFunc) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return f(ctx, md)
}

type baseMetrics struct {
	*baseImpl
	ConsumeMetricsFunc
}

// NewMetrics returns a Metrics configured with the provided options.
//
// Deprecated: use the cmetric subpackage instead.
func NewMetrics(consume ConsumeMetricsFunc, options ...Option) (Metrics, error) {
	if consume == nil {
		return nil, errNilFunc
	}
	return &baseMetrics{
		baseImpl:           newBaseImpl(options...),
		ConsumeMetricsFunc: consume,
	}, nil
}
