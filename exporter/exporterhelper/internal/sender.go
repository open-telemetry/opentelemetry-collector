// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

type Sender[K any] interface {
	component.Component
	Send(context.Context, K) error
}

type SendFunc[K any] func(ctx context.Context, data K) error

// sender is a Sender that emits the incoming request to the exporter consumer func.
type sender[K any] struct {
	component.StartFunc
	component.ShutdownFunc
	consFunc SendFunc[K]
}

func newSender[K any](consFunc SendFunc[K]) Sender[K] {
	return &sender[K]{consFunc: consFunc}
}

func (es *sender[K]) Send(ctx context.Context, req K) error {
	return es.consFunc(ctx, req)
}
