// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	pdatareq "go.opentelemetry.io/collector/pdata/xpdata/request"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	metricsMarshaler   = &pmetric.ProtoMarshaler{}
	metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}
)

// NewMetricsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using pmetric.Metrics.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsQueueBatchSettings() QueueBatchSettings {
	return QueueBatchSettings{
		Encoding:   metricsEncoding{},
		ItemsSizer: request.NewItemsSizer(),
		BytesSizer: request.BaseSizer{
			SizeofFunc: func(req request.Request) int64 {
				return int64(metricsMarshaler.MetricsSize(req.(*metricsRequest).md))
			},
		},
	}
}

type metricsRequest struct {
	md         pmetric.Metrics
	cachedSize int
}

func newMetricsRequest(md pmetric.Metrics) Request {
	return &metricsRequest{
		md:         md,
		cachedSize: -1,
	}
}

type metricsEncoding struct{}

var _ QueueBatchEncoding[Request] = metricsEncoding{}

func (metricsEncoding) Unmarshal(bytes []byte) (context.Context, Request, error) {
	if queue.PersistRequestContextOnRead {
		ctx, metrics, err := pdatareq.UnmarshalMetrics(bytes)
		if errors.Is(err, pdatareq.ErrInvalidFormat) {
			// fall back to unmarshaling without context
			metrics, err = metricsUnmarshaler.UnmarshalMetrics(bytes)
		}
		return ctx, newMetricsRequest(metrics), err
	}
	metrics, err := metricsUnmarshaler.UnmarshalMetrics(bytes)
	if err != nil {
		var req Request
		return context.Background(), req, err
	}
	return context.Background(), newMetricsRequest(metrics), nil
}

func (metricsEncoding) Marshal(ctx context.Context, req Request) ([]byte, error) {
	metrics := req.(*metricsRequest).md
	if queue.PersistRequestContextOnWrite {
		return pdatareq.MarshalMetrics(ctx, metrics)
	}
	return metricsMarshaler.MarshalMetrics(metrics)
}

func (req *metricsRequest) OnError(err error) Request {
	var metricsError consumererror.Metrics
	if errors.As(err, &metricsError) {
		return newMetricsRequest(metricsError.Data())
	}
	return req
}

func (req *metricsRequest) ItemsCount() int {
	return req.md.DataPointCount()
}

func (req *metricsRequest) size(sizer sizer.MetricsSizer) int {
	if req.cachedSize == -1 {
		req.cachedSize = sizer.MetricsSize(req.md)
	}
	return req.cachedSize
}

func (req *metricsRequest) setCachedSize(count int) {
	req.cachedSize = count
}

type metricsExporter struct {
	*internal.BaseExporter
	consumer.Metrics
}

// NewMetrics creates an exporter.Metrics that records observability metrics and wraps every request with a Span.
func NewMetrics(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeMetricsFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushMetrics
	}
	return NewMetricsRequest(ctx, set, requestFromMetrics(), requestConsumeFromMetrics(pusher),
		append([]Option{internal.WithQueueBatchSettings(NewMetricsQueueBatchSettings())}, options...)...)
}

// requestConsumeFromMetrics returns a RequestConsumeFunc that consumes pmetric.Metrics.
func requestConsumeFromMetrics(pusher consumer.ConsumeMetricsFunc) RequestConsumeFunc {
	return func(ctx context.Context, request Request) error {
		return pusher.ConsumeMetrics(ctx, request.(*metricsRequest).md)
	}
}

// requestFromMetrics returns a RequestFromMetricsFunc that converts pdata.Metrics into a Request.
func requestFromMetrics() RequestConverterFunc[pmetric.Metrics] {
	return func(_ context.Context, md pmetric.Metrics) (Request, error) {
		return newMetricsRequest(md), nil
	}
}

// NewMetricsRequest creates a new metrics exporter based on a custom MetricsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[pmetric.Metrics],
	pusher RequestConsumeFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilMetricsConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalMetrics, pusher, options...)
	if err != nil {
		return nil, err
	}

	mc, err := consumer.NewMetrics(newConsumeMetrics(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &metricsExporter{BaseExporter: be, Metrics: mc}, nil
}

func newConsumeMetrics(converter RequestConverterFunc[pmetric.Metrics], be *internal.BaseExporter, logger *zap.Logger) consumer.ConsumeMetricsFunc {
	return func(ctx context.Context, md pmetric.Metrics) error {
		req, err := converter(ctx, md)
		if err != nil {
			logger.Error("Failed to convert metrics. Dropping data.",
				zap.Int("dropped_data_points", md.DataPointCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}
