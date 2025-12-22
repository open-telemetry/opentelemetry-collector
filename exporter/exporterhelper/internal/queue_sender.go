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

const minConsumersWithController = 200

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

// queueSender wraps QueueBatch to add ConcurrencyController logic.
type queueSender struct {
	*queuebatch.QueueBatch
	ctrlID *component.ID
	ctrl   extensioncapabilities.ConcurrencyController

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
	enforceMinConsumers(&qCfg, qSet.Telemetry.Logger)

	qs := &queueSender{
		ctrlID: qCfg.ConcurrencyControllerID,
		id:     qSet.ID,
		signal: qSet.Signal,
	}

	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()

		ctrl := qs.ctrl
		if ctrl != nil {
			// This blocks the worker if the dynamic limit is reached.
			if err := ctrl.Acquire(ctx); err != nil {
				return err
			}

			start := time.Now()
			var errSend error
			defer func() { ctrl.Record(ctx, time.Since(start), errSend) }()

			errSend = next.Send(ctx, req)
			if errSend != nil {
				qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
					zap.Error(errSend), zap.Int("dropped_items", itemsCount))
				return errSend
			}
			return nil
		}

		errSend := next.Send(ctx, req)
		if errSend != nil {
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

	return qs, nil
}

// enforceMinConsumers ensures sufficient worker headroom when a concurrency controller is configured.
func enforceMinConsumers(cfg *queuebatch.Config, logger *zap.Logger) {
	if cfg.NumConsumers < 1 {
		cfg.NumConsumers = 1
	}

	if cfg.ConcurrencyControllerID == nil {
		return
	}

	defaultConsumers := NewDefaultQueueConfig().NumConsumers

	// If the user left the default worker count, bump it to allow the controller to be the governor.
	if cfg.NumConsumers == defaultConsumers && cfg.NumConsumers < minConsumersWithController {
		if logger != nil {
			logger.Warn("sending_queue.num_consumers overridden because concurrency_controller is configured",
				zap.Int("original", cfg.NumConsumers),
				zap.Int("override", minConsumersWithController),
				zap.String("controller", cfg.ConcurrencyControllerID.String()),
			)
		}
		cfg.NumConsumers = minConsumersWithController
		return
	}

	// If the user explicitly set num_consumers below the recommended headroom, don't override,
	// but warn that the controller will be capped by the worker pool.
	if cfg.NumConsumers < minConsumersWithController && logger != nil {
		logger.Warn("sending_queue.num_consumers caps concurrency_controller; consider increasing num_consumers for headroom",
			zap.Int("num_consumers", cfg.NumConsumers),
			zap.Int("recommended_min", minConsumersWithController),
			zap.String("controller", cfg.ConcurrencyControllerID.String()),
		)
	}
}

// Start overrides the default Start to resolve the extension.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	if qs.ctrlID != nil {
		ext, ok := host.GetExtensions()[*qs.ctrlID]
		if !ok {
			return fmt.Errorf("concurrency controller extension %q not found", qs.ctrlID.String())
		}
		factory, ok := ext.(extensioncapabilities.ConcurrencyControllerFactory)
		if !ok {
			return fmt.Errorf("extension %q does not implement ConcurrencyControllerFactory", qs.ctrlID.String())
		}
		ctrl, err := factory.CreateConcurrencyController(qs.id, qs.signal)
		if err != nil {
			return fmt.Errorf("failed to create concurrency controller: %w", err)
		}
		qs.ctrl = ctrl
	}

	if err := qs.QueueBatch.Start(ctx, host); err != nil {
		if qs.ctrl != nil {
			_ = qs.ctrl.Shutdown(ctx)
			qs.ctrl = nil
		}
		return err
	}
	return nil
}

// Shutdown ensures the controller is also shut down.
func (qs *queueSender) Shutdown(ctx context.Context) error {
	errQ := qs.QueueBatch.Shutdown(ctx)

	if qs.ctrl != nil {
		if err := qs.ctrl.Shutdown(ctx); err != nil && errQ == nil {
			return err
		}
		qs.ctrl = nil
	}
	return errQ
}
