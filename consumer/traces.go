// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"context"

	"go.opentelemetry.io/collector/consumer/internal"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Traces is an interface that receives ptrace.Traces, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Traces interface {
	internal.BaseConsumer
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
	*internal.BaseImpl
	ConsumeTracesFunc
}

// NewTraces returns a Traces configured with the provided options.
func NewTraces(consume ConsumeTracesFunc, options ...Option) (Traces, error) {
	if consume == nil {
		return nil, errNilFunc
	}

	baseImpl := internal.NewBaseImpl(options...)
	fn := func(ctx context.Context, td ptrace.Traces) error {
		ctx = baseImpl.ObsReport.StartTracesOp(ctx)
		spanCount := td.SpanCount()
		err := consume(ctx, td)
		baseImpl.ObsReport.EndTracesOp(ctx, spanCount, err)
		return err
	}

	return &baseTraces{
		BaseImpl:          baseImpl,
		ConsumeTracesFunc: fn,
	}, nil
}
