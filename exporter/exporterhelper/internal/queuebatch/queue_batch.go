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

// Settings is a subset of the queuebatch.Settings that are needed when used within an Exporter.
type Settings[T any] struct {
	ReferenceCounter queue.ReferenceCounter[T]
	Encoding         queue.Encoding[T]
	Partitioner      Partitioner[T]
	MergeCtx         func(context.Context, context.Context) context.Context
	// ReplayInOrder controls persisted dequeue order on restart. In-flight
	// items retain their original indices and precede newer queued work.
	ReplayInOrder bool
}

// AllSettings defines settings for creating a QueueBatch.
type AllSettings[T any] struct {
	Settings[T]
	Signal    pipeline.Signal
	ID        component.ID
	Telemetry component.TelemetrySettings
}

type QueueBatch struct {
	queue   queue.Queue[request.Request]
	batcher Batcher[request.Request]
}

func NewQueueBatch(
	set AllSettings[request.Request],
	cfg Config,
	next sender.SendFunc[request.Request],
) (*QueueBatch, error) {
	b, err := NewBatcher(cfg.Batch, batcherSettings[request.Request]{
		partitioner: set.Partitioner,
		mergeCtx:    set.MergeCtx,
		next:        next,
		maxWorkers:  cfg.NumConsumers,
		id:          set.ID,
		signal:      set.Signal,
		telemetry:   set.Telemetry,
		logger:      set.Telemetry.Logger,
	})
	if err != nil {
		return nil, err
	}
	if cfg.Batch.HasValue() && set.Partitioner == nil {
		// If batching is enabled and partitioner is not defined then keep the number of queue consumers to 1.
		// see: https://github.com/open-telemetry/opentelemetry-collector/issues/12473
		cfg.NumConsumers = 1
	}

	q, err := queue.NewQueue(queue.Settings[request.Request]{
		SizerType:        cfg.Sizer,
		Capacity:         cfg.QueueSize,
		NumConsumers:     cfg.NumConsumers,
		ReplayInOrder:    set.ReplayInOrder,
		WaitForResult:    cfg.WaitForResult,
		BlockOnOverflow:  cfg.BlockOnOverflow,
		Signal:           set.Signal,
		StorageID:        cfg.StorageID,
		ReferenceCounter: set.ReferenceCounter,
		Encoding:         set.Encoding,
		ID:               set.ID,
		Telemetry:        set.Telemetry,
	}, b.Consume)
	if err != nil {
		return nil, err
	}

	return &QueueBatch{queue: q, batcher: b}, nil
}

// NewAsyncQueueBatch builds the normal bounded queue without batching and
// dispatches requests through an asynchronous completion callback.
func NewAsyncQueueBatch(
	set AllSettings[request.Request],
	cfg Config,
	next sender.SendFunc[request.Request],
) (*QueueBatch, error) {
	if next == nil {
		return nil, errors.New("async queue batch: nil send function")
	}
	if cfg.Batch.HasValue() {
		return nil, errors.New("async queue batch: batching is not supported for ordered stream requests")
	}

	// Keep one FIFO queue reader. Ordered stream scheduling owns the configured
	// cross-partition write concurrency after the reader has staged each item.
	cfg.NumConsumers = 1
	var checkpointStore request.QueueCheckpointStore
	q, err := queue.NewQueue(queue.Settings[request.Request]{
		SizerType:         cfg.Sizer,
		Capacity:          cfg.QueueSize,
		NumConsumers:      cfg.NumConsumers,
		WaitForCompletion: true,
		ReplayInOrder:     true,
		WaitForResult:     cfg.WaitForResult,
		BlockOnOverflow:   cfg.BlockOnOverflow,
		Signal:            set.Signal,
		StorageID:         cfg.StorageID,
		ReferenceCounter:  set.ReferenceCounter,
		Encoding:          set.Encoding,
		ID:                set.ID,
		Telemetry:         set.Telemetry,
	}, func(ctx context.Context, req request.Request, done queue.Done) {
		if requestWithCheckpoint, ok := req.(request.QueueCheckpointStoreSetter); ok {
			requestWithCheckpoint.SetQueueCheckpointStore(checkpointStore)
		}
		deferred, ok := req.(request.DeferredQueueCompletion)
		if !ok {
			done.OnDone(errors.New("async queue requires a deferred-completion request"))
			return
		}
		if !deferred.SetQueueCompletion(done.OnDone) {
			done.OnDone(errors.New("async queue request completion is already registered"))
			return
		}
		// Stage requests synchronously on the FIFO reader. The coordinator
		// starts partition writes asynchronously; launching next in a goroutine
		// here would let later queue items race ahead of this request.
		if err := next(ctx, req); err != nil {
			done.OnDone(err)
		}
	})
	if err != nil {
		return nil, err
	}
	// Queue wrappers expose checkpoint methods even around a memory queue.
	// Only attach a store when persistence was configured; otherwise each ACK
	// serializes every open partition's tails for a save that is a no-op.
	if cfg.StorageID != nil {
		checkpointStore, _ = q.(request.QueueCheckpointStore)
	}
	return &QueueBatch{queue: q, batcher: &asyncBatcher{}}, nil
}

type asyncBatcher struct {
	component.StartFunc
	component.ShutdownFunc
}

func (*asyncBatcher) Consume(context.Context, request.Request, queue.Done) {}

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
