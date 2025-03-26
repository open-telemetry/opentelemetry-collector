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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	logsMarshaler   = &plog.ProtoMarshaler{}
	logsUnmarshaler = &plog.ProtoUnmarshaler{}
)

// NewLogsQueueBatchSettings returns a new QueueBatchSettings to configure to WithQueueBatch when using plog.Logs.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsQueueBatchSettings() QueueBatchSettings {
	return QueueBatchSettings{
		Encoding: logsEncoding{},
		Sizers: map[RequestSizerType]request.Sizer[Request]{
			RequestSizerTypeRequests: NewRequestsSizer(),
			RequestSizerTypeItems:    request.NewItemsSizer(),
			RequestSizerTypeBytes: request.BaseSizer{
				SizeofFunc: func(req request.Request) int64 {
					return int64(logsMarshaler.LogsSize(req.(*logsRequest).ld))
				},
			},
		},
	}
}

type logsRequest struct {
	ld          plog.Logs
	cachedItems int
	cachedBytes int
}

func newLogsRequest(ld plog.Logs) Request {
	return &logsRequest{
		ld:          ld,
		cachedItems: ld.LogRecordCount(),
		cachedBytes: -1,
	}
}

type logsEncoding struct{}

func (logsEncoding) Unmarshal(bytes []byte) (Request, error) {
	logs, err := logsUnmarshaler.UnmarshalLogs(bytes)
	if err != nil {
		return nil, err
	}
	return newLogsRequest(logs), nil
}

func (logsEncoding) Marshal(req Request) ([]byte, error) {
	return logsMarshaler.MarshalLogs(req.(*logsRequest).ld)
}

func (req *logsRequest) OnError(err error) Request {
	var logError consumererror.Logs
	if errors.As(err, &logError) {
		return newLogsRequest(logError.Data())
	}
	return req
}

func (req *logsRequest) ItemsCount() int {
	return req.cachedItems
}

func (req *logsRequest) ByteSize() int {
	if req.cachedBytes == -1 {
		sz := sizer.LogsBytesSizer{}
		req.cachedBytes = sz.LogsSize(req.ld)
	}
	return req.cachedBytes
}

// sizeFromSizer returns the size of the request based on the sizer. If the size is unknown, it returns 0.
func (req *logsRequest) sizeFromSizer(sz sizer.LogsSizer) int {
	switch sz.(type) {
	case *sizer.LogsCountSizer:
		return req.ItemsCount()
	case *sizer.LogsBytesSizer:
		return req.ByteSize()
	default:
		return 0
	}
}

func (req *logsRequest) setCachedSize(sz sizer.LogsSizer, size int) {
	switch sz.(type) {
	case *sizer.LogsCountSizer:
		req.cachedItems = size
	case *sizer.LogsBytesSizer:
		req.cachedBytes = size
	default:
	}
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
		return nil, errNilPushLogs
	}
	return NewLogsRequest(ctx, set, requestFromLogs(), requestConsumeFromLogs(pusher),
		append([]Option{internal.WithQueueBatchSettings(NewLogsQueueBatchSettings())}, options...)...)
}

// requestConsumeFromLogs returns a RequestConsumeFunc that consumes plog.Logs.
func requestConsumeFromLogs(pusher consumer.ConsumeLogsFunc) RequestConsumeFunc {
	return func(ctx context.Context, request Request) error {
		return pusher.ConsumeLogs(ctx, request.(*logsRequest).ld)
	}
}

// requestFromLogs returns a RequestFromLogsFunc that converts plog.Logs into a Request.
func requestFromLogs() RequestConverterFunc[plog.Logs] {
	return func(_ context.Context, ld plog.Logs) (Request, error) {
		return newLogsRequest(ld), nil
	}
}

// NewLogsRequest creates new logs exporter based on custom LogsConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestConverterFunc[plog.Logs],
	pusher RequestConsumeFunc,
	options ...Option,
) (exporter.Logs, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilLogsConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalLogs, pusher, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewLogs(newConsumeLogs(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &logsExporter{BaseExporter: be, Logs: lc}, nil
}

func newConsumeLogs(converter RequestConverterFunc[plog.Logs], be *internal.BaseExporter, logger *zap.Logger) consumer.ConsumeLogsFunc {
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
