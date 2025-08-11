// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
)

// ProcessLogsFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessLogsFunc func(context.Context, plog.Logs) (plog.Logs, error)

type logs struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Logs
}

// NewLogs creates a processor.Logs that ensure context propagation and the right tags are set.
func NewLogs(
	_ context.Context,
	set processor.Settings,
	_ component.Config,
	nextConsumer consumer.Logs,
	logsFunc ProcessLogsFunc,
	options ...Option,
) (processor.Logs, error) {
	if logsFunc == nil {
		return nil, errors.New("nil logsFunc")
	}

	obs, err := newObsReport(set, pipeline.SignalLogs)
	if err != nil {
		return nil, err
	}

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	logsConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)

		startTime := time.Now()

		recordsIn := ld.LogRecordCount()

		var errFunc error
		ld, errFunc = logsFunc(ctx, ld)
		obs.recordInternalDuration(ctx, startTime)
		span.AddEvent("End processing.", eventOptions)
		if errFunc != nil {
			obs.recordInOut(ctx, recordsIn, 0)
			if errors.Is(errFunc, ErrSkipProcessingData) {
				return nil
			}
			return errFunc
		}
		recordsOut := ld.LogRecordCount()
		obs.recordInOut(ctx, recordsIn, recordsOut)
		return nextConsumer.ConsumeLogs(ctx, ld)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &logs{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Logs:         logsConsumer,
	}, nil
}
