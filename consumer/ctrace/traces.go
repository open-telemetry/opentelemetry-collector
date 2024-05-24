// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctrace // import "go.opentelemetry.io/collector/consumer/ctrace"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/internal/base"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var errNilFunc = errors.New("nil consumer func")

type config struct {
	baseOptions []base.Option
}

// Option to construct new consumers.
type Option func(config) config

// Traces is an interface that receives ptrace.Traces, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Traces interface {
	base.Consumer
	// ConsumeTraces receives ptrace.Traces for consumption.
	ConsumeTraces(ctx context.Context, td ptrace.Traces) error
}

// ConsumeTracesFunc is a helper function that is similar to ConsumeTraces.
type ConsumeTracesFunc func(ctx context.Context, td ptrace.Traces) error

// ConsumeTraces calls f(ctx, td).
func (f ConsumeTracesFunc) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return f(ctx, td)
}

type baseTraces struct {
	*base.Impl
	ConsumeTracesFunc
}

// NewTraces returns a Traces configured with the provided options.
func NewTraces(consume ConsumeTracesFunc, options ...Option) (Traces, error) {
	if consume == nil {
		return nil, errNilFunc
	}

	cfg := config{}

	for _, op := range options {
		cfg = op(cfg)
	}

	return &baseTraces{
		Impl:              base.NewImpl(cfg.baseOptions...),
		ConsumeTracesFunc: consume,
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
