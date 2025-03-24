// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/pipeline"
)

// Settings defines settings for creating a QueueBatch.
type Settings[K any] struct {
	Signal    pipeline.Signal
	ID        component.ID
	Telemetry component.TelemetrySettings
	Encoding  exporterqueue.Encoding[K]
	Sizers    map[exporterbatcher.SizerType]Sizer[K]
}

type QueueBatch struct {
	queue   Queue[request.Request]
	batcher Batcher[request.Request]
}

func NewQueueBatch(
	qSet Settings[request.Request],
	qCfg exporterqueue.Config,
	bCfg exporterbatcher.Config,
	next sender.SendFunc[request.Request],
) (*QueueBatch, error) {
	var b Batcher[request.Request]
	switch bCfg.Enabled {
	case false:
		b = newDisabledBatcher[request.Request](next)
	default:
		b = newDefaultBatcher(bCfg, next, qCfg.NumConsumers)
	}
	// TODO: https://github.com/open-telemetry/opentelemetry-collector/issues/12244
	if bCfg.Enabled {
		qCfg.NumConsumers = 1
	}

	sizer, ok := qSet.Sizers[exporterbatcher.SizerTypeRequests]
	if !ok {
		return nil, errors.New("queue_batch: unsupported sizer")
	}

	var q Queue[request.Request]
	switch {
	case !qCfg.Enabled:
		q = newDisabledQueue(b.Consume)
	case qCfg.StorageID != nil:
		q = newAsyncQueue(newPersistentQueue[request.Request](persistentQueueSettings[request.Request]{
			sizer:     sizer,
			capacity:  int64(qCfg.QueueSize),
			blocking:  qCfg.Blocking,
			signal:    qSet.Signal,
			storageID: *qCfg.StorageID,
			encoding:  qSet.Encoding,
			id:        qSet.ID,
			telemetry: qSet.Telemetry,
		}), qCfg.NumConsumers, b.Consume)
	default:
		q = newAsyncQueue(newMemoryQueue[request.Request](memoryQueueSettings[request.Request]{
			sizer:    sizer,
			capacity: int64(qCfg.QueueSize),
			blocking: qCfg.Blocking,
		}), qCfg.NumConsumers, b.Consume)
	}

	oq, err := newObsQueue(qSet, q)
	if err != nil {
		return nil, err
	}

	return &QueueBatch{queue: oq, batcher: b}, nil
}

// Start is invoked during service startup.
func (qs *QueueBatch) Start(ctx context.Context, host component.Host) error {
	if err := qs.queue.Start(ctx, host); err != nil {
		return err
	}

	return qs.batcher.Start(ctx, host)
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
