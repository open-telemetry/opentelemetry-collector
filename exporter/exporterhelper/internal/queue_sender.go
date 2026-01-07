// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
	"go.opentelemetry.io/collector/pipeline"
)

// NewDefaultQueueConfig returns the default config for queuebatch.Config.
// By default:
//
// - the queue stores 1000 requests of telemetry
// - is non-blocking when full
// - concurrent exports limited to 10
// - emits batches of 8192 items, timeout 200ms
func NewDefaultQueueConfig() queuebatch.Config {
	return queuebatch.Config{
		Sizer:           request.SizerTypeRequests,
		NumConsumers:    10,
		QueueSize:       1_000,
		BlockOnOverflow: false,
		Batch: configoptional.Default(queuebatch.BatchConfig{
			FlushTimeout: 200 * time.Millisecond,
			Sizer:        request.SizerTypeItems,
			MinSize:      8192,
		}),
	}
}

// queueSender wraps QueueBatch to add RequestMiddleware logic.
type queueSender struct {
	*queuebatch.QueueBatch
	mwID *component.ID
	mw   extensioncapabilities.RequestMiddleware

	// Capture these fields during creation so we can use them in Start()
	// because QueueBatch does not expose them publicly.
	id     component.ID
	signal pipeline.Signal
}

func NewQueueSender(
	qSet queuebatch.AllSettings[request.Request],
	qCfg queuebatch.Config,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	if qCfg.RequestMiddlewareID == nil {
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

	warnIfNumConsumersMayCapMiddleware(&qCfg, qSet.Telemetry.Logger)

	qs := &queueSender{
		mwID:   qCfg.RequestMiddlewareID,
		id:     qSet.ID,
		signal: qSet.Signal,
		mw:     extensioncapabilities.NoopRequestMiddleware(), // Initialize with no-op to avoid nil checks
	}

	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()

		// Wrap the actual export call in a closure for the middleware
		nextCall := func(ctx context.Context) error {
			if errSend := next.Send(ctx, req); errSend != nil {
				qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
					zap.Error(errSend), zap.Int("dropped_items", itemsCount))
				return errSend
			}
			return nil
		}

		// Use the middleware to execute the request
		return qs.mw.Handle(ctx, nextCall)
	}

	qb, err := queuebatch.NewQueueBatch(qSet, qCfg, exportFunc)
	if err != nil {
		return nil, err
	}
	qs.QueueBatch = qb

	return qs, nil
}

// warnIfNumConsumersMayCapMiddleware ensures sufficient worker headroom when a concurrency controller is configured.
func warnIfNumConsumersMayCapMiddleware(cfg *queuebatch.Config, logger *zap.Logger) {
	if cfg.RequestMiddlewareID == nil {
		return
	}

	defaultConsumers := NewDefaultQueueConfig().NumConsumers

	// If the user left the default worker count, warn that the worker pool might artificially cap the controller.
	if cfg.NumConsumers == defaultConsumers && logger != nil {
		logger.Warn("sending_queue.num_consumers is at the default; request middleware may be capped by worker pool",
			zap.Int("num_consumers", cfg.NumConsumers),
			zap.String("concurrency_controller", cfg.RequestMiddlewareID.String()),
		)
	}
}

// Start overrides the default Start to resolve the extension.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	if qs.mwID != nil {
		ext, ok := host.GetExtensions()[*qs.mwID]
		if !ok {
			return fmt.Errorf("request middleware extension %q not found", qs.mwID.String())
		}
		factory, ok := ext.(extensioncapabilities.RequestMiddlewareFactory)
		if !ok {
			return fmt.Errorf("extension %q does not implement RequestMiddlewareFactory", qs.mwID.String())
		}
		mw, err := factory.CreateRequestMiddleware(qs.id, qs.signal)
		if err != nil {
			return fmt.Errorf("failed to create request middleware: %w", err)
		}

		// Guard against nil return from factory to avoid panic in Handle()
		if mw != nil {
			qs.mw = mw
		}
	}

	if err := qs.QueueBatch.Start(ctx, host); err != nil {
		if qs.mw != nil {
			_ = qs.mw.Shutdown(ctx)
			// Reset to no-op on failure cleanup
			qs.mw = extensioncapabilities.NoopRequestMiddleware()
		}
		return err
	}
	return nil
}

// Shutdown ensures the middleware is also shut down.
func (qs *queueSender) Shutdown(ctx context.Context) error {
	errQ := qs.QueueBatch.Shutdown(ctx)

	if qs.mw != nil {
		if err := qs.mw.Shutdown(ctx); err != nil && errQ == nil {
			return err
		}
		// Reset to no-op after shutdown
		qs.mw = extensioncapabilities.NoopRequestMiddleware()
	}
	return errQ
}
