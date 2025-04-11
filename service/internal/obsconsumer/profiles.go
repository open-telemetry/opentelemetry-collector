// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var _ xconsumer.Profiles = Profiles{}

func NewProfiles(consumer xconsumer.Profiles, itemCounter metric.Int64Counter, opts ...Option) Profiles {
	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}
	return Profiles{
		consumer:        consumer,
		itemCounter:     itemCounter,
		compiledOptions: o.compile(),
	}
}

type Profiles struct {
	consumer    xconsumer.Profiles
	itemCounter metric.Int64Counter
	compiledOptions
}

func (c Profiles) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
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

func (c Profiles) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
