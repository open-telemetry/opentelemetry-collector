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
	enforceMinConsumers(&qCfg)

	qs := &queueSender{
		ctrlID: qCfg.ConcurrencyControllerID,
		id:     qSet.ID,
		signal: qSet.Signal,
	}

	exportFunc := func(ctx context.Context, req request.Request) error {
		itemsCount := req.ItemsCount()

		if qs.ctrl != nil {
			// This blocks the worker if the dynamic limit is reached.
			if err := qs.ctrl.Acquire(ctx); err != nil {
				return err
			}
		}

		var errSend error
		var start time.Time
		if qs.ctrl != nil {
			start = time.Now()
			defer func() { qs.ctrl.Record(ctx, time.Since(start), errSend) }()
		}

		errSend = next.Send(ctx, req)

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
func enforceMinConsumers(cfg *queuebatch.Config) {
	if cfg.ConcurrencyControllerID != nil && cfg.NumConsumers < minConsumersWithController {
		cfg.NumConsumers = minConsumersWithController
	}
}

// Start overrides the default Start to resolve the extension.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	if qs.ctrlID != nil {
		ext, ok := host.GetExtensions()[*qs.ctrlID]
		if !ok {
			return fmt.Errorf("concurrency controller extension %q not found", *qs.ctrlID)
		}
		factory, ok := ext.(extensioncapabilities.ConcurrencyControllerFactory)
		if !ok {
			return fmt.Errorf("extension %q does not implement ConcurrencyControllerFactory", *qs.ctrlID)
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
	}
	return errQ
}
