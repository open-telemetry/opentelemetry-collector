// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package trace // import "go.opentelemetry.io/collector/consumer/trace"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Traces is an interface that receives ptrace.Traces, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Traces interface {
	consumer.Consumer
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
	*consumer.BaseImpl
	ConsumeTracesFunc
}

// NewTraces returns a Traces configured with the provided options.
func NewTraces(consume ConsumeTracesFunc, options ...consumer.Option) (Traces, error) {
	if consume == nil {
		return nil, consumer.ErrNilFunc
	}
	return &baseTraces{
		BaseImpl:          consumer.NewBaseImpl(options...),
		ConsumeTracesFunc: consume,
	}, nil
}
