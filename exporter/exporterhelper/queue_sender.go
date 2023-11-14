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
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

const defaultQueueSize = 1000

var (
	scopeName = "go.opentelemetry.io/collector/exporterhelper"
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
	StorageID *component.ID `mapstructure:"storage"`
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

	if qCfg.NumConsumers <= 0 {
		return errors.New("number of queue consumers must be positive")
	}

	return nil
}

type queueSender struct {
	baseRequestSender
	fullName         string
	signal           component.DataType
	queue            internal.Queue
	traceAttribute   attribute.KeyValue
	logger           *zap.Logger
	meter            otelmetric.Meter
	requeuingEnabled bool

	metricCapacity otelmetric.Int64ObservableGauge
	metricSize     otelmetric.Int64ObservableGauge
}

func newQueueSender(config QueueSettings, set exporter.CreateSettings, signal component.DataType,
	marshaler RequestMarshaler, unmarshaler RequestUnmarshaler) *queueSender {

	isPersistent := config.StorageID != nil
	var queue internal.Queue
	if isPersistent {
		queue = internal.NewPersistentQueue(config.QueueSize, config.NumConsumers, *config.StorageID,
			queueRequestMarshaler(marshaler), queueRequestUnmarshaler(unmarshaler), set)
	} else {
		queue = internal.NewBoundedMemoryQueue(config.QueueSize, config.NumConsumers)
	}
	return &queueSender{
		fullName:       set.ID.String(),
		signal:         signal,
		queue:          queue,
		traceAttribute: attribute.String(obsmetrics.ExporterKey, set.ID.String()),
		logger:         set.TelemetrySettings.Logger,
		meter:          set.TelemetrySettings.MeterProvider.Meter(scopeName),
		// TODO: this can be further exposed as a config param rather than relying on a type of queue
		requeuingEnabled: isPersistent,
	}
}

func (qs *queueSender) onTemporaryFailure(ctx context.Context, req Request, err error, logger *zap.Logger) error {
	if !qs.requeuingEnabled {
		logger.Error(
			"Exporting failed. No more retries left. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.ItemsCount()),
		)
		return err
	}

	if qs.queue.Offer(ctx, req) == nil {
		logger.Error(
			"Exporting failed. Putting back to the end of the queue.",
			zap.Error(err),
		)
	} else {
		logger.Error(
			"Exporting failed. Queue did not accept requeuing request. Dropping data.",
			zap.Error(err),
			zap.Int("dropped_items", req.ItemsCount()),
		)
	}
	return err
}

// Start is invoked during service startup.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	err := qs.queue.Start(ctx, host, internal.QueueSettings{
		DataType: qs.signal,
		Callback: func(qr internal.QueueRequest) {
			_ = qs.nextSender.send(qr.Context, qr.Request.(Request))
			// TODO: Update OnProcessingFinished to accept error and remove the retry->queue sender callback.
			qr.OnProcessingFinished()
		},
	})
	if err != nil {
		return err
	}

	if obsreportconfig.UseOtelForInternalMetricsfeatureGate.IsEnabled() {
		return qs.recordWithOtel()
	}
	return qs.recordWithOC()
}

func (qs *queueSender) recordWithOtel() error {
	var err, errs error

	attrs := otelmetric.WithAttributeSet(attribute.NewSet(attribute.String(obsmetrics.ExporterKey, qs.fullName)))

	qs.metricSize, err = qs.meter.Int64ObservableGauge(
		obsmetrics.ExporterKey+"/queue_size",
		otelmetric.WithDescription("Current size of the retry queue (in batches)"),
		otelmetric.WithUnit("1"),
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(qs.queue.Size()), attrs)
			return nil
		}),
	)
	errs = multierr.Append(errs, err)

	qs.metricCapacity, err = qs.meter.Int64ObservableGauge(
		obsmetrics.ExporterKey+"/queue_capacity",
		otelmetric.WithDescription("Fixed capacity of the retry queue (in batches)"),
		otelmetric.WithUnit("1"),
		otelmetric.WithInt64Callback(func(_ context.Context, o otelmetric.Int64Observer) error {
			o.Observe(int64(qs.queue.Capacity()), attrs)
			return nil
		}))

	errs = multierr.Append(errs, err)
	return errs
}

func (qs *queueSender) recordWithOC() error {
	// Start reporting queue length metric
	err := globalInstruments.queueSize.UpsertEntry(func() int64 {
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

// Shutdown is invoked during service shutdown.
func (qs *queueSender) Shutdown(ctx context.Context) error {
	// Cleanup queue metrics reporting
	_ = globalInstruments.queueSize.UpsertEntry(func() int64 {
		return int64(0)
	}, metricdata.NewLabelValue(qs.fullName))

	// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	return qs.queue.Shutdown(ctx)
}

// send implements the requestSender interface
func (qs *queueSender) send(ctx context.Context, req Request) error {
	// Prevent cancellation and deadline to propagate to the context stored in the queue.
	// The grpc/http based receivers will cancel the request context after this function returns.
	c := noCancellationContext{Context: ctx}

	span := trace.SpanFromContext(c)
	if err := qs.queue.Offer(c, req); err != nil {
		qs.logger.Error(
			"Dropping data because sending_queue is full. Try increasing queue_size.",
			zap.Int("dropped_items", req.ItemsCount()),
		)
		span.AddEvent("Dropped item, sending_queue is full.", trace.WithAttributes(qs.traceAttribute))
		return err
	}

	span.AddEvent("Enqueued item.", trace.WithAttributes(qs.traceAttribute))
	return nil
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

func queueRequestMarshaler(marshaler RequestMarshaler) internal.QueueRequestMarshaler {
	return func(req any) ([]byte, error) {
		return marshaler(req.(Request))
	}
}

func queueRequestUnmarshaler(unmarshaler RequestUnmarshaler) internal.QueueRequestUnmarshaler {
	return func(data []byte) (any, error) {
		return unmarshaler(data)
	}
}
