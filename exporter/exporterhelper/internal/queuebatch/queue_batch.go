// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"
	"fmt"

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
	return newQueueBatch(set, cfg, next, false)
}

func NewQueueBatchLegacyBatcher(
	set Settings[request.Request],
	cfg Config,
	next sender.SendFunc[request.Request],
) (*QueueBatch, error) {
	set.Telemetry.Logger.Warn("Configuring the exporter batcher capability separately is now deprecated. " +
		"Use sending_queue::batch instead.")
	return newQueueBatch(set, cfg, next, true)
}

func newQueueBatch(
	set Settings[request.Request],
	cfg Config,
	next sender.SendFunc[request.Request],
	oldBatcher bool,
) (*QueueBatch, error) {
	sizer := activeSizer(cfg.Sizer, set.ItemsSizer, set.BytesSizer)
	if sizer == nil {
		return nil, fmt.Errorf("queue_batch: unsupported sizer %q", cfg.Sizer)
	}

	bSet := batcherSettings[request.Request]{
		sizerType:   cfg.Sizer,
		sizer:       sizer,
		partitioner: set.Partitioner,
		next:        next,
		maxWorkers:  cfg.NumConsumers,
	}
	if oldBatcher {
		// If a user configures the old batcher, we only can support "items" sizer.
		bSet.sizerType = request.SizerTypeItems
		bSet.sizer = request.NewItemsSizer()
	}
	b := NewBatcher(cfg.Batch, bSet)
	if cfg.Batch != nil {
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

	oq, err := newObsQueue(set, q)
	if err != nil {
		return nil, err
	}

	return &QueueBatch{queue: oq, batcher: b}, nil
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

func activeSizer[T any](sizerType request.SizerType, itemsSizer, bytesSizer request.Sizer[T]) request.Sizer[T] {
	switch sizerType {
	case request.SizerTypeBytes:
		return bytesSizer
	case request.SizerTypeItems:
		return itemsSizer
	default:
		return request.RequestsSizer[T]{}
	}
}
