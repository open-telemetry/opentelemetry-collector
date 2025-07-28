// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

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

func NewProfiles(cons xconsumer.Profiles, itemCounter metric.Int64Counter, sizeCounter metric.Int64Counter, opts ...Option) xconsumer.Profiles {
	if !telemetry.NewPipelineTelemetryGate.IsEnabled() {
		return cons
	}

	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	return obsProfiles{
		consumer:        cons,
		itemCounter:     itemCounter,
		sizeCounter:     sizeCounter,
		compiledOptions: o.compile(),
	}
}

type obsProfiles struct {
	consumer    xconsumer.Profiles
	itemCounter metric.Int64Counter
	sizeCounter metric.Int64Counter
	compiledOptions
}

// ConsumeProfiles measures telemetry before calling ConsumeProfiles because the data may be mutated downstream
func (c obsProfiles) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	// Measure before calling ConsumeProfiles because the data may be mutated downstream
	attrs := &c.withSuccessAttrs

	itemCount := pd.SampleCount()
	defer func() {
		c.itemCounter.Add(ctx, int64(itemCount), *attrs)
	}()

	if isEnabled(ctx, c.sizeCounter) {
		byteCount := int64(profilesMarshaler.ProfilesSize(pd))
		defer func() {
			c.sizeCounter.Add(ctx, byteCount, *attrs)
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
	}
	return err
}

func (c obsProfiles) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
