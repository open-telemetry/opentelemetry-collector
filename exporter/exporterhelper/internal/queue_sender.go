// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

// QueueBatchSettings is a subset of the queuebatch.Settings that are needed when used within an Exporter.
type QueueBatchSettings[K any] struct {
	Encoding exporterqueue.Encoding[K]
	Sizers   map[exporterbatcher.SizerType]queuebatch.Sizer[K]
}

func NewQueueSender(
	qSet queuebatch.Settings[request.Request],
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
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	return queuebatch.NewQueueBatch(qSet, qCfg, bCfg, exportFunc)
}
