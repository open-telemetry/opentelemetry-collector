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
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// ErrSkipProcessingData is a sentinel value to indicate when traces or metrics should intentionally be dropped
// from further processing in the pipeline because the data is determined to be irrelevant. A processor can return this error
// to stop further processing without propagating an error back up the pipeline to logs.
var ErrSkipProcessingData = errors.New("sentinel error to skip processing data from the remainder of the pipeline")

// TProcessor is a helper interface that allows avoiding implementing all functions in TracesProcessor by using NewTracesProcessor.
type TProcessor interface {
	// ProcessTraces is a helper function that processes the incoming data and returns the data to be sent to the next component.
	// If error is returned then returned data are ignored. It MUST not call the next component.
	ProcessTraces(context.Context, pdata.Traces) (pdata.Traces, error)
}

// MProcessor is a helper interface that allows avoiding implementing all functions in MetricsProcessor by using NewTracesProcessor.
type MProcessor interface {
	// ProcessMetrics is a helper function that processes the incoming data and returns the data to be sent to the next component.
	// If error is returned then returned data are ignored. It MUST not call the next component.
	ProcessMetrics(context.Context, pdata.Metrics) (pdata.Metrics, error)
}

// LProcessor is a helper interface that allows avoiding implementing all functions in LogsProcessor by using NewLogsProcessor.
type LProcessor interface {
	// ProcessLogs is a helper function that processes the incoming data and returns the data to be sent to the next component.
	// If error is returned then returned data are ignored. It MUST not call the next component.
	ProcessLogs(context.Context, pdata.Logs) (pdata.Logs, error)
}

// Option apply changes to internalOptions.
type Option func(*baseSettings)

// WithStart overrides the default Start function for an processor.
// The default shutdown function does nothing and always returns nil.
func WithStart(start componenthelper.StartFunc) Option {
	return func(o *baseSettings) {
		o.componentOptions = append(o.componentOptions, componenthelper.WithStart(start))
	}
}

// WithShutdown overrides the default Shutdown function for an processor.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown componenthelper.ShutdownFunc) Option {
	return func(o *baseSettings) {
		o.componentOptions = append(o.componentOptions, componenthelper.WithShutdown(shutdown))
	}
}

// WithCapabilities overrides the default GetCapabilities function for an processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *baseSettings) {
		o.capabilities = capabilities
	}
}

type baseSettings struct {
	componentOptions []componenthelper.Option
	capabilities     consumer.Capabilities
}

// fromOptions returns the internal settings starting from the default and applying all options.
func fromOptions(options []Option) *baseSettings {
	// Start from the default options:
	opts := &baseSettings{
		capabilities: consumer.Capabilities{MutatesData: true},
	}

	for _, op := range options {
		op(opts)
	}

	return opts
}

// internalOptions contains internalOptions concerning how an Processor is configured.
type baseProcessor struct {
	component.Component
	capabilities    consumer.Capabilities
	traceAttributes []trace.Attribute
}

// Construct the internalOptions from multiple Option.
func newBaseProcessor(id config.ComponentID, options ...Option) baseProcessor {
	bs := fromOptions(options)
	be := baseProcessor{
		Component:    componenthelper.New(bs.componentOptions...),
		capabilities: bs.capabilities,
		traceAttributes: []trace.Attribute{
			trace.StringAttribute(obsreport.ProcessorKey, id.String()),
		},
	}

	return be
}

func (bp *baseProcessor) Capabilities() consumer.Capabilities {
	return bp.capabilities
}

type tracesProcessor struct {
	baseProcessor
	processor    TProcessor
	nextConsumer consumer.Traces
}

func (tp *tracesProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	span := trace.FromContext(ctx)
	span.Annotate(tp.traceAttributes, "Start processing.")
	var err error
	td, err = tp.processor.ProcessTraces(ctx, td)
	span.Annotate(tp.traceAttributes, "End processing.")
	if err != nil {
		if errors.Is(err, ErrSkipProcessingData) {
			return nil
		}
		return err
	}
	return tp.nextConsumer.ConsumeTraces(ctx, td)
}

// NewTracesProcessor creates a TracesProcessor that ensure context propagation and the right tags are set.
// TODO: Add observability metrics support
func NewTracesProcessor(
	cfg config.Processor,
	nextConsumer consumer.Traces,
	processor TProcessor,
	options ...Option,
) (component.TracesProcessor, error) {
	if processor == nil {
		return nil, errors.New("nil processor")
	}

	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	return &tracesProcessor{
		baseProcessor: newBaseProcessor(cfg.ID(), options...),
		processor:     processor,
		nextConsumer:  nextConsumer,
	}, nil
}

type metricsProcessor struct {
	baseProcessor
	processor    MProcessor
	nextConsumer consumer.Metrics
}

func (mp *metricsProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	span := trace.FromContext(ctx)
	span.Annotate(mp.traceAttributes, "Start processing.")
	var err error
	md, err = mp.processor.ProcessMetrics(ctx, md)
	span.Annotate(mp.traceAttributes, "End processing.")
	if err != nil {
		if errors.Is(err, ErrSkipProcessingData) {
			return nil
		}
		return err
	}
	return mp.nextConsumer.ConsumeMetrics(ctx, md)
}

// NewMetricsProcessor creates a MetricsProcessor that ensure context propagation and the right tags are set.
// TODO: Add observability metrics support
func NewMetricsProcessor(
	cfg config.Processor,
	nextConsumer consumer.Metrics,
	processor MProcessor,
	options ...Option,
) (component.MetricsProcessor, error) {
	if processor == nil {
		return nil, errors.New("nil processor")
	}

	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	return &metricsProcessor{
		baseProcessor: newBaseProcessor(cfg.ID(), options...),
		processor:     processor,
		nextConsumer:  nextConsumer,
	}, nil
}

type logProcessor struct {
	baseProcessor
	processor    LProcessor
	nextConsumer consumer.Logs
}

func (lp *logProcessor) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	span := trace.FromContext(ctx)
	span.Annotate(lp.traceAttributes, "Start processing.")
	var err error
	ld, err = lp.processor.ProcessLogs(ctx, ld)
	span.Annotate(lp.traceAttributes, "End processing.")
	if err != nil {
		if errors.Is(err, ErrSkipProcessingData) {
			return nil
		}
		return err
	}
	return lp.nextConsumer.ConsumeLogs(ctx, ld)
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

	return &logProcessor{
		baseProcessor: newBaseProcessor(cfg.ID(), options...),
		processor:     processor,
		nextConsumer:  nextConsumer,
	}, nil
}
