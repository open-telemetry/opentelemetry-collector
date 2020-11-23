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
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

// ErrSkipProcessingData is a sentinel value to indicate when traces or metrics should intentionally be dropped
// from further processing in the pipeline because the data is determined to be irrelevant. A processor can return this error
// to stop further processing without propagating an error back up the pipeline to logs.
var ErrSkipProcessingData = errors.New("sentinel error to skip processing data from the remainder of the pipeline")

// TProcessor is a helper interface that allows avoiding implementing all functions in TracesProcessor by using NewTraceProcessor.
type TProcessor interface {
	// ProcessTraces is a helper function that processes the incoming data and returns the data to be sent to the next component.
	// If error is returned then returned data are ignored. It MUST not call the next component.
	ProcessTraces(context.Context, pdata.Traces) (pdata.Traces, error)
}

// MProcessor is a helper interface that allows avoiding implementing all functions in MetricsProcessor by using NewTraceProcessor.
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
func WithStart(start componenthelper.Start) Option {
	return func(o *baseSettings) {
		o.Start = start
	}
}

// WithShutdown overrides the default Shutdown function for an processor.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown componenthelper.Shutdown) Option {
	return func(o *baseSettings) {
		o.Shutdown = shutdown
	}
}

// WithShutdown overrides the default GetCapabilities function for an processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities component.ProcessorCapabilities) Option {
	return func(o *baseSettings) {
		o.capabilities = capabilities
	}
}

type baseSettings struct {
	*componenthelper.ComponentSettings
	capabilities component.ProcessorCapabilities
}

// fromOptions returns the internal settings starting from the default and applying all options.
func fromOptions(options []Option) *baseSettings {
	// Start from the default options:
	opts := &baseSettings{
		ComponentSettings: componenthelper.DefaultComponentSettings(),
		capabilities:      component.ProcessorCapabilities{MutatesConsumedData: true},
	}

	for _, op := range options {
		op(opts)
	}

	return opts
}

// internalOptions contains internalOptions concerning how an Processor is configured.
type baseProcessor struct {
	component.Component
	fullName        string
	capabilities    component.ProcessorCapabilities
	traceAttributes []trace.Attribute
}

// Construct the internalOptions from multiple Option.
func newBaseProcessor(fullName string, options ...Option) baseProcessor {
	bs := fromOptions(options)
	be := baseProcessor{
		Component:    componenthelper.NewComponent(bs.ComponentSettings),
		fullName:     fullName,
		capabilities: bs.capabilities,
		traceAttributes: []trace.Attribute{
			trace.StringAttribute(obsreport.ProcessorKey, fullName),
		},
	}

	return be
}

func (bp *baseProcessor) GetCapabilities() component.ProcessorCapabilities {
	return bp.capabilities
}

type tracesProcessor struct {
	baseProcessor
	processor    TProcessor
	nextConsumer consumer.TracesConsumer
}

func (tp *tracesProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	span := trace.FromContext(ctx)
	span.Annotate(tp.traceAttributes, "Start processing.")
	var err error
	td, err = tp.processor.ProcessTraces(ctx, td)
	span.Annotate(tp.traceAttributes, "End processing.")
	if err != nil {
		return err
	}
	return tp.nextConsumer.ConsumeTraces(ctx, td)
}

// NewTraceProcessor creates a TracesProcessor that ensure context propagation and the right tags are set.
// TODO: Add observability metrics support
func NewTraceProcessor(
	config configmodels.Processor,
	nextConsumer consumer.TracesConsumer,
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
		baseProcessor: newBaseProcessor(config.Name(), options...),
		processor:     processor,
		nextConsumer:  nextConsumer,
	}, nil
}

type metricsProcessor struct {
	baseProcessor
	processor    MProcessor
	nextConsumer consumer.MetricsConsumer
}

func (mp *metricsProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	span := trace.FromContext(ctx)
	span.Annotate(mp.traceAttributes, "Start processing.")
	var err error
	md, err = mp.processor.ProcessMetrics(ctx, md)
	span.Annotate(mp.traceAttributes, "End processing.")
	if err != nil {
		if err == ErrSkipProcessingData {
			return nil
		}
		return err
	}
	return mp.nextConsumer.ConsumeMetrics(ctx, md)
}

// NewMetricsProcessor creates a MetricsProcessor that ensure context propagation and the right tags are set.
// TODO: Add observability metrics support
func NewMetricsProcessor(
	config configmodels.Processor,
	nextConsumer consumer.MetricsConsumer,
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
		baseProcessor: newBaseProcessor(config.Name(), options...),
		processor:     processor,
		nextConsumer:  nextConsumer,
	}, nil
}

type logProcessor struct {
	baseProcessor
	processor    LProcessor
	nextConsumer consumer.LogsConsumer
}

func (lp *logProcessor) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	span := trace.FromContext(ctx)
	span.Annotate(lp.traceAttributes, "Start processing.")
	var err error
	ld, err = lp.processor.ProcessLogs(ctx, ld)
	span.Annotate(lp.traceAttributes, "End processing.")
	if err != nil {
		return err
	}
	return lp.nextConsumer.ConsumeLogs(ctx, ld)
}

// NewLogsProcessor creates a LogsProcessor that ensure context propagation and the right tags are set.
// TODO: Add observability metrics support
func NewLogsProcessor(
	config configmodels.Processor,
	nextConsumer consumer.LogsConsumer,
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
		baseProcessor: newBaseProcessor(config.Name(), options...),
		processor:     processor,
		nextConsumer:  nextConsumer,
	}, nil
}
