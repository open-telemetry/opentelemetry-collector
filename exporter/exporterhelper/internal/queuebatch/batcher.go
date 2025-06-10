// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher[T any] interface {
	component.Component
	Consume(context.Context, T, Done)
}

type batcherSettings[T any] struct {
	sizerType   request.SizerType
	sizer       request.Sizer[T]
	partitioner Partitioner[T]
	next        sender.SendFunc[T]
	maxWorkers  int
}

func NewBatcher(cfg *BatchConfig, set batcherSettings[request.Request]) Batcher[request.Request] {
	if cfg == nil {
		return newDisabledBatcher[request.Request](set.next)
	}

	if set.partitioner == nil {
		return newPartitionBatcher(*cfg, set.sizerType, set.sizer, newWorkerPool(set.maxWorkers), set.next)
	}

	return newMultiBatcher(*cfg, set.sizerType, set.sizer, newWorkerPool(set.maxWorkers), set.partitioner, set.next)
}
