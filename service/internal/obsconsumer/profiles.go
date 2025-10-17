// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var (
	_                 xconsumer.Profiles = obsProfiles{}
	profilesMarshaler                    = pprofile.ProtoMarshaler{}
)

func NewProfiles(cons xconsumer.Profiles, set Settings, opts ...Option) xconsumer.Profiles {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	consumerSet := Settings{
		ItemCounter: set.ItemCounter,
		SizeCounter: set.SizeCounter,
		Logger:      set.Logger.With(telemetry.ToZapFields(o.staticDataPointAttributes)...),
	}

	return obsProfiles{
		consumer:        cons,
		set:             consumerSet,
		compiledOptions: o.compile(),
	}
}

type obsProfiles struct {
	consumer xconsumer.Profiles
	set      Settings
	compiledOptions
}

// ConsumeProfiles measures telemetry before calling ConsumeProfiles because the data may be mutated downstream
func (c obsProfiles) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	// Measure before calling ConsumeProfiles because the data may be mutated downstream
	attrs := &c.withSuccessAttrs

	itemCount := pd.SampleCount()
	defer func() {
		c.set.ItemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.set.SizeCounter) {
		byteCount := int64(profilesMarshaler.ProfilesSize(pd))
		defer func() {
			c.set.SizeCounter.Add(ctx, byteCount, *attrs)
		}()
	}

	err := c.consumer.ConsumeProfiles(ctx, pd)
	if err != nil {
		if consumererror.IsDownstream(err) {
			attrs = &c.withRefusedAttrs
		} else {
			attrs = &c.withFailureAttrs
			err = consumererror.NewDownstream(err)
		}
		if c.set.Logger.Core().Enabled(zap.DebugLevel) {
			c.set.Logger.Debug("Profiles pipeline component had an error", zap.Error(err), zap.Int("item count", itemCount))
		}
	}
	return err
}

func (c obsProfiles) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
