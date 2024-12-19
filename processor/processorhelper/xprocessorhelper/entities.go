// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessorhelper // import "go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pentity"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

// ProcessEntitiesFunc is a helper function that processes the incoming data and returns the data to be sent to the next
// component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessEntitiesFunc func(context.Context, pentity.Entities) (pentity.Entities, error)

type entities struct {
	component.StartFunc
	component.ShutdownFunc
	xconsumer.Entities
}

// NewEntities creates a xprocessor.Entities that ensure context propagation and the right tags are set.
func NewEntities(
	_ context.Context,
	_ processor.Settings,
	_ component.Config,
	nextConsumer xconsumer.Entities,
	entitiesFunc ProcessEntitiesFunc,
	options ...Option,
) (xprocessor.Entities, error) {
	if entitiesFunc == nil {
		return nil, errors.New("nil entitiesFunc")
	}

	bs := fromOptions(options)
	entitiesConsumer, err := xconsumer.NewEntities(func(ctx context.Context, pd pentity.Entities) (err error) {
		pd, err = entitiesFunc(ctx, pd)
		if err != nil {
			if errors.Is(err, processorhelper.ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		return nextConsumer.ConsumeEntities(ctx, pd)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &entities{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Entities:     entitiesConsumer,
	}, nil
}
