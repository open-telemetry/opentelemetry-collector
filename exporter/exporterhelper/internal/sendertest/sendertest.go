// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sendertest // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sendertest"

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

func NewNopSenderFunc[K any]() sender.SendFunc[K] {
	return func(context.Context, K) error {
		return nil
	}
}

func NewErrSenderFunc[K any](err error) sender.SendFunc[K] {
	return func(context.Context, K) error {
		return err
	}
}
