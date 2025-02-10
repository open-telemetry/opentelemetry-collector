// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context" // Sender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).

	"go.opentelemetry.io/collector/component"
)

type Sender[K any] interface {
	component.Component
	Send(context.Context, K) error
}

type SendFunc[K any] func(context.Context, K) error

func (f SendFunc[K]) Send(ctx context.Context, k K) error {
	if f == nil {
		return nil
	}
	return f(ctx, k)
}
