// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opencensus.io/metric/metricdata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

const defaultQueueSize = 1000

var errSendingQueueIsFull = errors.New("sending_queue is full")

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
	StorageID *component.ID `mapstructure:"storage"`
	// WaitOnSend if enabled will wait when queue is full instead of dropping requests.
	WaitOnSend WaitOnSendSettings `mapstructure:"wait_on_send"`
}

type WaitOnSendSettings struct {
	// Enabled indicates whether sending requests will block instead of dropping while queue is full. Default is disabled.
	Enabled bool `mapstructure:"enabled"`
	// Timeout is the duration to wait before dropping a request. Default is 60s.
	Timeout time.Duration `mapstructure:"timeout"`
}

// NewDefaultQueueSettings returns the default settings for QueueSettings.
func NewDefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Enabled:      true,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize: defaultQueueSize,
		WaitOnSend: WaitOnSendSettings{
			Enabled: true,
			Timeout: 1 * time.Minute,
		},
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

type queueSender struct {
	baseRequestSender
	fullName         string
	id               component.ID
	signal           component.DataType
	queue            internal.ProducerConsumerQueue
	traceAttribute   attribute.KeyValue
	logger           *zap.Logger
	requeuingEnabled bool
	waitset          WaitOnSendSettings
}

func newQueueSender(id component.ID, signal component.DataType, queue internal.ProducerConsumerQueue, logger *zap.Logger, waitset WaitOnSendSettings) *queueSender {
	return &queueSender{
		fullName:       id.String(),
		id:             id,
		signal:         signal,
		queue:          queue,
		traceAttribute: attribute.String(obsmetrics.ExporterKey, id.String()),
		logger:         logger,
		// TODO: this can be further exposed as a config param rather than relying on a type of queue
		requeuingEnabled: queue != nil && queue.IsPersistent(),
		waitset:          waitset,
	}
}

func (qs *queueSender) onTemporaryFailure(logger *zap.Logger, req internal.Request, err error) error {
	if !qs.requeuingEnabled || qs.queue == nil {
		logger.Error(
			"Exporting failed. No more retries left. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.Count()),
		)
		return err
	}

	if qs.queue.Produce(req) {
		logger.Error(
			"Exporting failed. Putting back to the end of the queue.",
			zap.Error(err),
		)
	} else {
		logger.Error(
			"Exporting failed. Queue did not accept requeuing request. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.Count()),
		)
	}
	return err
}

// start is invoked during service startup.
func (qs *queueSender) start(ctx context.Context, host component.Host, set exporter.CreateSettings) error {
	if qs.queue == nil {
		return nil
	}

	err := qs.queue.Start(ctx, host, internal.QueueSettings{
		CreateSettings: set,
		DataType:       qs.signal,
		Callback: func(item internal.Request) {
			err := qs.nextSender.send(item)
			ch := qs.queue.GetErrCh()
			// channel capacity is NumConsumers, so this callback will block
			// until these return values are processed.
			if ch != nil {
				ch <- err
			}
			item.OnProcessingFinished()
		},
	})
	if err != nil {
		return err
	}

	// Start reporting queue length metric
	err = globalInstruments.queueSize.UpsertEntry(func() int64 {
		return int64(qs.queue.Size())
	}, metricdata.NewLabelValue(qs.fullName))
	if err != nil {
		return fmt.Errorf("failed to create retry queue size metric: %w", err)
	}
	err = globalInstruments.queueCapacity.UpsertEntry(func() int64 {
		return int64(qs.queue.Capacity())
	}, metricdata.NewLabelValue(qs.fullName))
	if err != nil {
		return fmt.Errorf("failed to create retry queue capacity metric: %w", err)
	}

	return nil
}

// shutdown is invoked during service shutdown.
func (qs *queueSender) shutdown() {
	if qs.queue != nil {
		// Cleanup queue metrics reporting
		_ = globalInstruments.queueSize.UpsertEntry(func() int64 {
			return int64(0)
		}, metricdata.NewLabelValue(qs.fullName))

		// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
		// try once every request.
		qs.queue.Stop()
	}
}

// send implements the requestSender interface
func (qs *queueSender) send(req internal.Request) error {
	if qs.queue == nil {
		err := qs.nextSender.send(req)
		if err != nil {
			qs.logger.Error(
				"Exporting failed. Dropping data. Try enabling sending_queue to survive temporary failures.",
				zap.Int("dropped_items", req.Count()),
			)
		}
		return err
	}

	// Prevent cancellation and deadline to propagate to the context stored in the queue.
	// The grpc/http based receivers will cancel the request context after this function returns.
	req.SetContext(noCancellationContext{Context: req.Context()})

	span := trace.SpanFromContext(req.Context())
	if !qs.waitset.Enabled {
		if !qs.queue.Produce(req) {
			qs.logger.Error(
				"Dropping data because sending_queue is full. Try increasing queue_size.",
				zap.Int("dropped_items", req.Count()),
			)
			span.AddEvent("Dropped item, sending_queue is full.", trace.WithAttributes(qs.traceAttribute))
			return errSendingQueueIsFull
		}
	} else { // wait for response
		err := qs.sendAndWait(req)
		if err != nil {
			qs.logger.Error(
				"Dropping data: found error when sending", err,
				zap.Int("dropped_items", req.Count()),
			)
			span.AddEvent("Dropped item, found error when sending.", trace.WithAttributes(qs.traceAttribute))
			return err
		}
	}

	span.AddEvent("Enqueued item.", trace.WithAttributes(qs.traceAttribute))
	return nil
}

func (qs *queueSender) sendAndWait(req internal.Request) error {
	span := trace.SpanFromContext(req.Context())
	// should this call to ProduceAndWait be in a goroutine,
	// so fetching a response will not be blocked by a full queue
	err := ProduceAndWait(req, qs.waitset.Timeout)
	if err != nil {
		return err
	}
	// blocks until we get first ready response.
	err, ok := <-qs.queue.GetErrCh()

	if !ok {
		// channel closed
		return nil
	}

	return err
}

type noCancellationContext struct {
	context.Context
}

func (noCancellationContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (noCancellationContext) Done() <-chan struct{} {
	return nil
}

func (noCancellationContext) Err() error {
	return nil
}
