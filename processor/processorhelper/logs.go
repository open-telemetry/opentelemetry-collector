// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processorhelper // import "go.opentelemetry.io/collector/processor/processorhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
)

// ProcessLogsFunc is a helper function that processes the incoming data and returns the data to be sent to the next component.
// If error is returned then returned data are ignored. It MUST not call the next component.
type ProcessLogsFunc func(context.Context, plog.Logs) (plog.Logs, error)

type logProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Logs
}

// NewLogsProcessor creates a component.LogsProcessor that ensure context propagation and the right tags are set.
func NewLogsProcessor(
	_ context.Context,
	set processor.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Logs,
	logsFunc ProcessLogsFunc,
	options ...Option,
) (processor.Logs, error) {
	// TODO: Add observability metrics support
	if logsFunc == nil {
		return nil, errors.New("nil logsFunc")
	}

	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	eventOptions := spanAttributes(set.ID)
	bs := fromOptions(options)
	logsConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		span := trace.SpanFromContext(ctx)
		span.AddEvent("Start processing.", eventOptions)
		var err error
		ld, err = logsFunc(ctx, ld)
		span.AddEvent("End processing.", eventOptions)
		if err != nil {
			if errors.Is(err, ErrSkipProcessingData) {
				return nil
			}
			return err
		}
		return nextConsumer.ConsumeLogs(ctx, ld)
	}, bs.consumerOptions...)
	if err != nil {
		return nil, err
	}

	return &logProcessor{
		StartFunc:    bs.StartFunc,
		ShutdownFunc: bs.ShutdownFunc,
		Logs:         logsConsumer,
	}, nil
}
