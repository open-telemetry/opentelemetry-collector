// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context" // Sender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/internal"
)

type Sender[K any] interface {
	component.Component
	Send(context.Context, K) error
	SetNextSender(nextSender Sender[K])
}

type BaseSender[K any] struct {
	component.StartFunc
	component.ShutdownFunc
	NextSender Sender[K]
}

var _ Sender[internal.Request] = (*BaseSender[internal.Request])(nil)

func (b *BaseSender[K]) Send(ctx context.Context, req K) error {
	return b.NextSender.Send(ctx, req)
}

func (b *BaseSender[K]) SetNextSender(nextSender Sender[K]) {
	b.NextSender = nextSender
}
