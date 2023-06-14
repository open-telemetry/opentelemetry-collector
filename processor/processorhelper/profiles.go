// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
)

// ProcessProfilesFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessProfilesFunc func(context.Context, pprofile.Profiles) (pprofile.Profiles, error)

type profileProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Profiles
}

// NewProfilesProcessor creates a processor.Profiles that ensure context propagation and the right tags are set.
func NewProfilesProcessor(
	_ context.Context,
	set processor.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Profiles,
	profilesFunc ProcessProfilesFunc,
	options ...Option,
) (processor.Profiles, error) {
	// TODO: Add observability metrics support
	if profilesFunc == nil {
		return nil, errors.New("nil profilesFunc")
	}

	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	profilesConsumer, err := consumer.NewProfiles(func(ctx context.Context, ld pprofile.Profiles) error {
		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)
		var err error
		ld, err = profilesFunc(ctx, ld)
		span.AddEvent("End processing.", eventOptions)
		if err != nil {
			if errors.Is(err, ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		return nextConsumer.ConsumeProfiles(ctx, ld)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &profileProcessor{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Profiles:     profilesConsumer,
	}, nil
}
