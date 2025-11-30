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
	ld         plog.Logs
	cachedSize int
}

func newLogsRequest(ld plog.Logs) request.Request {
	return &logsRequest{
		ld:         ld,
		cachedSize: -1,
	}
}

type logsEncoding struct{}

var _ encoding[request.Request] = logsEncoding{}

func (logsEncoding) Unmarshal(bytes []byte) (context.Context, request.Request, error) {
	if queue.PersistRequestContextOnRead() {
		ctx, logs, err := pdatareq.UnmarshalLogs(bytes)
		if errors.Is(err, pdatareq.ErrInvalidFormat) {
			// fall back to unmarshaling without context
			logs, err = logsUnmarshaler.UnmarshalLogs(bytes)
		}
		return ctx, newLogsRequest(logs), err
	}

	logs, err := logsUnmarshaler.UnmarshalLogs(bytes)
	if err != nil {
		var req request.Request
		return context.Background(), req, err
	}
	return context.Background(), newLogsRequest(logs), nil
}

func (logsEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	logs := req.(*logsRequest).ld
	if queue.PersistRequestContextOnWrite() {
		return pdatareq.MarshalLogs(ctx, logs)
	}
	return logsMarshaler.MarshalLogs(logs)
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
	return req.ld.LogRecordCount()
}

func (req *logsRequest) size(sizer sizer.LogsSizer) int {
	if req.cachedSize == -1 {
		req.cachedSize = sizer.LogsSize(req.ld)
	}
	return req.cachedSize
}

func (req *logsRequest) setCachedSize(size int) {
	req.cachedSize = size
}

func (req *logsRequest) BytesSize() int {
	return logsMarshaler.LogsSize(req.ld)
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
