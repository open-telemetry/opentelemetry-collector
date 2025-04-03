// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sendertest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sendertest"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

func NewNopSenderFunc[T any]() sender.SendFunc[T] {
	return func(context.Context, T) error {
		return nil
	}
}

func NewErrSenderFunc[T any](err error) sender.SendFunc[T] {
	return func(context.Context, T) error {
		return err
	}
}
