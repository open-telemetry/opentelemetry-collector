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
	intrequest "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var metricsMarshaler = &pmetric.ProtoMarshaler{}
var metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}

type metricsRequest struct {
	md     pmetric.Metrics
	pusher consumer.ConsumeMetricsFunc
}

func newMetricsRequest(md pmetric.Metrics, pusher consumer.ConsumeMetricsFunc) request.Request {
	return &metricsRequest{
		md:     md,
		pusher: pusher,
	}
}

func newMetricsRequestUnmarshalerFunc(pusher consumer.ConsumeMetricsFunc) request.Unmarshaler {
	return func(bytes []byte) (request.Request, error) {
		metrics, err := metricsUnmarshaler.UnmarshalMetrics(bytes)
		if err != nil {
			return nil, err
		}
		return newMetricsRequest(metrics, pusher), nil
	}
}

func metricsRequestMarshaler(req request.Request) ([]byte, error) {
	return metricsMarshaler.MarshalMetrics(req.(*metricsRequest).md)
}

func (req *metricsRequest) OnError(err error) request.Request {
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
		req := intrequest.New(ctx, newMetricsRequest(md, pusher))
		serr := be.send(req)
		if errors.Is(serr, errSendingQueueIsFull) {
			be.obsrep.recordEnqueueFailure(req.Context(), component.DataTypeMetrics, int64(req.Count()))
		}
		return serr
	}, be.consumerOptions...)

	return &metricsExporter{
		baseExporter: be,
		Metrics:      mc,
	}, err
}

// MetricsConverter provides an interface for converting pmetric.Metrics into a request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type MetricsConverter interface {
	// RequestFromMetrics converts pdata.Metrics into a request.
	RequestFromMetrics(context.Context, pmetric.Metrics) (request.Request, error)
}

// NewMetricsRequestExporter creates a new metrics exporter based on a custom MetricsConverter and RequestSender.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewMetricsRequestExporter(
	_ context.Context,
	set exporter.CreateSettings,
	converter MetricsConverter,
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
		req, cErr := converter.RequestFromMetrics(ctx, md)
		if cErr != nil {
			set.Logger.Error("Failed to convert metrics. Dropping data.",
				zap.Int("dropped_data_points", md.DataPointCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		r := intrequest.New(ctx, req)
		sErr := be.send(r)
		if errors.Is(sErr, errSendingQueueIsFull) {
			be.obsrep.recordEnqueueFailure(r.Context(), component.DataTypeMetrics, int64(r.Count()))
		}
		return sErr
	}, be.consumerOptions...)

	return &metricsExporter{
		baseExporter: be,
		Metrics:      mc,
	}, err
}

type metricsSenderWithObservability struct {
	obsrep     *ObsReport
	nextSender *senderWrapper
}

func newMetricsSenderWithObservability(obsrep *ObsReport, nextSender *senderWrapper) requestSender {
	return &metricsSenderWithObservability{obsrep: obsrep, nextSender: nextSender}
}

func (mewo *metricsSenderWithObservability) send(req *intrequest.Request) error {
	req.SetContext(mewo.obsrep.StartMetricsOp(req.Context()))
	err := mewo.nextSender.send(req)
	mewo.obsrep.EndMetricsOp(req.Context(), req.Count(), err)
	return err
}
