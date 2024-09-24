// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/queue"
)

const defaultQueueSize = 1000

// Deprecated: [v0.110.0] Use QueueConfig instead.
type QueueSettings = QueueConfig

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
	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue
	StorageID *component.ID `mapstructure:"storage"`
}

// Deprecated: [v0.110.0] Use NewDefaultQueueConfig instead.
func NewDefaultQueueSettings() QueueSettings {
	return NewDefaultQueueConfig()
}

// NewDefaultQueueConfig returns the default config for QueueConfig.
func NewDefaultQueueConfig() QueueConfig {
	return QueueConfig{
		Enabled:      true,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize: defaultQueueSize,
	}
}

// Validate checks if the QueueConfig configuration is valid
func (qCfg *QueueConfig) Validate() error {
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

type QueueSender struct {
	BaseRequestSender
	queue          exporterqueue.Queue[internal.Request]
	numConsumers   int
	traceAttribute attribute.KeyValue
	consumers      *queue.Consumers[internal.Request]

	obsrep     *ObsReport
	exporterID component.ID
}

func NewQueueSender(q exporterqueue.Queue[internal.Request], set exporter.Settings, numConsumers int,
	exportFailureMessage string, obsrep *ObsReport) *QueueSender {
	qs := &QueueSender{
		queue:          q,
		numConsumers:   numConsumers,
		traceAttribute: attribute.String(ExporterKey, set.ID.String()),
		obsrep:         obsrep,
		exporterID:     set.ID,
	}
	consumeFunc := func(ctx context.Context, req internal.Request) error {
		err := qs.NextSender.Send(ctx, req)
		if err != nil {
			set.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(err), zap.Int("dropped_items", req.ItemsCount()))
		}
		return err
	}
	qs.consumers = queue.NewQueueConsumers[internal.Request](q, numConsumers, consumeFunc)
	return qs
}

// Start is invoked during service startup.
func (qs *QueueSender) Start(ctx context.Context, host component.Host) error {
	if err := qs.consumers.Start(ctx, host); err != nil {
		return err
	}

	dataTypeAttr := attribute.String(DataTypeKey, qs.obsrep.Signal.String())
	return multierr.Append(
		qs.obsrep.TelemetryBuilder.InitExporterQueueSize(func() int64 { return int64(qs.queue.Size()) },
			metric.WithAttributeSet(attribute.NewSet(qs.traceAttribute, dataTypeAttr))),
		qs.obsrep.TelemetryBuilder.InitExporterQueueCapacity(func() int64 { return int64(qs.queue.Capacity()) },
			metric.WithAttributeSet(attribute.NewSet(qs.traceAttribute))),
	)
}

// Shutdown is invoked during service shutdown.
func (qs *QueueSender) Shutdown(ctx context.Context) error {
	// Stop the queue and consumers, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
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
