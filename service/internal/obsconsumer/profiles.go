// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
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
		consumer:    consumer,
		itemCounter: itemCounter,
		options:     o,
	}
}

type Profiles struct {
	consumer    xconsumer.Profiles
	itemCounter metric.Int64Counter
	options
}

func (c Profiles) ConsumeProfiles(ctx context.Context, pd pprofile.Profiles) error {
	// Measure before calling ConsumeProfiles because the data may be mutated downstream
	itemCount := pd.SampleCount()

	err := c.consumer.ConsumeProfiles(ctx, pd)
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}

	var attrs []attribute.KeyValue
	attrs = append(attrs, c.staticDataPointAttributes...)
	attrs = append(attrs, attribute.String("outcome", outcome))
	c.itemCounter.Add(ctx, int64(itemCount), metric.WithAttributes(attrs...))
	return err
}

func (c Profiles) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
