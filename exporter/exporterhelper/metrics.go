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
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	metricsMarshaler   = &pmetric.ProtoMarshaler{}
	metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}
)

type metricsRequest struct {
	md               pmetric.Metrics
	pusher           consumer.ConsumeMetricsFunc
	cachedItemsCount int
}

func newMetricsRequest(md pmetric.Metrics, pusher consumer.ConsumeMetricsFunc) Request {
	return &metricsRequest{
		md:               md,
		pusher:           pusher,
		cachedItemsCount: md.DataPointCount(),
	}
}

type metricsEncoding struct {
	pusher consumer.ConsumeMetricsFunc
}

func (me *metricsEncoding) Unmarshal(bytes []byte) (Request, error) {
	metrics, err := metricsUnmarshaler.UnmarshalMetrics(bytes)
	if err != nil {
		return nil, err
	}
	return newMetricsRequest(metrics, me.pusher), nil
}

func (me *metricsEncoding) Marshal(req Request) ([]byte, error) {
	return metricsMarshaler.MarshalMetrics(req.(*metricsRequest).md)
}

func (req *metricsRequest) OnError(err error) Request {
	var metricsError consumererror.Metrics
	if errors.As(err, &metricsError) {
		return newMetricsRequest(metricsError.Data(), req.pusher)
	}
	return req
}

func (req *metricsRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.md)
}

func (req *metricsRequest) ItemsCount() int {
	return req.cachedItemsCount
}

func (req *metricsRequest) setCachedItemsCount(count int) {
	req.cachedItemsCount = count
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
		return nil, errNilPushMetricsData
	}
	return NewMetricsRequest(ctx, set, requestFromMetrics(pusher), append([]Option{internal.WithEncoding(&metricsEncoding{pusher: pusher})}, options...)...)
}

// RequestFromMetricsFunc converts pdata.Metrics into a user-defined request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromMetricsFunc func(context.Context, pmetric.Metrics) (Request, error)

// requestFromMetrics returns a RequestFromMetricsFunc that converts pdata.Metrics into a Request.
func requestFromMetrics(pusher consumer.ConsumeMetricsFunc) RequestFromMetricsFunc {
	return func(_ context.Context, md pmetric.Metrics) (Request, error) {
		return newMetricsRequest(md, pusher), nil
	}
}

// NewMetricsRequest creates a new metrics exporter based on a custom MetricsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestFromMetricsFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilMetricsConverter
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalMetrics, options...)
	if err != nil {
		return nil, err
	}

	mc, err := consumer.NewMetrics(newConsumeMetrics(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &metricsExporter{BaseExporter: be, Metrics: mc}, nil
}

func newConsumeMetrics(converter RequestFromMetricsFunc, be *internal.BaseExporter, logger *zap.Logger) consumer.ConsumeMetricsFunc {
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
