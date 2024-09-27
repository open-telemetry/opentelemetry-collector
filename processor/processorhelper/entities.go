// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pentity"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
)

// ProcessEntitiesFunc is a helper function that processes the incoming data and returns the data to be sent to the next
// component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessEntitiesFunc func(context.Context, pentity.Entities) (pentity.Entities, error)

type entities struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Entities
}

// NewEntities creates a processor.Entities that ensure context propagation and the right tags are set.
func NewEntities(
	_ context.Context,
	set processor.Settings,
	_ component.Config,
	nextConsumer consumer.Entities,
	logsFunc ProcessEntitiesFunc,
	options ...Option,
) (processor.Entities, error) {
	if logsFunc == nil {
		return nil, errors.New("nil logsFunc")
	}

	obs, err := newObsReport(set, pipeline.SignalEntities)
	if err != nil {
		return nil, err
	}

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	logsConsumer, err := consumer.NewEntities(func(ctx context.Context, ld pentity.Entities) error {
		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)
		recordsIn := ld.EntityCount()

		var errFunc error
		ld, errFunc = logsFunc(ctx, ld)
		span.AddEvent("End processing.", eventOptions)
		if errFunc != nil {
			obs.recordInOut(ctx, recordsIn, 0)
			if errors.Is(errFunc, ErrSkipProcessingData) {
				return nil
			}
			return errFunc
		}
		recordsOut := ld.EntityCount()
		obs.recordInOut(ctx, recordsIn, recordsOut)
		return nextConsumer.ConsumeEntities(ctx, ld)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &entities{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Entities:     logsConsumer,
	}, nil
}
