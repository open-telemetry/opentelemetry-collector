// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	pdatareq "go.opentelemetry.io/collector/pdata/xpdata/request"
)

var (
	metricsMarshaler   = &pmetric.ProtoMarshaler{}
	metricsUnmarshaler = &pmetric.ProtoUnmarshaler{}
)

func NewMetricsQueueBatchSettings() Settings[request.Request] {
	return Settings[request.Request]{
		ReferenceCounter: metricsReferenceCounter{},
		Encoding:         metricsEncoding{},
	}
}

var (
	_ request.Request      = (*metricsRequest)(nil)
	_ request.ErrorHandler = (*metricsRequest)(nil)
)

type metricsRequest struct {
	md         pmetric.Metrics
	cachedSize int
}

func newMetricsRequest(md pmetric.Metrics) request.Request {
	return &metricsRequest{
		md:         md,
		cachedSize: -1,
	}
}

type metricsEncoding struct{}

var _ encoding[request.Request] = metricsEncoding{}

func (metricsEncoding) Unmarshal(bytes []byte) (context.Context, request.Request, error) {
	if queue.PersistRequestContextOnRead() {
		ctx, metrics, err := pdatareq.UnmarshalMetrics(bytes)
		if errors.Is(err, pdatareq.ErrInvalidFormat) {
			// fall back to unmarshaling without context
			metrics, err = metricsUnmarshaler.UnmarshalMetrics(bytes)
		}
		return ctx, newMetricsRequest(metrics), err
	}
	metrics, err := metricsUnmarshaler.UnmarshalMetrics(bytes)
	if err != nil {
		var req request.Request
		return context.Background(), req, err
	}
	return context.Background(), newMetricsRequest(metrics), nil
}

func (metricsEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	metrics := req.(*metricsRequest).md
	if queue.PersistRequestContextOnWrite() {
		return pdatareq.MarshalMetrics(ctx, metrics)
	}
	return metricsMarshaler.MarshalMetrics(metrics)
}

var _ queue.ReferenceCounter[request.Request] = metricsReferenceCounter{}

type metricsReferenceCounter struct{}

func (metricsReferenceCounter) Ref(req request.Request) {
	pref.RefMetrics(req.(*metricsRequest).md)
}

func (metricsReferenceCounter) Unref(req request.Request) {
	pref.UnrefMetrics(req.(*metricsRequest).md)
}

func (req *metricsRequest) OnError(err error) request.Request {
	var metricsError consumererror.Metrics
	if errors.As(err, &metricsError) {
		// TODO: Add logic to unref the new request created here.
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

func (req *metricsRequest) BytesSize() int {
	return metricsMarshaler.MetricsSize(req.md)
}

// RequestFromMetrics returns a RequestFromMetricsFunc that converts pdata.Metrics into a Request.
func RequestFromMetrics() request.RequestConverterFunc[pmetric.Metrics] {
	return func(_ context.Context, md pmetric.Metrics) (request.Request, error) {
		return newMetricsRequest(md), nil
	}
}

// RequestConsumeFromMetrics returns a RequestConsumeFunc that consumes pmetric.Metrics.
func RequestConsumeFromMetrics(pusher consumer.ConsumeMetricsFunc) request.RequestConsumeFunc {
	return func(ctx context.Context, request request.Request) error {
		return pusher.ConsumeMetrics(ctx, request.(*metricsRequest).md)
	}
}
