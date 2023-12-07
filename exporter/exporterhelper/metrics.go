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

func newMetricsRequestUnmarshalerFunc(pusher consumer.ConsumeMetricsFunc) RequestUnmarshaler {
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
	*baseExporter
	consumer.Metrics
}

// NewMetricsExporter creates an exporter.Metrics that records observability metrics and wraps every request with a Span.
func NewMetricsExporter(
	_ context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
	pusher consumer.ConsumeMetricsFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if set.Logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushMetricsData
	}

	be, err := newBaseExporter(set, component.DataTypeMetrics, false, metricsRequestMarshaler,
		newMetricsRequestUnmarshalerFunc(pusher), newMetricsSenderWithObservability, options...)
	if err != nil {
		return nil, err
	}

	mc, err := consumer.NewMetrics(func(ctx context.Context, md pmetric.Metrics) error {
		req := newMetricsRequest(md, pusher)
		serr := be.send(ctx, req)
		if errors.Is(serr, internal.ErrQueueIsFull) {
			be.obsrep.recordEnqueueFailure(ctx, component.DataTypeMetrics, int64(req.ItemsCount()))
		}
		return serr
	}, be.consumerOptions...)

	return &metricsExporter{
		baseExporter: be,
		Metrics:      mc,
	}, err
}

// RequestFromMetricsFunc converts pdata.Metrics into a user-defined request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromMetricsFunc func(context.Context, pmetric.Metrics) (Request, error)

// NewMetricsRequestExporter creates a new metrics exporter based on a custom MetricsConverter and RequestSender.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequestExporter(
	_ context.Context,
	set exporter.CreateSettings,
	converter RequestFromMetricsFunc,
	options ...Option,
) (exporter.Metrics, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilMetricsConverter
	}

	be, err := newBaseExporter(set, component.DataTypeMetrics, true, nil, nil, newMetricsSenderWithObservability, options...)
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
		sErr := be.send(ctx, req)
		if errors.Is(sErr, internal.ErrQueueIsFull) {
			be.obsrep.recordEnqueueFailure(ctx, component.DataTypeMetrics, int64(req.ItemsCount()))
		}
		return sErr
	}, be.consumerOptions...)

	return &metricsExporter{
		baseExporter: be,
		Metrics:      mc,
	}, err
}

type metricsSenderWithObservability struct {
	baseRequestSender
	obsrep *ObsReport
}

func newMetricsSenderWithObservability(obsrep *ObsReport) requestSender {
	return &metricsSenderWithObservability{obsrep: obsrep}
}

func (mewo *metricsSenderWithObservability) send(ctx context.Context, req Request) error {
	c := mewo.obsrep.StartMetricsOp(ctx)
	err := mewo.nextSender.send(c, req)
	mewo.obsrep.EndMetricsOp(c, req.ItemsCount(), err)
	return err
}
