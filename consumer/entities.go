// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"context"

	"go.opentelemetry.io/collector/consumer/internal"
	"go.opentelemetry.io/collector/pdata/pentity"
)

// Entities is an interface that receives pentity.Entities, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Entities interface {
	internal.BaseConsumer
	// ConsumeEntities receives pentity.Entities for consumption.
	ConsumeEntities(ctx context.Context, td pentity.Entities) error
}

// ConsumeEntitiesFunc is a helper function that is similar to ConsumeEntities.
type ConsumeEntitiesFunc func(ctx context.Context, td pentity.Entities) error

// ConsumeEntities calls f(ctx, td).
func (f ConsumeEntitiesFunc) ConsumeEntities(ctx context.Context, td pentity.Entities) error {
	return f(ctx, td)
}

type baseEntities struct {
	*internal.BaseImpl
	ConsumeEntitiesFunc
}

// NewEntities returns a Entities configured with the provided options.
func NewEntities(consume ConsumeEntitiesFunc, options ...Option) (Entities, error) {
	if consume == nil {
		return nil, errNilFunc
	}
	return &baseEntities{
		BaseImpl:            internal.NewBaseImpl(options...),
		ConsumeEntitiesFunc: consume,
	}, nil
}
