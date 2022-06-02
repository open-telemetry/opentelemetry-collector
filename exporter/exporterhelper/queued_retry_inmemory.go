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

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

// TODO: rename the file to queued_retry_sender.go after merging the PR

import (
	"context"
	"errors"
	"fmt"

	"go.opencensus.io/metric/metricdata"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue
	StorageID *config.ComponentID `mapstructure:"storage"`
	// StorageEnabled describes whether persistence via a file storage extension is enabled using the single
	// default storage extension.
	StorageEnabled bool `mapstructure:"persistent_storage_enabled"`
}

// NewDefaultQueueSettings returns the default settings for QueueSettings.
func NewDefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Enabled:      true,
		NumConsumers: 10,
		// For 5000 queue elements at 100 requests/sec gives about 50 sec of survival of destination outage.
		// This is a pretty decent value for production.
		// User should calculate this from the perspective of how many seconds to buffer in case of a backend outage,
		// multiply that by the number of requests per seconds.
		QueueSize:      5000,
		StorageEnabled: false,
	}
}

// Validate checks if the QueueSettings configuration is valid
func (qCfg *QueueSettings) Validate() error {
	if !qCfg.Enabled {
		return nil
	}

	if qCfg.QueueSize <= 0 {
		return errors.New("queue size must be positive")
	}

	return nil
}

func (qCfg *QueueSettings) getStorageExtension(logger *zap.Logger, extensions map[config.ComponentID]component.Extension) (storage.Extension, error) {
	if qCfg.StorageID != nil {
		if ext, found := extensions[*qCfg.StorageID]; found {
			if storageExt, ok := ext.(storage.Extension); ok {
				return storageExt, nil
			}
			return nil, errWrongExtensionType
		}
	} else if qCfg.StorageEnabled {
		var storageExtension storage.Extension
		for _, ext := range extensions {
			if se, ok := ext.(storage.Extension); ok {
				if storageExtension != nil {
					logger.Error("multiple storage extensions found, please specify which one to use using `storage: ` configuration attribute")
					return nil, errMultipleStorageClients
				}
				storageExtension = se
			}
		}

		return storageExtension, nil
	}

	return nil, nil
}

func (qCfg *QueueSettings) toStorageClient(ctx context.Context, logger *zap.Logger, host component.Host, ownerID config.ComponentID, signal config.DataType) (storage.Client, error) {
	extension, err := qCfg.getStorageExtension(logger, host.GetExtensions())
	if err != nil {
		return nil, err
	}
	if extension == nil {
		return nil, errNoStorageClient
	}

	client, err := extension.GetClient(ctx, component.KindExporter, ownerID, string(signal))
	if err != nil {
		return nil, err
	}

	return client, err
}

var (
	errNoStorageClient        = errors.New("no storage client extension found")
	errWrongExtensionType     = errors.New("requested extension is not a storage extension")
	errMultipleStorageClients = errors.New("multiple storage extensions found while default extension expected")
)

type queuedRetrySender struct {
	fullName           string
	id                 config.ComponentID
	signal             config.DataType
	cfg                QueueSettings
	consumerSender     requestSender
	queue              internal.ProducerConsumerQueue
	queueStartFunc     internal.PersistentQueueStartFunc
	retryStopCh        chan struct{}
	traceAttributes    []attribute.KeyValue
	logger             *zap.Logger
	requeuingEnabled   bool
	requestUnmarshaler internal.RequestUnmarshaler
}

func newQueuedRetrySender(id config.ComponentID, signal config.DataType, qCfg QueueSettings, rCfg RetrySettings, reqUnmarshaler internal.RequestUnmarshaler, nextSender requestSender, logger *zap.Logger) *queuedRetrySender {
	retryStopCh := make(chan struct{})
	sampledLogger := createSampledLogger(logger)
	traceAttr := attribute.String(obsmetrics.ExporterKey, id.String())

	qrs := &queuedRetrySender{
		fullName:           id.String(),
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

	if qCfg.StorageEnabled || qCfg.StorageID != nil {
		qrs.queue, qrs.queueStartFunc = internal.NewPersistentQueue(qrs.fullName, qrs.signal, qrs.cfg.QueueSize, qrs.logger, qrs.requestUnmarshaler)
		// TODO: following can be further exposed as a config param rather than relying on a type of queue
		qrs.requeuingEnabled = true
	} else {
		qrs.queue = internal.NewBoundedMemoryQueue(qrs.cfg.QueueSize, func(item interface{}) {})
	}

	return qrs
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
	if qrs.queueStartFunc != nil {
		storageClient, err := qrs.cfg.toStorageClient(ctx, qrs.logger, host, qrs.id, qrs.signal)
		if err != nil {
			return err
		}
		qrs.queueStartFunc(ctx, storageClient)
	}

	qrs.queue.StartConsumers(qrs.cfg.NumConsumers, func(item interface{}) {
		req := item.(request)
		_ = qrs.consumerSender.send(req)
		req.OnProcessingFinished()
	})

	// Start reporting queue length metric
	if qrs.cfg.Enabled {
		err := globalInstruments.queueSize.UpsertEntry(func() int64 {
			return int64(qrs.queue.Size())
		}, metricdata.NewLabelValue(qrs.fullName))
		if err != nil {
			return fmt.Errorf("failed to create retry queue size metric: %w", err)
		}
		err = globalInstruments.queueCapacity.UpsertEntry(func() int64 {
			return int64(qrs.cfg.QueueSize)
		}, metricdata.NewLabelValue(qrs.fullName))
		if err != nil {
			return fmt.Errorf("failed to create retry queue capacity metric: %w", err)
		}
	}

	return nil
}

// shutdown is invoked during service shutdown.
func (qrs *queuedRetrySender) shutdown() {
	// Cleanup queue metrics reporting
	if qrs.cfg.Enabled {
		_ = globalInstruments.queueSize.UpsertEntry(func() int64 {
			return int64(0)
		}, metricdata.NewLabelValue(qrs.fullName))
	}

	// First Stop the retry goroutines, so that unblocks the queue numWorkers.
	close(qrs.retryStopCh)

	// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	if qrs.queue != nil {
		qrs.queue.Stop()
	}
}
