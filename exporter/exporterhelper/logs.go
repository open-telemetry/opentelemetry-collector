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
	"go.opentelemetry.io/collector/pdata/plog"
)

var logsMarshaler = &plog.ProtoMarshaler{}
var logsUnmarshaler = &plog.ProtoUnmarshaler{}

type logsRequest struct {
	baseRequest
	ld     plog.Logs
	pusher consumer.ConsumeLogsFunc
}

func newLogsRequest(ctx context.Context, ld plog.Logs, pusher consumer.ConsumeLogsFunc) internal.Request {
	return &logsRequest{
		baseRequest: baseRequest{ctx: ctx},
		ld:          ld,
		pusher:      pusher,
	}
}

func newLogsRequestUnmarshalerFunc(pusher consumer.ConsumeLogsFunc) internal.RequestUnmarshaler {
	return func(bytes []byte) (internal.Request, error) {
		logs, err := logsUnmarshaler.UnmarshalLogs(bytes)
		if err != nil {
			return nil, err
		}
		return newLogsRequest(context.Background(), logs, pusher), nil
	}
}

func logsRequestMarshaler(req internal.Request) ([]byte, error) {
	return logsMarshaler.MarshalLogs(req.(*logsRequest).ld)
}

func (req *logsRequest) OnError(err error) internal.Request {
	var logError consumererror.Logs
	if errors.As(err, &logError) {
		return newLogsRequest(req.ctx, logError.Data(), req.pusher)
	}
	return req
}

func (req *logsRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.ld)
}

func (req *logsRequest) Count() int {
	return req.ld.LogRecordCount()
}

type logsExporter struct {
	*baseExporter
	consumer.Logs
}

// NewLogsExporter creates an exporter.Logs that records observability metrics and wraps every request with a Span.
func NewLogsExporter(
	_ context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
	pusher consumer.ConsumeLogsFunc,
	options ...Option,
) (exporter.Logs, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if set.Logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushLogsData
	}

	be, err := newBaseExporter(set, component.DataTypeLogs, false, logsRequestMarshaler,
		newLogsRequestUnmarshalerFunc(pusher), newLogsExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		req := newLogsRequest(ctx, ld, pusher)
		serr := be.send(req)
		if errors.Is(serr, errSendingQueueIsFull) {
			be.obsrep.recordEnqueueFailure(req.Context(), component.DataTypeLogs, int64(req.Count()))
		}
		return serr
	}, be.consumerOptions...)

	return &logsExporter{
		baseExporter: be,
		Logs:         lc,
	}, err
}

// LogsConverter provides an interface for converting plog.Logs into a request.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type LogsConverter interface {
	// RequestFromLogs converts plog.Logs data into a request.
	RequestFromLogs(context.Context, plog.Logs) (Request, error)
}

// NewLogsRequestExporter creates new logs exporter based on custom LogsConverter and RequestSender.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewLogsRequestExporter(
	_ context.Context,
	set exporter.CreateSettings,
	converter LogsConverter,
	options ...Option,
) (exporter.Logs, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilLogsConverter
	}

	be, err := newBaseExporter(set, component.DataTypeLogs, true, nil, nil, newLogsExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		req, cErr := converter.RequestFromLogs(ctx, ld)
		if cErr != nil {
			set.Logger.Error("Failed to convert logs. Dropping data.",
				zap.Int("dropped_log_records", ld.LogRecordCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		r := newRequest(ctx, req)
		sErr := be.send(r)
		if errors.Is(sErr, errSendingQueueIsFull) {
			be.obsrep.recordEnqueueFailure(r.Context(), component.DataTypeLogs, int64(r.Count()))
		}
		return sErr
	}, be.consumerOptions...)

	return &logsExporter{
		baseExporter: be,
		Logs:         lc,
	}, err
}

type logsExporterWithObservability struct {
	baseRequestSender
	obsrep *ObsReport
}

func newLogsExporterWithObservability(obsrep *ObsReport) requestSender {
	return &logsExporterWithObservability{obsrep: obsrep}
}

func (lewo *logsExporterWithObservability) send(req internal.Request) error {
	req.SetContext(lewo.obsrep.StartLogsOp(req.Context()))
	err := lewo.nextSender.send(req)
	lewo.obsrep.EndLogsOp(req.Context(), req.Count(), err)
	return err
}
