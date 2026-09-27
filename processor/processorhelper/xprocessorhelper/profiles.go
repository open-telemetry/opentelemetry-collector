// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessorhelper // import "go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/obsreport"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

// ProcessProfilesFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessProfilesFunc func(context.Context, pprofile.Profiles) (pprofile.Profiles, error)

type profiles struct {
	component.StartFunc
	component.ShutdownFunc
	xconsumer.Profiles
}

// NewProfiles creates a xprocessor.Profiles that ensure context propagation.
func NewProfiles(
	_ context.Context,
	set processor.Settings,
	_ component.Config,
	nextConsumer xconsumer.Profiles,
	profilesFunc ProcessProfilesFunc,
	options ...Option,
) (xprocessor.Profiles, error) {
	if profilesFunc == nil {
		return nil, errors.New("nil profilesFunc")
	}

	obs, err := obsreport.New(set, xpipeline.SignalProfiles)
	if err != nil {
		return nil, err
	}

	bs := fromOptions(options)
	profilesConsumer, err := xconsumer.NewProfiles(func(ctx context.Context, pd pprofile.Profiles) (err error) {
		startTime := time.Now()
		samplesIn := pd.SampleCount()

		pd, err = profilesFunc(ctx, pd)
		obs.RecordInternalDuration(ctx, startTime)
		if err != nil {
			obs.RecordInOut(ctx, samplesIn, 0)
			if errors.Is(err, processorhelper.ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		samplesOut := pd.SampleCount()
		obs.RecordInOut(ctx, samplesIn, samplesOut)
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
