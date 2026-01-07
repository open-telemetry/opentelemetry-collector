// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"

	// Import the new package (aliased to avoid confusion with existing extensionmiddleware)
	requestmiddleware "go.opentelemetry.io/collector/extension/xextension/extensionmiddleware"
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
	mwIDs []component.ID
	mws   []requestmiddleware.RequestMiddleware

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
	if len(qCfg.RequestMiddlewares) == 0 {
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
		mwIDs:  qCfg.RequestMiddlewares,
		id:     qSet.ID,
		signal: qSet.Signal,
		mws:    make([]requestmiddleware.RequestMiddleware, 0, len(qCfg.RequestMiddlewares)),
	}

	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()

		// 1. Define the base exporter call
		baseSender := func(ctx context.Context) error {
			if errSend := next.Send(ctx, req); errSend != nil {
				qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
					zap.Error(errSend), zap.Int("dropped_items", itemsCount))
				return errSend
			}
			return nil
		}

		// 2. Chain middlewares: m1(m2(base))
		// We iterate in reverse so that the first middleware in the config is the outer-most wrapper.
		nextCall := baseSender
		for i := len(qs.mws) - 1; i >= 0; i-- {
			mw := qs.mws[i]
			currNext := nextCall
			nextCall = func(ctx context.Context) error {
				return mw.Handle(ctx, currNext)
			}
		}

		// 3. Execute the chain
		return nextCall(ctx)
	}

	qb, err := queuebatch.NewQueueBatch(qSet, qCfg, exportFunc)
	if err != nil {
		return nil, err
	}
	qs.QueueBatch = qb

	return qs, nil
}

// warnIfNumConsumersMayCapMiddleware ensures sufficient worker headroom when request middleware is configured.
func warnIfNumConsumersMayCapMiddleware(cfg *queuebatch.Config, logger *zap.Logger) {
	if len(cfg.RequestMiddlewares) == 0 {
		return
	}

	defaultConsumers := NewDefaultQueueConfig().NumConsumers

	// If the user left the default worker count, warn that the worker pool might artificially cap the controller.
	if cfg.NumConsumers == defaultConsumers && logger != nil {
		logger.Warn("sending_queue.num_consumers is at the default; request middleware may be capped by worker pool",
			zap.Int("num_consumers", cfg.NumConsumers),
			zap.Strings("request_middlewares", func() []string {
				ids := make([]string, len(cfg.RequestMiddlewares))
				for i, id := range cfg.RequestMiddlewares {
					ids[i] = id.String()
				}
				return ids
			}()),
		)
	}
}

// Start overrides the default Start to resolve the extensions.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	for _, id := range qs.mwIDs {
		ext, ok := host.GetExtensions()[id]
		if !ok {
			// Cleanup any middlewares successfully created before this failure
			_ = qs.Shutdown(ctx)
			return fmt.Errorf("request middleware extension %q not found", id.String())
		}
		factory, ok := ext.(requestmiddleware.RequestMiddlewareFactory)
		if !ok {
			// Cleanup any middlewares successfully created before this failure
			_ = qs.Shutdown(ctx)
			return fmt.Errorf("extension %q does not implement RequestMiddlewareFactory", id.String())
		}
		mw, err := factory.CreateRequestMiddleware(qs.id, qs.signal)
		if err != nil {
			// Cleanup any middlewares successfully created before this failure
			_ = qs.Shutdown(ctx)
			return fmt.Errorf("failed to create request middleware for %q: %w", id.String(), err)
		}

		if mw != nil {
			qs.mws = append(qs.mws, mw)
		}
	}

	if err := qs.QueueBatch.Start(ctx, host); err != nil {
		// Ensure middlewares are shut down if queue fails to start.
		_ = qs.Shutdown(ctx)
		return err
	}
	return nil
}

// Shutdown ensures the middlewares are also shut down.
func (qs *queueSender) Shutdown(ctx context.Context) error {
	var errs error
	if qs.QueueBatch != nil {
		errs = multierr.Append(errs, qs.QueueBatch.Shutdown(ctx))
	}

	// Shutdown middlewares in reverse order (LIFO)
	for i := len(qs.mws) - 1; i >= 0; i-- {
		errs = multierr.Append(errs, qs.mws[i].Shutdown(ctx))
	}
	// Clear middlewares after shutdown
	qs.mws = nil

	return errs
}
