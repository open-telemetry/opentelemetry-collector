// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher[T any] interface {
	component.Component
	Consume(context.Context, T, queue.Done)
}

type batcherSettings[T any] struct {
	itemsSizer  request.Sizer[T]
	bytesSizer  request.Sizer[T]
	partitioner Partitioner[T]
	next        sender.SendFunc[T]
	maxWorkers  int
	logger      *zap.Logger
}

func NewBatcher(cfg configoptional.Optional[BatchConfig], set batcherSettings[request.Request]) (Batcher[request.Request], error) {
	if !cfg.HasValue() {
		return newDisabledBatcher[request.Request](set.next), nil
	}

	sizer := activeSizer(cfg.Get().Sizer, set.itemsSizer, set.bytesSizer)
	if sizer == nil {
		return nil, fmt.Errorf("queue_batch: unsupported sizer %q", cfg.Get().Sizer)
	}

	if set.partitioner == nil {
		return newPartitionBatcher(*cfg.Get(), sizer, newWorkerPool(set.maxWorkers), set.next, set.logger), nil
	}

	return newMultiBatcher(*cfg.Get(), sizer, newWorkerPool(set.maxWorkers), set.partitioner, set.next, set.logger), nil
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
