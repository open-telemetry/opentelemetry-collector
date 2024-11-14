// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelperprofiles // import "go.opentelemetry.io/collector/processor/processorhelper/processorhelperprofiles"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processorprofiles"
)

// ProcessProfilesFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessProfilesFunc func(context.Context, pprofile.Profiles) (pprofile.Profiles, error)

type profiles struct {
	component.StartFunc
	component.ShutdownFunc
	consumerprofiles.Profiles
}

// NewProfiles creates a processorprofiles.Profiles that ensure context propagation.
func NewProfiles(
	_ context.Context,
	_ processor.Settings,
	_ component.Config,
	nextConsumer consumerprofiles.Profiles,
	profilesFunc ProcessProfilesFunc,
	options ...Option,
) (processorprofiles.Profiles, error) {
	if profilesFunc == nil {
		return nil, errors.New("nil profilesFunc")
	}

	bs := fromOptions(options)
	profilesConsumer, err := consumerprofiles.NewProfiles(func(ctx context.Context, pd pprofile.Profiles) (err error) {
		pd, err = profilesFunc(ctx, pd)
		if err != nil {
			if errors.Is(err, processorhelper.ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		return nextConsumer.ConsumeProfiles(ctx, pd)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &profiles{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Profiles:     profilesConsumer,
	}, nil
}
