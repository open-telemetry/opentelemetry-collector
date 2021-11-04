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

//go:build !enable_unstable
// +build !enable_unstable

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"fmt"

	"go.opencensus.io/metric/metricdata"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// queued_retry_inmemory includes the code for memory-backed (original) queued retry helper only
// enabled when "enable_unstable" build tag is not set

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
}

// DefaultQueueSettings returns the default settings for QueueSettings.
func DefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Enabled:      true,
		NumConsumers: 10,
		// For 5000 queue elements at 100 requests/sec gives about 50 sec of survival of destination outage.
		// This is a pretty decent value for production.
		// User should calculate this from the perspective of how many seconds to buffer in case of a backend outage,
		// multiply that by the number of requests per seconds.
		QueueSize: 5000,
	}
}

type queuedRetrySender struct {
	fullName        string
	cfg             QueueSettings
	consumerSender  requestSender
	queue           internal.ProducerConsumerQueue
	retryStopCh     chan struct{}
	traceAttributes []attribute.KeyValue
	logger          *zap.Logger
}

func newQueuedRetrySender(id config.ComponentID, _ config.DataType, qCfg QueueSettings, rCfg RetrySettings, _ internal.RequestUnmarshaler, nextSender requestSender, logger *zap.Logger) *queuedRetrySender {
	retryStopCh := make(chan struct{})
	sampledLogger := createSampledLogger(logger)
	traceAttr := attribute.String(obsmetrics.ExporterKey, id.String())
	return &queuedRetrySender{
		fullName: id.String(),
		cfg:      qCfg,
		consumerSender: &retrySender{
			traceAttribute:     traceAttr,
			cfg:                rCfg,
			nextSender:         nextSender,
			stopCh:             retryStopCh,
			logger:             sampledLogger,
			onTemporaryFailure: onTemporaryFailure,
		},
		queue:           internal.NewBoundedMemoryQueue(qCfg.QueueSize, func(item interface{}) {}),
		retryStopCh:     retryStopCh,
		traceAttributes: []attribute.KeyValue{traceAttr},
		logger:          sampledLogger,
	}
}

func onTemporaryFailure(logger *zap.Logger, req request, err error) error {
	logger.Error(
		"Exporting failed. No more retries left. Dropping data.",
		zap.Error(err),
		zap.Int("dropped_items", req.count()),
	)
	return err
}

// start is invoked during service startup.
func (qrs *queuedRetrySender) start(context.Context, component.Host) error {
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
			return fmt.Errorf("failed to create retry queue size metric: %v", err)
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
