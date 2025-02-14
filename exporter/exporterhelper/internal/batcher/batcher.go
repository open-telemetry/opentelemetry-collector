// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batcher // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher interface {
	component.Component
	Consume(context.Context, request.Request, exporterqueue.Done)
}

func NewBatcher(batchCfg exporterbatcher.Config,
	exportFunc func(ctx context.Context, req request.Request) error,
	maxWorkers int,
) (Batcher, error) {
	if !batchCfg.Enabled {
		return newDisabledBatcher(exportFunc), nil
	}
	return newDefaultBatcher(batchCfg, exportFunc, maxWorkers), nil
}
