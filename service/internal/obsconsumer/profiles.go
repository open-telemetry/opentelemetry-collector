// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var (
	_                 xconsumer.Profiles = profilesItemCounter{}
	_                 xconsumer.Profiles = profilesSizeCounter{}
	profilesMarshaler                    = pprofile.ProtoMarshaler{}
)

func WithProfilesItemCounter(itemCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.profilesItemCounter = itemCounter
	}
}

func WithProfilesSizeCounter(sizeCounter *metric.Int64Counter) Option {
	return func(opts *options) {
		opts.profilesSizeCounter = sizeCounter
	}
}

func NewProfiles(cons xconsumer.Profiles, opts ...Option) xconsumer.Profiles {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.profilesItemCounter == nil && o.profilesSizeCounter == nil {
		return cons
	}

	copts := o.compile()
	if o.profilesItemCounter != nil {
		cons = profilesItemCounter{
			consumer:        cons,
			itemCounter:     *o.profilesItemCounter,
			compiledOptions: copts,
		}
	}
	if o.profilesSizeCounter != nil {
		cons = profilesSizeCounter{
			consumer:        cons,
			sizeCounter:     *o.profilesSizeCounter,
			compiledOptions: copts,
		}
	}
	return cons
}

type profilesItemCounter struct {
	consumer    xconsumer.Profiles
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c profilesItemCounter) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	// Measure before calling ConsumeProfiles because the data may be mutated downstream
	itemCount := pd.SampleCount()
	err := c.consumer.ConsumeProfiles(ctx, pd)
	if err == nil {
		c.itemCounter.Add(ctx, int64(itemCount), c.withSuccessAttrs)
	} else {
		c.itemCounter.Add(ctx, int64(itemCount), c.withFailureAttrs)
	}
	return err
}

func (c profilesItemCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}

type profilesSizeCounter struct {
	consumer    xconsumer.Profiles
	sizeCounter metric.Int64Counter
	compiledOptions
}

func (c profilesSizeCounter) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	// Measure before calling ConsumeProfiles because the data may be mutated downstream
	byteCount := profilesMarshaler.ProfilesSize(pd)
	err := c.consumer.ConsumeProfiles(ctx, pd)
	if err == nil {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withSuccessAttrs)
	} else {
		c.sizeCounter.Add(ctx, int64(byteCount), c.withFailureAttrs)
	}
	return err
}

func (c profilesSizeCounter) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
