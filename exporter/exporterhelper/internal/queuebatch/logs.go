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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	pdatareq "go.opentelemetry.io/collector/pdata/xpdata/request"
)

var (
	logsMarshaler   = &plog.ProtoMarshaler{}
	logsUnmarshaler = &plog.ProtoUnmarshaler{}
)

// NewLogsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using plog.Logs.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsQueueBatchSettings() Settings[request.Request] {
	return Settings[request.Request]{
		ReferenceCounter: logsReferenceCounter{},
		Encoding:         logsEncoding{},
	}
}

var (
	_ request.Request      = (*logsRequest)(nil)
	_ request.ErrorHandler = (*logsRequest)(nil)
)

type logsRequest struct {
	ld plog.Logs
	// Sizes are cached per sizer type: the batcher and the queue may ask for
	// bytes and items on the same request, and a single cache would return a
	// value computed for the wrong sizer. -1 means "not yet computed".
	cachedItemsSize int
	cachedBytesSize int
}

func newLogsRequest(ld plog.Logs) request.Request {
	return &logsRequest{
		ld:              ld,
		cachedItemsSize: -1,
		cachedBytesSize: -1,
	}
}

type logsEncoding struct{}

var _ encoding[request.Request] = logsEncoding{}

func (logsEncoding) Unmarshal(bytes []byte) (context.Context, request.Request, error) {
	ctx, logs, err := pdatareq.UnmarshalLogs(bytes)
	if errors.Is(err, pdatareq.ErrInvalidFormat) {
		// fall back to unmarshaling without context
		logs, err = logsUnmarshaler.UnmarshalLogs(bytes)
	}
	return ctx, newLogsRequest(logs), err
}

func (logsEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	return pdatareq.MarshalLogs(ctx, req.(*logsRequest).ld)
}

var _ queue.ReferenceCounter[request.Request] = logsReferenceCounter{}

type logsReferenceCounter struct{}

func (logsReferenceCounter) Ref(req request.Request) {
	pref.RefLogs(req.(*logsRequest).ld)
}

func (logsReferenceCounter) Unref(req request.Request) {
	pref.UnrefLogs(req.(*logsRequest).ld)
}

func (req *logsRequest) OnError(err error) request.Request {
	var logError consumererror.Logs
	if errors.As(err, &logError) {
		// TODO: Add logic to unref the new request created here.
		return newLogsRequest(logError.Data())
	}
	return req
}

func (req *logsRequest) ItemsCount() int {
	if req.cachedItemsSize < 0 {
		req.cachedItemsSize = req.ld.LogRecordCount()
	}
	return req.cachedItemsSize
}

func (req *logsRequest) size(sz sizer.LogsSizer) int {
	switch sz.(type) {
	case *sizer.LogsCountSizer:
		if req.cachedItemsSize < 0 {
			req.cachedItemsSize = sz.LogsSize(req.ld)
		}
		return req.cachedItemsSize
	case *sizer.LogsBytesSizer:
		if req.cachedBytesSize < 0 {
			req.cachedBytesSize = sz.LogsSize(req.ld)
		}
		return req.cachedBytesSize
	default:
		return sz.LogsSize(req.ld)
	}
}

// setCachedSize records the size for sz's dimension and invalidates the other,
// which the caller (mergeTo/split) did not maintain. This keeps the dimension
// used by the batcher (a single configured sizer) O(1) across merges while
// never returning a stale value for the other dimension.
func (req *logsRequest) setCachedSize(sz sizer.LogsSizer, size int) {
	switch sz.(type) {
	case *sizer.LogsCountSizer:
		req.cachedItemsSize = size
		req.cachedBytesSize = -1
	case *sizer.LogsBytesSizer:
		req.cachedBytesSize = size
		req.cachedItemsSize = -1
	}
}

func (req *logsRequest) BytesSize() int {
	if req.cachedBytesSize < 0 {
		req.cachedBytesSize = logsMarshaler.LogsSize(req.ld)
	}
	return req.cachedBytesSize
}

// RequestConsumeFromLogs returns a RequestConsumeFunc that consumes plog.Logs.
func RequestConsumeFromLogs(pusher consumer.ConsumeLogsFunc) request.RequestConsumeFunc {
	return func(ctx context.Context, request request.Request) error {
		return pusher.ConsumeLogs(ctx, request.(*logsRequest).ld)
	}
}

// RequestFromLogs returns a RequestFromLogsFunc that converts plog.Logs into a Request.
func RequestFromLogs() request.RequestConverterFunc[plog.Logs] {
	return func(_ context.Context, ld plog.Logs) (request.Request, error) {
		return newLogsRequest(ld), nil
	}
}
