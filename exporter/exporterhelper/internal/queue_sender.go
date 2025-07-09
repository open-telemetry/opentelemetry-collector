// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// QueueBatchSettings is a subset of the queuebatch.Settings that are needed when used within an Exporter.
type QueueBatchSettings[T any] struct {
	Encoding    queue.Encoding[T]
	ItemsSizer  request.Sizer[T]
	BytesSizer  request.Sizer[T]
	Partitioner queuebatch.Partitioner[T]
}

// NewDefaultQueueConfig returns the default config for queuebatch.Config.
// By default, the queue stores 1000 requests of telemetry and is non-blocking when full.
func NewDefaultQueueConfig() queuebatch.Config {
	return queuebatch.Config{
		Enabled:      true,
		Sizer:        request.SizerTypeRequests,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize:       1_000,
		BlockOnOverflow: false,
		StorageID:       nil,
		Batch:           nil,
	}
}

func NewQueueSender(
	qSet queuebatch.Settings[request.Request],
	qCfg queuebatch.Config,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()
		if errSend := next.Send(ctx, req); errSend != nil {
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	return queuebatch.NewQueueBatch(qSet, qCfg, exportFunc)
}
