// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/arc"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
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
		Arc: arc.DefaultConfig(),
	}
}

func NewQueueSender(
	qSet queuebatch.AllSettings[request.Request],
	qCfg queuebatch.Config,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	var arcCtl *arc.Controller
	exportFunc := func(ctx context.Context, req request.Request) error {
		itemsCount := req.ItemsCount()
		if errSend := next.Send(ctx, req); errSend != nil {
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	if qCfg.Arc.Enabled {
		// Ensure physical consumers can support the theoretical ARC limit
		if qCfg.NumConsumers < qCfg.Arc.MaxConcurrency {
			qCfg.NumConsumers = qCfg.Arc.MaxConcurrency
		}

		tel, err := metadata.NewTelemetryBuilder(qSet.Telemetry)
		if err != nil {
			return nil, fmt.Errorf("failed to create telemetry builder for ARC: %w", err)
		}
		arcCtl = arc.NewController(qCfg.Arc, tel, qSet.ID, qSet.Signal)

		exporterAttr := attribute.String("exporter", qSet.ID.String())
		dataTypeAttr := attribute.String("data_type", strings.ToLower(qSet.Signal.String()))
		arcAttrs := metric.WithAttributes(exporterAttr, dataTypeAttr)

		origExportFunc := exportFunc
		exportFunc = func(ctx context.Context, req request.Request) error {
			startWait := time.Now()
			if !arcCtl.Acquire(ctx) {
				waitMs := time.Since(startWait).Milliseconds()
				tel.ExporterArcAcquireWaitMs.Record(ctx, waitMs, arcAttrs)
				tel.ExporterArcFailures.Add(ctx, 1, arcAttrs)

				if err := ctx.Err(); err != nil {
					return err
				}

				return experr.NewShutdownErr(errors.New("arc semaphore closed"))
			}
			waitMs := time.Since(startWait).Milliseconds()
			tel.ExporterArcAcquireWaitMs.Record(ctx, waitMs, arcAttrs)

			arcCtl.StartRequest()
			startTime := time.Now()
			err := origExportFunc(ctx, req)
			rtt := time.Since(startTime)

			isBackpressure := experr.IsRetryableError(err)
			isSuccess := err == nil

			arcCtl.ReleaseWithSample(ctx, rtt, isSuccess, isBackpressure)
			return err
		}
	}

	qbs, err := queuebatch.NewQueueBatch(qSet, qCfg, exportFunc)
	if err != nil {
		return nil, err
	}

	if arcCtl != nil {
		return &queueSenderWithArc{
			QueueBatch: qbs,
			arcCtl:     arcCtl,
		}, nil
	}

	return qbs, nil
}

// queueSenderWithArc combines the QueueBatch sender with an arc.Controller
// to ensure both are started and shut down correctly.
type queueSenderWithArc struct {
	*queuebatch.QueueBatch
	arcCtl *arc.Controller
}

// Shutdown shuts down both the queue/batcher and the ARC controller.
func (qs *queueSenderWithArc) Shutdown(ctx context.Context) error {
	qs.arcCtl.Shutdown()
	return qs.QueueBatch.Shutdown(ctx)
}

// Start starts both the queue/batcher and the ARC controller.
// Note: ARC controller doesn't have a Start, so this just passes through.
func (qs *queueSenderWithArc) Start(ctx context.Context, host component.Host) error {
	return qs.QueueBatch.Start(ctx, host)
}
