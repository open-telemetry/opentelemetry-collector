// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher[K any] interface {
	component.Component
	Consume(context.Context, K, queuebatch.Done)
}

func NewBatcher(batchCfg exporterbatcher.Config, exportFunc sender.SendFunc[request.Request], maxWorkers int) (Batcher[request.Request], error) {
	if !batchCfg.Enabled {
		return newDisabledBatcher[request.Request](exportFunc), nil
	}
	return newDefaultBatcher(batchCfg, exportFunc, maxWorkers), nil
}
