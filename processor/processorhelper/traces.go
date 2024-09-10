// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

// ProcessTracesFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessTracesFunc func(context.Context, ptrace.Traces) (ptrace.Traces, error)

type tracesProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Traces
}

// NewTracesProcessor creates a processor.Traces that ensure context propagation and the right tags are set.
func NewTracesProcessor(
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

	obs, err := newObsReport(ObsReportSettings{
		ProcessorID:             set.ID,
		ProcessorCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}
	obs.otelAttrs = append(obs.otelAttrs, attribute.String("otel.signal", "traces"))

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	marshaler := new(ptrace.ProtoMarshaler)
	traceConsumer, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		var spansIn, spansOut, bytesIn, bytesOut int

		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)
		spansIn = td.SpanCount()
		if obs.metricLevel >= configtelemetry.LevelDetailed {
			bytesIn = marshaler.TracesSize(td)
		}

		td, err = tracesFunc(ctx, td)
		span.AddEvent("End processing.", eventOptions)
		if err != nil {
			if errors.Is(err, ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		spansOut = td.SpanCount()
		obs.recordInOut(ctx, spansIn, spansOut)
		if obs.metricLevel >= configtelemetry.LevelDetailed {
			bytesOut = marshaler.TracesSize(td)
			obs.recordInOutSize(ctx, bytesIn, bytesOut)
		}
		return nextConsumer.ConsumeTraces(ctx, td)
	}, bs.consumerOptions...)

	if err != nil {
		return nil, err
	}

	return &tracesProcessor{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Traces:       traceConsumer,
	}, nil
}
