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

package processorhelper

import (
	"context"
	"errors"

	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerhelper"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// LProcessor is a helper interface that allows avoiding implementing all functions in LogsProcessor by using NewLogsProcessor.
type LProcessor interface {
	// ProcessLogs is a helper function that processes the incoming data and returns the data to be sent to the next component.
	// If error is returned then returned data are ignored. It MUST not call the next component.
	ProcessLogs(context.Context, pdata.Logs) (pdata.Logs, error)
}

type logProcessor struct {
	component.Component
	consumer.Logs
}

// NewLogsProcessor creates a LogsProcessor that ensure context propagation and the right tags are set.
// TODO: Add observability metrics support
func NewLogsProcessor(
	cfg config.Processor,
	nextConsumer consumer.Logs,
	processor LProcessor,
	options ...Option,
) (component.LogsProcessor, error) {
	if processor == nil {
		return nil, errors.New("nil processor")
	}

	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	traceAttributes := spanAttributes(cfg.ID())
	bs := fromOptions(options)
	logsConsumer, err := consumerhelper.NewLogs(func(ctx context.Context, ld pdata.Logs) error {
		span := trace.FromContext(ctx)
		span.Annotate(traceAttributes, "Start processing.")
		var err error
		ld, err = processor.ProcessLogs(ctx, ld)
		span.Annotate(traceAttributes, "End processing.")
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
		Component: componenthelper.New(bs.componentOptions...),
		Logs:      logsConsumer,
	}, nil
}
