// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
)

// WithQueue overrides the default QueueBatchConfig for an exporter.
// The default QueueBatchConfig is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(config configoptional.Optional[QueueBatchConfig]) Option {
	return internal.WithQueue(config)
}

// QueueBatchConfig defines configuration for queueing and batching for the exporter.
type QueueBatchConfig = queuebatch.Config

// BatchConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
type BatchConfig = queuebatch.BatchConfig

// QueueBatchEncoding defines the encoding to be used if persistent queue is configured.
// Duplicate definition with queuebatch.Encoding since aliasing generics is not supported by default.
type QueueBatchEncoding[T any] interface {
	// Marshal is a function that can marshal a request and its context into bytes.
	Marshal(context.Context, T) ([]byte, error)

	// Unmarshal is a function that can unmarshal bytes into a request and its context.
	Unmarshal([]byte) (context.Context, T, error)
}

var ErrQueueIsFull = queue.ErrQueueIsFull

// NewDefaultQueueConfig returns the default config for QueueBatchConfig.
// By default, the queue stores 1000 requests of telemetry and is non-blocking when full.
var NewDefaultQueueConfig = internal.NewDefaultQueueConfig
