// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metric // import "go.opentelemetry.io/collector/consumer/metric"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/internal/base"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var errNilFunc = errors.New("nil consumer func")

type config struct {
	baseOptions []base.Option
}

// Option to construct new consumers.
type Option func(config) config

// Metrics is an interface that receives pmetric.Metrics, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Metrics interface {
	base.Consumer
	// ConsumeMetrics receives pmetric.Metrics for consumption.
	ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error
}

// ConsumeMetricsFunc is a helper function that is similar to ConsumeMetrics.
type ConsumeMetricsFunc func(ctx context.Context, md pmetric.Metrics) error

// ConsumeMetrics calls f(ctx, md).
func (f ConsumeMetricsFunc) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return f(ctx, md)
}

type baseMetrics struct {
	*base.Impl
	ConsumeMetricsFunc
}

// NewMetrics returns a Metrics configured with the provided options.
func NewMetrics(consume ConsumeMetricsFunc, options ...Option) (Metrics, error) {
	if consume == nil {
		return nil, errNilFunc
	}

	cfg := config{}
	for _, op := range options {
		cfg = op(cfg)
	}

	return &baseMetrics{
		Impl:               base.NewImpl(cfg.baseOptions...),
		ConsumeMetricsFunc: consume,
	}, nil
}

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(c config) config {
		c.baseOptions = append(c.baseOptions, base.WithCapabilities(capabilities))
		return c
	}
}
