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
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
)

var logsMarshaler = &plog.ProtoMarshaler{}
var logsUnmarshaler = &plog.ProtoUnmarshaler{}

type logsRequest struct {
	ld     plog.Logs
	pusher consumer.ConsumeLogsFunc
}

func newLogsRequest(ld plog.Logs, pusher consumer.ConsumeLogsFunc) Request {
	return &logsRequest{
		ld:     ld,
		pusher: pusher,
	}
}

func newLogsRequestUnmarshalerFunc(pusher consumer.ConsumeLogsFunc) exporterqueue.Unmarshaler[Request] {
	return func(bytes []byte) (Request, error) {
		logs, err := logsUnmarshaler.UnmarshalLogs(bytes)
		if err != nil {
			return nil, err
		}
		return newLogsRequest(logs, pusher), nil
	}
}

func logsRequestMarshaler(req Request) ([]byte, error) {
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

type logsExporter struct {
	*internal.BaseExporter
	consumer.Logs
}

// NewLogsExporter creates an exporter.Logs that records observability metrics and wraps every request with a Span.
func NewLogsExporter(
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
	logsOpts := []Option{
		internal.WithMarshaler(logsRequestMarshaler), internal.WithUnmarshaler(newLogsRequestUnmarshalerFunc(pusher)),
		internal.WithBatchFuncs(mergeLogs, mergeSplitLogs),
	}
	return NewLogsRequestExporter(ctx, set, requestFromLogs(pusher), append(logsOpts, options...)...)
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

// NewLogsRequestExporter creates new logs exporter based on custom LogsConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsRequestExporter(
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

	be, err := internal.NewBaseExporter(set, pipeline.SignalLogs, newLogsExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		req, cErr := converter(ctx, ld)
		if cErr != nil {
			set.Logger.Error("Failed to convert logs. Dropping data.",
				zap.Int("dropped_log_records", ld.LogRecordCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		sErr := be.Send(ctx, req)
		if errors.Is(sErr, queue.ErrQueueIsFull) {
			be.Obsrep.RecordEnqueueFailure(ctx, pipeline.SignalLogs, int64(req.ItemsCount()))
		}
		return sErr
	}, be.ConsumerOptions...)

	return &logsExporter{
		BaseExporter: be,
		Logs:         lc,
	}, err
}

type logsExporterWithObservability struct {
	internal.BaseRequestSender
	obsrep *internal.ObsReport
}

func newLogsExporterWithObservability(obsrep *internal.ObsReport) internal.RequestSender {
	return &logsExporterWithObservability{obsrep: obsrep}
}

func (lewo *logsExporterWithObservability) Send(ctx context.Context, req Request) error {
	c := lewo.obsrep.StartLogsOp(ctx)
	numLogRecords := req.ItemsCount()
	err := lewo.NextSender.Send(c, req)
	lewo.obsrep.EndLogsOp(c, numLogRecords, err)
	return err
}
