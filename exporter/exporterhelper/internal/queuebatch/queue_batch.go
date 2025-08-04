// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
)

// Settings defines settings for creating a QueueBatch.
type Settings[T any] struct {
	Signal      pipeline.Signal
	ID          component.ID
	Telemetry   component.TelemetrySettings
	Encoding    queue.Encoding[T]
	ItemsSizer  request.Sizer[T]
	BytesSizer  request.Sizer[T]
	Partitioner Partitioner[T]
}

type QueueBatch struct {
	queue   queue.Queue[request.Request]
	batcher Batcher[request.Request]
}

func NewQueueBatch(
	set Settings[request.Request],
	cfg Config,
	next sender.SendFunc[request.Request],
) (*QueueBatch, error) {
	b, err := NewBatcher(cfg.Batch, batcherSettings[request.Request]{
		itemsSizer:  set.ItemsSizer,
		bytesSizer:  set.BytesSizer,
		partitioner: set.Partitioner,
		next:        next,
		maxWorkers:  cfg.NumConsumers,
		logger:      set.Telemetry.Logger,
	})
	if err != nil {
		return nil, err
	}
	if cfg.Batch.HasValue() {
		// If batching is enabled, keep the number of queue consumers to 1 if batching is enabled until we support
		// sharding as described in https://github.com/open-telemetry/opentelemetry-collector/issues/12473
		cfg.NumConsumers = 1
	}

	q, err := queue.NewQueue[request.Request](queue.Settings[request.Request]{
		SizerType:       cfg.Sizer,
		ItemsSizer:      set.ItemsSizer,
		BytesSizer:      set.BytesSizer,
		Capacity:        cfg.QueueSize,
		NumConsumers:    cfg.NumConsumers,
		WaitForResult:   cfg.WaitForResult,
		BlockOnOverflow: cfg.BlockOnOverflow,
		Signal:          set.Signal,
		StorageID:       cfg.StorageID,
		Encoding:        set.Encoding,
		ID:              set.ID,
		Telemetry:       set.Telemetry,
	}, b.Consume)
	if err != nil {
		return nil, err
	}

	return &QueueBatch{queue: q, batcher: b}, nil
}

// Start is invoked during service startup.
func (qs *QueueBatch) Start(ctx context.Context, host component.Host) error {
	if err := qs.batcher.Start(ctx, host); err != nil {
		return err
	}
	if err := qs.queue.Start(ctx, host); err != nil {
		return errors.Join(err, qs.batcher.Shutdown(ctx))
	}
	return nil
}

// Shutdown is invoked during service shutdown.
func (qs *QueueBatch) Shutdown(ctx context.Context) error {
	// Stop the queue and batcher, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	return errors.Join(qs.queue.Shutdown(ctx), qs.batcher.Shutdown(ctx))
}

// Send implements the requestSender interface. It puts the request in the queue.
func (qs *QueueBatch) Send(ctx context.Context, req request.Request) error {
	return qs.queue.Offer(ctx, req)
}
