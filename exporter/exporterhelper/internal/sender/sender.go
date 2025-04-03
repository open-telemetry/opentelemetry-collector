// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sender // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

type Sender[T any] interface {
	component.Component
	Send(context.Context, T) error
}

type SendFunc[T any] func(ctx context.Context, data T) error

func NewSender[T any](consFunc SendFunc[T]) Sender[T] {
	return &sender[T]{consFunc: consFunc}
}

// sender is a Sender that emits the incoming request to the exporter consumer func.
type sender[T any] struct {
	component.StartFunc
	component.ShutdownFunc
	consFunc SendFunc[T]
}

func (es *sender[T]) Send(ctx context.Context, req T) error {
	return es.consFunc(ctx, req)
}
