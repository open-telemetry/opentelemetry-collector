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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	logsMarshaler   = &plog.ProtoMarshaler{}
	logsUnmarshaler = &plog.ProtoUnmarshaler{}
)

type logsRequest struct {
	ld         plog.Logs
	pusher     consumer.ConsumeLogsFunc
	cachedSize int
}

func newLogsRequest(ld plog.Logs, pusher consumer.ConsumeLogsFunc) Request {
	return &logsRequest{
		ld:         ld,
		pusher:     pusher,
		cachedSize: -1,
	}
}

type logsEncoding struct {
	pusher consumer.ConsumeLogsFunc
}

func (le *logsEncoding) Unmarshal(bytes []byte) (Request, error) {
	logs, err := logsUnmarshaler.UnmarshalLogs(bytes)
	if err != nil {
		return nil, err
	}
	return newLogsRequest(logs, le.pusher), nil
}

func (le *logsEncoding) Marshal(req Request) ([]byte, error) {
	return logsMarshaler.MarshalLogs(req.(*logsRequest).ld)
}

func (req *logsRequest) OnError(err error) Request {
	var logError consumererror.Logs
	if errors.As(err, &logError) {
		return newLogsRequest(logError.Data(), req.pusher)
	}
	return req
}

func (req *logsRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.ld)
}

func (req *logsRequest) ItemsCount() int {
	return req.ld.LogRecordCount()
}

func (req *logsRequest) Size(sizer sizer.LogsSizer) int {
	if req.cachedSize == -1 {
		req.cachedSize = sizer.LogsSize(req.ld)
	}
	return req.cachedSize
}

func (req *logsRequest) setCachedSize(size int) {
	req.cachedSize = size
}

type logsExporter struct {
	*internal.BaseExporter
	consumer.Logs
}

// NewLogs creates an exporter.Logs that records observability logs and wraps every request with a Span.
func NewLogs(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeLogsFunc,
	options ...Option,
) (exporter.Logs, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushLogsData
	}
	return NewLogsRequest(ctx, set, requestFromLogs(pusher), append([]Option{internal.WithEncoding(&logsEncoding{pusher: pusher})}, options...)...)
}

// RequestFromLogsFunc converts plog.Logs data into a user-defined request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromLogsFunc func(context.Context, plog.Logs) (Request, error)

// requestFromLogs returns a RequestFromLogsFunc that converts plog.Logs into a Request.
func requestFromLogs(pusher consumer.ConsumeLogsFunc) RequestFromLogsFunc {
	return func(_ context.Context, ld plog.Logs) (Request, error) {
		return newLogsRequest(ld, pusher), nil
	}
}

// NewLogsRequest creates new logs exporter based on custom LogsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestFromLogsFunc,
	options ...Option,
) (exporter.Logs, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilLogsConverter
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalLogs, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewLogs(newConsumeLogs(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &logsExporter{BaseExporter: be, Logs: lc}, nil
}

func newConsumeLogs(converter RequestFromLogsFunc, be *internal.BaseExporter, logger *zap.Logger) consumer.ConsumeLogsFunc {
	return func(ctx context.Context, ld plog.Logs) error {
		req, err := converter(ctx, ld)
		if err != nil {
			logger.Error("Failed to convert logs. Dropping data.",
				zap.Int("dropped_log_records", ld.LogRecordCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}
