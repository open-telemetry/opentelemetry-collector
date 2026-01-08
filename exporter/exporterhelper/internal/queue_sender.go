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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/pipeline"
)

// NewDefaultQueueConfig returns the default config for queuebatch.Config.
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

	// next is the sender that the queue consumers will call.
	// It is initialized to the downstream sender, but can be wrapped by middlewares in Start.
	next sender.Sender[request.Request]

	// activeMiddlewares tracks the wrappers created during Start so they can be shutdown.
	activeMiddlewares []component.Component

	// Cached exporter settings for Start().
	id        component.ID
	signal    pipeline.Signal
	telemetry component.TelemetrySettings
}

func NewQueueSender(
	qSet queuebatch.AllSettings[request.Request],
	qCfg queuebatch.Config,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {

	qs := &queueSender{
		mwIDs:     qCfg.RequestMiddlewares,
		id:        qSet.ID,
		signal:    qSet.Signal,
		telemetry: qSet.Telemetry,
		next:      next,
	}

	// exportFunc is called by the queue consumers.
	// It delegates to qs.next, which allows us to swap qs.next in Start() without recreating the queue.
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()

		if errSend := qs.next.Send(ctx, req); errSend != nil {
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	qb, err := queuebatch.NewQueueBatch(qSet, qCfg, exportFunc)
	if err != nil {
		return nil, err
	}
	qs.QueueBatch = qb

	if len(qCfg.RequestMiddlewares) > 0 {
		warnIfNumConsumersMayCapMiddleware(&qCfg, qSet.Telemetry.Logger)
	}

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

// Start overrides the default Start to resolve the extensions and wrap the sender.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	mws := make([]requestmiddleware.RequestMiddleware, 0, len(qs.mwIDs))

	// 1. Resolve all extensions first to ensure they exist and implement the interface
	for _, id := range qs.mwIDs {
		ext, ok := host.GetExtensions()[id]
		if !ok {
			return fmt.Errorf("request middleware extension %q not found", id.String())
		}
		mw, ok := ext.(requestmiddleware.RequestMiddleware)
		if !ok {
			return fmt.Errorf("extension %q does not implement RequestMiddleware", id.String())
		}
		mws = append(mws, mw)
	}

	settings := requestmiddleware.RequestMiddlewareSettings{
		ID:        qs.id,
		Signal:    qs.signal,
		Telemetry: qs.telemetry,
	}

	// 2. Wrap the sender.
	// Wrap sender in reverse so first-configured middleware is outermost.
	wrapped := qs.next
	for i := len(mws) - 1; i >= 0; i-- {
		var err error
		wrapped, err = mws[i].WrapSender(settings, wrapped)
		if err != nil {
			_ = qs.Shutdown(ctx) // Clean up any previously started middlewares
			return fmt.Errorf("failed to wrap sender for %q: %w", qs.mwIDs[i].String(), err)
		}
		if wrapped == nil {
			_ = qs.Shutdown(ctx)
			return fmt.Errorf("request middleware %q returned nil sender", qs.mwIDs[i].String())
		}

		// If the wrapper is a component, start it.
		if err := wrapped.Start(ctx, host); err != nil {
			_ = qs.Shutdown(ctx)
			return err
		}
		qs.activeMiddlewares = append(qs.activeMiddlewares, wrapped)
	}

	// Update the next sender to point to the head of the chain
	qs.next = wrapped

	// 3. Start the queue (which starts the consumers).
	// The consumers will use the updated qs.next.
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

	// Shutdown active middlewares in reverse order of creation (LIFO).
	// Since we appended them as we wrapped (inner to outer), we simply iterate backwards.
	for i := len(qs.activeMiddlewares) - 1; i >= 0; i-- {
		errs = multierr.Append(errs, qs.activeMiddlewares[i].Shutdown(ctx))
	}
	qs.activeMiddlewares = nil

	return errs
}
