// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
)

// Batcher is in charge of reading items from the queue and send them out asynchronously.
type Batcher interface {
	component.Component
	Consume(context.Context, internal.Request, exporterqueue.DoneCallback)
}

func NewBatcher(batchCfg exporterbatcher.Config,
	exportFunc func(ctx context.Context, req internal.Request) error,
	maxWorkers int,
) (Batcher, error) {
	if !batchCfg.Enabled {
		return newDisabledBatcher(exportFunc), nil
	}
	return newDefaultBatcher(batchCfg, exportFunc, maxWorkers), nil
}
