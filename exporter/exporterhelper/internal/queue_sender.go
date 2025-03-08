// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/batcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/featuregate"
)

var _ = featuregate.GlobalRegistry().MustRegister(
	"exporter.UsePullingBasedExporterQueueBatcher",
	featuregate.StageStable,
	featuregate.WithRegisterFromVersion("v0.115.0"),
	featuregate.WithRegisterToVersion("v0.121.0"),
	featuregate.WithRegisterDescription("if set to true, turns on the pulling-based exporter queue bathcer"),
)

type QueueSender struct {
	queue   exporterqueue.Queue[request.Request]
	batcher component.Component
}

func NewQueueSender(
	qSet exporterqueue.Settings[request.Request],
	qCfg exporterqueue.Config,
	bCfg exporterbatcher.Config,
	exportFailureMessage string,
	next Sender[request.Request],
) (*QueueSender, error) {
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()
		err := next.Send(ctx, req)
		if err != nil {
			qSet.ExporterSettings.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(err), zap.Int("dropped_items", itemsCount))
		}
		return err
	}

	b, err := batcher.NewBatcher(bCfg, exportFunc, qCfg.NumConsumers)
	if err != nil {
		return nil, err
	}
	// TODO: https://github.com/open-telemetry/opentelemetry-collector/issues/12244
	if bCfg.Enabled {
		qCfg.NumConsumers = 1
	}
	q, err := newObsQueue(qSet, exporterqueue.NewQueue(context.Background(), qSet, qCfg, b.Consume))
	if err != nil {
		return nil, err
	}

	return &QueueSender{queue: q, batcher: b}, nil
}

// Start is invoked during service startup.
func (qs *QueueSender) Start(ctx context.Context, host component.Host) error {
	if err := qs.queue.Start(ctx, host); err != nil {
		return err
	}

	return qs.batcher.Start(ctx, host)
}

// Shutdown is invoked during service shutdown.
func (qs *QueueSender) Shutdown(ctx context.Context) error {
	// Stop the queue and batcher, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	return errors.Join(qs.queue.Shutdown(ctx), qs.batcher.Shutdown(ctx))
}

// Send implements the requestSender interface. It puts the request in the queue.
func (qs *QueueSender) Send(ctx context.Context, req request.Request) error {
	return qs.queue.Offer(ctx, req)
}

type MockHost struct {
	component.Host
	Ext map[component.ID]component.Component
}

func (nh *MockHost) GetExtensions() map[component.ID]component.Component {
	return nh.Ext
}
