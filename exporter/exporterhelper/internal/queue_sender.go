// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/queue"
)

// QueueConfig defines configuration for queueing batches before sending to the consumerSender.
type QueueConfig struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue. Defaults to 10.
	// If batching is enabled, a combined batch cannot contain more requests than the number of consumers.
	// So it's recommended to set higher number of consumers if batching is enabled.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
	// Blocking controls the queue behavior when full.
	// If true it blocks until enough space to add the new request to the queue.
	Blocking bool `mapstructure:"blocking"`
	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue
	StorageID *component.ID `mapstructure:"storage"`
}

// NewDefaultQueueConfig returns the default config for QueueConfig.
func NewDefaultQueueConfig() QueueConfig {
	return QueueConfig{
		Enabled:      true,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize: 1_000,
		Blocking:  false,
	}
}

// Validate checks if the QueueConfig configuration is valid
func (qCfg *QueueConfig) Validate() error {
	if !qCfg.Enabled {
		return nil
	}

	if qCfg.QueueSize <= 0 {
		return errors.New("`queue_size` must be positive")
	}

	if qCfg.NumConsumers <= 0 {
		return errors.New("`num_consumers` must be positive")
	}

	return nil
}

type QueueSender struct {
	BaseSender[internal.Request]
	queue          exporterqueue.Queue[internal.Request]
	numConsumers   int
	traceAttribute attribute.KeyValue
	batcher        queue.Batcher
	consumers      *queue.Consumers[internal.Request]

	obsrep      *ObsReport
	exporterID  component.ID
	logger      *zap.Logger
	shutdownFns []component.ShutdownFunc
}

func NewQueueSender(
	q exporterqueue.Queue[internal.Request],
	set exporter.Settings,
	numConsumers int,
	exportFailureMessage string,
	obsrep *ObsReport,
	batcherCfg exporterbatcher.Config,
) *QueueSender {
	qs := &QueueSender{
		queue:          q,
		numConsumers:   numConsumers,
		traceAttribute: attribute.String(ExporterKey, set.ID.String()),
		obsrep:         obsrep,
		exporterID:     set.ID,
		logger:         set.Logger,
	}

	exportFunc := func(ctx context.Context, req internal.Request) error {
		err := qs.NextSender.Send(ctx, req)
		if err != nil {
			set.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(err), zap.Int("dropped_items", req.ItemsCount()))
		}
		return err
	}
	if usePullingBasedExporterQueueBatcher.IsEnabled() {
		qs.batcher, _ = queue.NewBatcher(batcherCfg, q, exportFunc, numConsumers)
	} else {
		qs.consumers = queue.NewQueueConsumers[internal.Request](q, numConsumers, exportFunc)
	}
	return qs
}

// Start is invoked during service startup.
func (qs *QueueSender) Start(ctx context.Context, host component.Host) error {
	if err := qs.queue.Start(ctx, host); err != nil {
		return err
	}

	if usePullingBasedExporterQueueBatcher.IsEnabled() {
		if err := qs.batcher.Start(ctx, host); err != nil {
			return err
		}
	} else {
		if err := qs.consumers.Start(ctx, host); err != nil {
			return err
		}
	}

	dataTypeAttr := attribute.String(DataTypeKey, qs.obsrep.Signal.String())

	reg1, err1 := qs.obsrep.TelemetryBuilder.InitExporterQueueSize(func() int64 { return qs.queue.Size() },
		metric.WithAttributeSet(attribute.NewSet(qs.traceAttribute, dataTypeAttr)))

	if reg1 != nil {
		qs.shutdownFns = append(qs.shutdownFns, func(context.Context) error {
			return reg1.Unregister()
		})
	}

	reg2, err2 := qs.obsrep.TelemetryBuilder.InitExporterQueueCapacity(func() int64 { return qs.queue.Capacity() },
		metric.WithAttributeSet(attribute.NewSet(qs.traceAttribute)))

	if reg2 != nil {
		qs.shutdownFns = append(qs.shutdownFns, func(context.Context) error {
			return reg2.Unregister()
		})
	}

	return errors.Join(err1, err2)
}

// Shutdown is invoked during service shutdown.
func (qs *QueueSender) Shutdown(ctx context.Context) error {
	// Stop the queue and consumers, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.

	for _, fn := range qs.shutdownFns {
		err := fn(ctx)
		if err != nil {
			qs.logger.Warn("Error while shutting down QueueSender", zap.Error(err))
		}
	}
	qs.shutdownFns = nil

	if err := qs.queue.Shutdown(ctx); err != nil {
		return err
	}
	if usePullingBasedExporterQueueBatcher.IsEnabled() {
		return qs.batcher.Shutdown(ctx)
	}
	return qs.consumers.Shutdown(ctx)
}

// send implements the requestSender interface. It puts the request in the queue.
func (qs *QueueSender) Send(ctx context.Context, req internal.Request) error {
	// Prevent cancellation and deadline to propagate to the context stored in the queue.
	// The grpc/http based receivers will cancel the request context after this function returns.
	c := context.WithoutCancel(ctx)

	span := trace.SpanFromContext(c)
	if err := qs.queue.Offer(c, req); err != nil {
		span.AddEvent("Failed to enqueue item.", trace.WithAttributes(qs.traceAttribute))
		return err
	}

	span.AddEvent("Enqueued item.", trace.WithAttributes(qs.traceAttribute))
	return nil
}

type MockHost struct {
	component.Host
	Ext map[component.ID]component.Component
}

func (nh *MockHost) GetExtensions() map[component.ID]component.Component {
	return nh.Ext
}
