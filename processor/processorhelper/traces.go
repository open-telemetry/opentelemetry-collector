// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
)

// ProcessTracesFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessTracesFunc func(context.Context, ptrace.Traces) (ptrace.Traces, error)

type traces struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Traces
}

// NewTraces creates a processor.Traces that ensure context propagation and the right tags are set.
func NewTraces(
	_ context.Context,
	set processor.Settings,
	_ component.Config,
	nextConsumer consumer.Traces,
	tracesFunc ProcessTracesFunc,
	options ...Option,
) (processor.Traces, error) {
	if tracesFunc == nil {
		return nil, errors.New("nil tracesFunc")
	}

	obs, err := newObsReport(set, pipeline.SignalTraces)
	if err != nil {
		return nil, err
	}

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	traceConsumer, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)

		startTime := time.Now()

		spansIn := td.SpanCount()

		var errFunc error
		td, errFunc = tracesFunc(ctx, td)
		obs.recordInternalDuration(ctx, startTime)
		span.AddEvent("End processing.", eventOptions)
		if errFunc != nil {
			obs.recordInOut(ctx, spansIn, 0)
			if errors.Is(errFunc, ErrSkipProcessingData) {
				return nil
			}
			return errFunc
		}
		spansOut := td.SpanCount()
		obs.recordInOut(ctx, spansIn, spansOut)
		return nextConsumer.ConsumeTraces(ctx, td)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &traces{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Traces:       traceConsumer,
	}, nil
}
