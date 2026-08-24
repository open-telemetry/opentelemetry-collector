// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// NewDefaultQueueConfig returns the default config for queuebatch.Config.
// By default:
//
// - the queue stores 1000 requests of telemetry
// - is non-blocking when full
// - concurrent exports limited to 10
// - emits batches of 8192 items, timeout 200ms
//
// Batching is disabled by default, unless the
// pkg.exporterhelper.queueBatchEnabled feature gate is enabled. See
// https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/batching-migration.md.
func NewDefaultQueueConfig() queuebatch.Config {
	batchCfg := queuebatch.BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      8192,
	}
	var batch configoptional.Optional[queuebatch.BatchConfig]
	if metadata.PkgExporterhelperQueueBatchEnabledFeatureGate.IsEnabled() {
		batch = configoptional.Some(batchCfg)
	} else {
		batch = configoptional.Default(batchCfg)
	}
	return queuebatch.Config{
		Sizer:           request.SizerTypeRequests,
		NumConsumers:    10,
		QueueSize:       1_000,
		BlockOnOverflow: false,
		Batch:           batch,
	}
}

func NewQueueSender(
	qSet queuebatch.AllSettings[request.Request],
	qCfg queuebatch.Config,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()
		if errSend := next.Send(ctx, req); errSend != nil {
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	return queuebatch.NewQueueBatch(qSet, qCfg, exportFunc)
}
