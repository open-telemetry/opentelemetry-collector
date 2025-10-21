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
	partitioner Partitioner[T]
	mergeCtx    func(context.Context, context.Context) context.Context
	next        sender.SendFunc[T]
	maxWorkers  int
	logger      *zap.Logger
}

func NewBatcher(cfg configoptional.Optional[BatchConfig], set batcherSettings[request.Request]) (Batcher[request.Request], error) {
	if !cfg.HasValue() {
		return newDisabledBatcher(set.next), nil
	}

	sizer := request.NewSizer(cfg.Get().Sizer)
	if sizer == nil {
		return nil, fmt.Errorf("queue_batch: unsupported sizer %q", cfg.Get().Sizer)
	}

	if set.partitioner == nil {
		return newPartitionBatcher(*cfg.Get(), sizer, set.mergeCtx, newWorkerPool(set.maxWorkers), set.next, set.logger), nil
	}

	return newMultiBatcher(*cfg.Get(), sizer, newWorkerPool(set.maxWorkers), set.partitioner, set.mergeCtx, set.next, set.logger), nil
}
