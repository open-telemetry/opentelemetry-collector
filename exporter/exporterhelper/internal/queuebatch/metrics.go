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
	md pmetric.Metrics
	// Sizes are cached per sizer type: the batcher and the queue may ask for
	// bytes and items on the same request, and a single cache would return a
	// value computed for the wrong sizer. -1 means "not yet computed".
	cachedItemsSize int
	cachedBytesSize int
}

func newMetricsRequest(md pmetric.Metrics) request.Request {
	return &metricsRequest{
		md:              md,
		cachedItemsSize: -1,
		cachedBytesSize: -1,
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
	var metricsError consumererror.Metrics
	if errors.As(err, &metricsError) {
		// TODO: Add logic to unref the new request created here.
		return newMetricsRequest(metricsError.Data())
	}
	return req
}

func (req *metricsRequest) ItemsCount() int {
	if req.cachedItemsSize < 0 {
		req.cachedItemsSize = req.md.DataPointCount()
	}
	return req.cachedItemsSize
}

func (req *metricsRequest) size(sz sizer.MetricsSizer) int {
	switch sz.(type) {
	case *sizer.MetricsCountSizer:
		if req.cachedItemsSize < 0 {
			req.cachedItemsSize = sz.MetricsSize(req.md)
		}
		return req.cachedItemsSize
	case *sizer.MetricsBytesSizer:
		if req.cachedBytesSize < 0 {
			req.cachedBytesSize = sz.MetricsSize(req.md)
		}
		return req.cachedBytesSize
	default:
		return sz.MetricsSize(req.md)
	}
}

// setCachedSize records the size for sz's dimension and invalidates the other,
// which the caller (mergeTo/split) did not maintain. This keeps the dimension
// used by the batcher (a single configured sizer) O(1) across merges while
// never returning a stale value for the other dimension.
func (req *metricsRequest) setCachedSize(sz sizer.MetricsSizer, size int) {
	switch sz.(type) {
	case *sizer.MetricsCountSizer:
		req.cachedItemsSize = size
		req.cachedBytesSize = -1
	case *sizer.MetricsBytesSizer:
		req.cachedBytesSize = size
		req.cachedItemsSize = -1
	}
}

func (req *metricsRequest) BytesSize() int {
	if req.cachedBytesSize < 0 {
		req.cachedBytesSize = metricsMarshaler.MetricsSize(req.md)
	}
	return req.cachedBytesSize
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
