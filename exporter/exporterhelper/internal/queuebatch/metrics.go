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
	md    pmetric.Metrics
	sizes request.SizeCache
}

func newMetricsRequest(md pmetric.Metrics) request.Request {
	return &metricsRequest{
		md:    md,
		sizes: request.NewSizeCache(),
	}
}

type metricsEncoding struct{}

var _ encoding[request.Request] = metricsEncoding{}

func (metricsEncoding) Unmarshal(bytes []byte) (context.Context, request.Request, error) {
	ctx, metrics, err := pdatareq.UnmarshalMetrics(bytes)
	if errors.Is(err, pdatareq.ErrInvalidFormat) {
		// fall back to unmarshaling without context
		metrics, err = metricsUnmarshaler.UnmarshalMetrics(bytes)
	}
	if err == nil {
		// Rehydrated data from Unmarshal has no upstream ownership claim yet.
		// This marks the bit so a downstream refconsumer boundary won't also claim/release this object's ref.
		pref.MarkPipelineOwnedMetrics(metrics)
	}
	return ctx, newMetricsRequest(metrics), err
}

func (metricsEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	return pdatareq.MarshalMetrics(ctx, req.(*metricsRequest).md)
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
	if metricsError, ok := errors.AsType[consumererror.Metrics](err); ok {
		// TODO: Add logic to unref the new request created here.
		return newMetricsRequest(metricsError.Data())
	}
	return req
}

func (req *metricsRequest) ItemsCount() int {
	return req.sizes.SizeOf(request.SizerTypeItems, func() int { return req.md.DataPointCount() })
}

func (req *metricsRequest) size(sz sizer.MetricsSizer, szt request.SizerType) int {
	return req.sizes.SizeOf(szt, func() int { return sz.MetricsSize(req.md) })
}

func (req *metricsRequest) BytesSize() int {
	return req.sizes.SizeOf(request.SizerTypeBytes, func() int { return metricsMarshaler.MetricsSize(req.md) })
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
