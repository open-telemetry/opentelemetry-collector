// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context" // RequestSender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/internal"
)

type RequestSender interface {
	component.Component
	Send(context.Context, internal.Request) error
	SetNextSender(nextSender RequestSender)
}

type BaseRequestSender struct {
	component.StartFunc
	component.ShutdownFunc
	NextSender RequestSender
}

var _ RequestSender = (*BaseRequestSender)(nil)

func (b *BaseRequestSender) Send(ctx context.Context, req internal.Request) error {
	return b.NextSender.Send(ctx, req)
}

func (b *BaseRequestSender) SetNextSender(nextSender RequestSender) {
	b.NextSender = nextSender
}
