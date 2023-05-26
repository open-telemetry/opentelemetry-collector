// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

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
	internal.BaseRequest
	md     pmetric.Metrics
	pusher consumer.ConsumeMetricsFunc
}

func newMetricsRequest(ctx context.Context, md pmetric.Metrics, pusher consumer.ConsumeMetricsFunc) internal.Request {
	return &metricsRequest{
		BaseRequest: internal.BaseRequest{Ctx: ctx},
		md:          md,
		pusher:      pusher,
	}
}

func newMetricsRequestUnmarshalerFunc(pusher consumer.ConsumeMetricsFunc) internal.RequestUnmarshaler {
	return func(bytes []byte) (internal.Request, error) {
		metrics, err := metricsUnmarshaler.UnmarshalMetrics(bytes)
		if err != nil {
			return nil, err
		}
		return newMetricsRequest(context.Background(), metrics, pusher), nil
	}
}

func (req *metricsRequest) OnError(err error) internal.Request {
	var metricsError consumererror.Metrics
	if errors.As(err, &metricsError) {
		return newMetricsRequest(req.Ctx, metricsError.Data(), req.pusher)
	}
	return req
}

func (req *metricsRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.md)
}

// Marshal provides serialization capabilities required by persistent queue
func (req *metricsRequest) Marshal() ([]byte, error) {
	return metricsMarshaler.MarshalMetrics(req.md)
}

func (req *metricsRequest) Count() int {
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

	bs := fromOptions(options...)
	be, err := newBaseExporter(set, bs, component.DataTypeMetrics, newMetricsRequestUnmarshalerFunc(pusher))
	if err != nil {
		return nil, err
	}
	be.wrapConsumerSender(func(nextSender internal.RequestSender) internal.RequestSender {
		return &metricsSenderWithObservability{
			obsrep:     be.obsrep,
			nextSender: nextSender,
		}
	})

	mc, err := consumer.NewMetrics(func(ctx context.Context, md pmetric.Metrics) error {
		req := newMetricsRequest(ctx, md, pusher)
		serr := be.sender.Send(req)
		if errors.Is(serr, errSendingQueueIsFull) {
			be.obsrep.recordMetricsEnqueueFailure(req.Context(), int64(req.Count()))
		}
		return serr
	}, bs.consumerOptions...)

	return &metricsExporter{
		baseExporter: be,
		Metrics:      mc,
	}, err
}

type metricsSenderWithObservability struct {
	obsrep     *obsExporter
	nextSender internal.RequestSender
}

func (mewo *metricsSenderWithObservability) Send(req internal.Request) error {
	req.SetContext(mewo.obsrep.StartMetricsOp(req.Context()))
	err := mewo.nextSender.Send(req)
	mewo.obsrep.EndMetricsOp(req.Context(), req.Count(), err)
	return err
}
