// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

func NewQueueSender(
	qSet queuebatch.QueueSettings[request.Request],
	qCfg exporterqueue.Config,
	bCfg exporterbatcher.Config,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()
		if errSend := next.Send(ctx, req); errSend != nil {
			qSet.ExporterSettings.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}
	return NewQueueBatch(qSet, qCfg, bCfg, exportFunc)
}

type QueueBatch struct {
	queue   queuebatch.Queue[request.Request]
	batcher batcher.Batcher[request.Request]
}

func NewQueueBatch(
	qSet queuebatch.QueueSettings[request.Request],
	qCfg exporterqueue.Config,
	bCfg exporterbatcher.Config,
	next sender.SendFunc[request.Request],
) (*QueueBatch, error) {
	b, err := batcher.NewBatcher(bCfg, next, qCfg.NumConsumers)
	if err != nil {
		return nil, err
	}
	// TODO: https://github.com/open-telemetry/opentelemetry-collector/issues/12244
	if bCfg.Enabled {
		qCfg.NumConsumers = 1
	}
	q, err := newObsQueue(qSet, queuebatch.NewQueue(context.Background(), qSet, qCfg, b.Consume))
	if err != nil {
		return nil, err
	}

	return &QueueBatch{queue: q, batcher: b}, nil
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
