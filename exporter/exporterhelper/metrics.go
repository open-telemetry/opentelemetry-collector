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
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pipeline"
)

var metricsMarshaler = &pmetric.ProtoMarshaler{}
var metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}

type metricsRequest struct {
	md     pmetric.Metrics
	pusher consumer.ConsumeMetricsFunc
}

func newMetricsRequest(md pmetric.Metrics, pusher consumer.ConsumeMetricsFunc) Request {
	return &metricsRequest{
		md:     md,
		pusher: pusher,
	}
}

func newMetricsRequestUnmarshalerFunc(pusher consumer.ConsumeMetricsFunc) exporterqueue.Unmarshaler[Request] {
	return func(bytes []byte) (Request, error) {
		metrics, err := metricsUnmarshaler.UnmarshalMetrics(bytes)
		if err != nil {
			return nil, err
		}
		return newMetricsRequest(metrics, pusher), nil
	}
}

func metricsRequestMarshaler(req Request) ([]byte, error) {
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
	return req.md.DataPointCount()
}

type metricsExporter struct {
	*internal.BaseExporter
	consumer.Metrics
}

// NewMetricsExporter creates an exporter.Metrics that records observability metrics and wraps every request with a Span.
func NewMetricsExporter(
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
	metricsOpts := []Option{
		internal.WithMarshaler(metricsRequestMarshaler), internal.WithUnmarshaler(newMetricsRequestUnmarshalerFunc(pusher)),
		internal.WithBatchFuncs(mergeMetrics, mergeSplitMetrics),
	}
	return NewMetricsRequestExporter(ctx, set, requestFromMetrics(pusher), append(metricsOpts, options...)...)
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

// NewMetricsRequestExporter creates a new metrics exporter based on a custom MetricsConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequestExporter(
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

	be, err := internal.NewBaseExporter(set, pipeline.SignalMetrics, newMetricsSenderWithObservability, options...)
	if err != nil {
		return nil, err
	}

	mc, err := consumer.NewMetrics(func(ctx context.Context, md pmetric.Metrics) error {
		req, cErr := converter(ctx, md)
		if cErr != nil {
			set.Logger.Error("Failed to convert metrics. Dropping data.",
				zap.Int("dropped_data_points", md.DataPointCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		sErr := be.Send(ctx, req)
		if errors.Is(sErr, queue.ErrQueueIsFull) {
			be.Obsrep.RecordEnqueueFailure(ctx, pipeline.SignalMetrics, int64(req.ItemsCount()))
		}
		return sErr
	}, be.ConsumerOptions...)

	return &metricsExporter{
		BaseExporter: be,
		Metrics:      mc,
	}, err
}

type metricsSenderWithObservability struct {
	internal.BaseRequestSender
	obsrep *internal.ObsReport
}

func newMetricsSenderWithObservability(obsrep *internal.ObsReport) internal.RequestSender {
	return &metricsSenderWithObservability{obsrep: obsrep}
}

func (mewo *metricsSenderWithObservability) Send(ctx context.Context, req Request) error {
	c := mewo.obsrep.StartMetricsOp(ctx)
	numMetricDataPoints := req.ItemsCount()
	err := mewo.NextSender.Send(c, req)
	mewo.obsrep.EndMetricsOp(c, numMetricDataPoints, err)
	return err
}
