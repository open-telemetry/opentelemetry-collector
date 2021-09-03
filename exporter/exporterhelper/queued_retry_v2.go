// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build enable_unstable
// +build enable_unstable

package exporterhelper

import (
	"context"
	"errors"
	"fmt"

	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/extension/storage"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// queued_retry_v2 includes the code for both memory-backed and persistent-storage backed queued retry helpers
// enabled by setting "enable_unstable" build tag

var (
	currentlyDispatchedBatchesGauge, _ = r.AddInt64DerivedGauge(
		obsmetrics.ExporterKey+"/currently_dispatched_batches",
		metric.WithDescription("Number of batches that are currently being sent"),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	errNoStorageClient        = errors.New("no storage client extension found")
	errMultipleStorageClients = errors.New("multiple storage extensions found")
)

type queuedRetrySender struct {
	id                 config.ComponentID
	signal             config.DataType
	cfg                QueueSettings
	consumerSender     requestSender
	queue              consumersQueue
	retryStopCh        chan struct{}
	traceAttributes    []attribute.KeyValue
	logger             *zap.Logger
	requeuingEnabled   bool
	requestUnmarshaler requestUnmarshaler
}

func (qrs *queuedRetrySender) fullName() string {
	if qrs.signal == "" {
		return qrs.id.String()
	}
	return fmt.Sprintf("%s-%s", qrs.id.String(), qrs.signal)
}

func newQueuedRetrySender(id config.ComponentID, signal config.DataType, qCfg QueueSettings, rCfg RetrySettings, reqUnmarshaler requestUnmarshaler, nextSender requestSender, logger *zap.Logger) *queuedRetrySender {
	retryStopCh := make(chan struct{})
	sampledLogger := createSampledLogger(logger)
	traceAttr := attribute.String(obsmetrics.ExporterKey, id.String())

	qrs := &queuedRetrySender{
		id:                 id,
		signal:             signal,
		cfg:                qCfg,
		retryStopCh:        retryStopCh,
		traceAttributes:    []attribute.KeyValue{traceAttr},
		logger:             sampledLogger,
		requestUnmarshaler: reqUnmarshaler,
	}

	qrs.consumerSender = &retrySender{
		traceAttribute: traceAttr,
		cfg:            rCfg,
		nextSender:     nextSender,
		stopCh:         retryStopCh,
		logger:         sampledLogger,
		// Following three functions actually depend on queuedRetrySender
		onTemporaryFailure: qrs.onTemporaryFailure,
	}

	if !qCfg.PersistentStorageEnabled {
		qrs.queue = internal.NewBoundedQueue(qrs.cfg.QueueSize, func(item interface{}) {})
	}
	// The Persistent Queue is initialized separately as it needs extra information about the component

	return qrs
}

func getStorageClient(ctx context.Context, host component.Host, id config.ComponentID, signal config.DataType) (*storage.Client, error) {
	var storageExtension storage.Extension
	for _, ext := range host.GetExtensions() {
		if se, ok := ext.(storage.Extension); ok {
			if storageExtension != nil {
				return nil, errMultipleStorageClients
			}
			storageExtension = se
		}
	}

	if storageExtension == nil {
		return nil, errNoStorageClient
	}

	client, err := storageExtension.GetClient(ctx, component.KindExporter, id, string(signal))
	if err != nil {
		return nil, err
	}

	return &client, err
}

// initializePersistentQueue uses extra information for initialization available from component.Host
func (qrs *queuedRetrySender) initializePersistentQueue(ctx context.Context, host component.Host) error {
	if qrs.cfg.PersistentStorageEnabled {
		storageClient, err := getStorageClient(ctx, host, qrs.id, qrs.signal)
		if err != nil {
			return err
		}

		qrs.queue = newPersistentQueue(ctx, qrs.fullName(), qrs.cfg.QueueSize, qrs.logger, *storageClient, qrs.requestUnmarshaler)

		// TODO: this can be further exposed as a config param rather than relying on a type of queue
		qrs.requeuingEnabled = true
	}

	return nil
}

func (qrs *queuedRetrySender) onTemporaryFailure(logger *zap.Logger, req request, err error) error {
	if !qrs.requeuingEnabled || qrs.queue == nil {
		logger.Error(
			"Exporting failed. No more retries left. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.count()),
		)
		return err
	}

	if qrs.queue.Produce(req) {
		logger.Error(
			"Exporting failed. Putting back to the end of the queue.",
			zap.Error(err),
		)
	} else {
		logger.Error(
			"Exporting failed. Queue did not accept requeuing request. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.count()),
		)
	}
	return err
}

// start is invoked during service startup.
func (qrs *queuedRetrySender) start(ctx context.Context, host component.Host) error {
	err := qrs.initializePersistentQueue(ctx, host)
	if err != nil {
		return err
	}

	qrs.queue.StartConsumers(qrs.cfg.NumConsumers, func(item interface{}) {
		req := item.(request)
		_ = qrs.consumerSender.send(req)
		req.onProcessingFinished()
	})

	// Start reporting queue length metric
	if qrs.cfg.Enabled {
		err := queueSizeGauge.UpsertEntry(func() int64 {
			return int64(qrs.queue.Size())
		}, metricdata.NewLabelValue(qrs.fullName()))
		if err != nil {
			return fmt.Errorf("failed to create retry queue size metric: %v", err)
		}
	}

	return nil
}

// shutdown is invoked during service shutdown.
func (qrs *queuedRetrySender) shutdown() {
	// Cleanup queue metrics reporting
	if qrs.cfg.Enabled {
		_ = queueSizeGauge.UpsertEntry(func() int64 {
			return int64(0)
		}, metricdata.NewLabelValue(qrs.fullName()))
	}

	// First Stop the retry goroutines, so that unblocks the queue numWorkers.
	close(qrs.retryStopCh)

	// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	if qrs.queue != nil {
		qrs.queue.Stop()
	}
}
