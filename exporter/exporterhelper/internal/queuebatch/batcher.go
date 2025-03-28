// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"

	"go.opentelemetry.io/collector/component"
)

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher[T any] interface {
	component.Component
	Consume(context.Context, T, Done)
}
