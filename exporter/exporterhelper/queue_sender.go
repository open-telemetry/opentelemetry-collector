// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
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
	fullName       string
	queue          internal.Queue[Request]
	traceAttribute attribute.KeyValue
	logger         *zap.Logger
	meter          otelmetric.Meter
	consumers      *internal.QueueConsumers[Request]

	metricCapacity otelmetric.Int64ObservableGauge
	metricSize     otelmetric.Int64ObservableGauge
}

func newQueueSender(config QueueSettings, set exporter.CreateSettings, signal component.DataType,
	marshaler RequestMarshaler, unmarshaler RequestUnmarshaler, consumeErrHandler func(error, Request)) *queueSender {

	isPersistent := config.StorageID != nil
	var queue internal.Queue[Request]
	queueSizer := &internal.RequestSizer[Request]{}
	if isPersistent {
		queue = internal.NewPersistentQueue[Request](internal.PersistentQueueSettings[Request]{
			Sizer:            queueSizer,
			Capacity:         config.QueueSize,
			DataType:         signal,
			StorageID:        *config.StorageID,
			Marshaler:        marshaler,
			Unmarshaler:      unmarshaler,
			ExporterSettings: set,
		})
	} else {
		queue = internal.NewBoundedMemoryQueue[Request](internal.MemoryQueueSettings[Request]{
			Sizer:    queueSizer,
			Capacity: config.QueueSize,
		})
	}
	qs := &queueSender{
		fullName:       set.ID.String(),
		queue:          queue,
		traceAttribute: attribute.String(obsmetrics.ExporterKey, set.ID.String()),
		logger:         set.TelemetrySettings.Logger,
		meter:          set.TelemetrySettings.MeterProvider.Meter(scopeName),
	}
	consumeFunc := func(ctx context.Context, req Request) error {
		err := qs.nextSender.send(ctx, req)
		if err != nil {
			consumeErrHandler(err, req)
		}
		return err
	}
	qs.consumers = internal.NewQueueConsumers(queue, config.NumConsumers, consumeFunc)
	return qs
}

// Start is invoked during service startup.
func (qs *queueSender) Start(ctx context.Context, host component.Host) error {
	if err := qs.consumers.Start(ctx, host); err != nil {
		return err
	}

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

// Shutdown is invoked during service shutdown.
func (qs *queueSender) Shutdown(ctx context.Context) error {
	// Cleanup queue metrics reporting
	_ = globalInstruments.queueSize.UpsertEntry(func() int64 {
		return int64(0)
	}, metricdata.NewLabelValue(qs.fullName))

	// Stop the queue and consumers, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	return qs.consumers.Shutdown(ctx)
}

// send implements the requestSender interface. It puts the request in the queue.
func (qs *queueSender) send(ctx context.Context, req Request) error {
	// Prevent cancellation and deadline to propagate to the context stored in the queue.
	// The grpc/http based receivers will cancel the request context after this function returns.
	c := noCancellationContext{Context: ctx}

	span := trace.SpanFromContext(c)
	if err := qs.queue.Offer(c, req); err != nil {
		span.AddEvent("Failed to enqueue item.", trace.WithAttributes(qs.traceAttribute))
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
