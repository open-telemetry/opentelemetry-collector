package tracespipeline

import (
	"context"
	"fmt"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/converter"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// TraceReceiver ...
type TraceReceiver = component.TraceReceiver

// TraceExporter ...
type TraceExporter = component.TraceExporter

// TraceExporterOld ...
type TraceExporterOld = component.TraceExporterOld

// TraceProcessor ...
type TraceProcessor = component.TraceProcessor

// TraceProcessorOld ...
type TraceProcessorOld = component.TraceProcessorOld

// TraceConsumer ...
type TraceConsumer = consumer.TraceConsumer

// TraceConsumerOld ...
type TraceConsumerOld = consumer.TraceConsumerOld

// ProcessorFactory ...
type ProcessorFactory interface {
	component.ProcessorFactoryBase

	CreateTraceProcessor(context.Context, component.ProcessorCreateParams,
		TraceConsumer, configmodels.Processor) (TraceProcessor, error)
}

// ProcessorFactoryOld ...
type ProcessorFactoryOld interface {
	component.ProcessorFactoryBase

	CreateTraceProcessor(*zap.Logger, TraceConsumerOld, configmodels.Processor) (TraceProcessorOld, error)
}

// TraceExporterFactory ...
type TraceExporterFactory interface {
	component.ExporterFactoryBase

	CreateTraceExporter(context.Context, component.ExporterCreateParams,
		configmodels.Exporter) (TraceExporter, error)
}

// TraceExporterFactoryOld ...
type TraceExporterFactoryOld interface {
	component.ExporterFactoryBase

	CreateTraceExporter(logger *zap.Logger, cfg configmodels.Exporter) (TraceExporterOld, error)
}

type ReceiverFactory interface {
	component.ReceiverFactoryBase

	CreateTraceReceiver(ctx context.Context, params component.ReceiverCreateParams,
		cfg configmodels.Receiver, nextConsumer consumer.TraceConsumer) (TraceReceiver, error)
}

type ReceiverFactoryOld interface {
	component.ReceiverFactoryBase

	CreateTraceReceiver(ctx context.Context, logger *zap.Logger, cfg configmodels.Receiver,
		nextConsumer consumer.TraceConsumerOld) (TraceReceiver, error)
}

// Factory ...
type Factory struct{}

// Type ...
func (f *Factory) Type() configmodels.Type {
	return configmodels.Type("traces")
}

// CreateReceiver ...
func (f *Factory) CreateReceiver(
	ctx context.Context,
	factoryBase component.ReceiverFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer component.Consumer,
) (component.Receiver, error) {

	if factory, ok := factoryBase.(ReceiverFactory); ok {
		creationParams := component.ReceiverCreateParams{Logger: logger}

		// If both receiver and consumer are of the new type (can manipulate on internal data structure),
		// use ProcessorFactory.CreateTraceReceiver.
		if nextConsumer, ok := nextConsumer.(TraceConsumer); ok {
			return factory.CreateTraceReceiver(ctx, creationParams, cfg, nextConsumer)
		}

		// If receiver is of the new type, but downstream consumer is of the old type,
		// use internalToOCTraceConverter compatibility shim.
		traceConverter := converter.NewInternalToOCTraceConverter(nextConsumer.(TraceConsumerOld))
		return factory.CreateTraceReceiver(ctx, creationParams, cfg, traceConverter)
	}

	factoryOld := factoryBase.(ReceiverFactoryOld)

	// If both receiver and consumer are of the old type (can manipulate on OC traces only),
	// use Factory.CreateTraceReceiver.
	if nextConsumer, ok := nextConsumer.(TraceConsumerOld); ok {
		return factoryOld.CreateTraceReceiver(ctx, logger, cfg, nextConsumer)
	}

	// If receiver is of the old type, but downstream consumer is of the new type,
	// use NewInternalToOCTraceConverter compatibility shim to convert traces from internal format to OC.
	traceConverter := converter.NewOCToInternalTraceConverter(nextConsumer.(TraceConsumer))
	return factoryOld.CreateTraceReceiver(ctx, logger, cfg, traceConverter)

}

// CreateExporter ...
func (f *Factory) CreateExporter(
	ctx context.Context,
	factoryBase component.ExporterFactoryBase,
	logger *zap.Logger,
	config configmodels.Exporter,
) (component.Exporter, error) {
	if factory, ok := factoryBase.(TraceExporterFactory); ok {
		return factory.CreateTraceExporter(ctx, component.ExporterCreateParams{logger}, config)
	}

	if factory, ok := factoryBase.(TraceExporterFactoryOld); ok {
		return factory.CreateTraceExporter(logger, config)
	}
	return nil, fmt.Errorf("is not a valid exporter factory")
}

// CreateProcessor ...
func (f *Factory) CreateProcessor(ctx context.Context, procFactory component.ProcessorFactoryBase, logger *zap.Logger,
	nextConsumer component.Consumer, cfg configmodels.Processor) (component.Processor, error) {

	if factory, ok := procFactory.(ProcessorFactory); ok {
		creationParams := component.ProcessorCreateParams{Logger: logger}

		// If both processor and consumer are of the new type (can manipulate on internal data structure),
		// use ProcessorFactory.CreateTraceProcessor.
		if nextConsumer, ok := nextConsumer.(TraceConsumer); ok {
			return factory.CreateTraceProcessor(ctx, creationParams, nextConsumer, cfg)
		}

		// If processor is of the new type, but downstream consumer is of the old type,
		// use internalToOCTraceConverter compatibility shim.
		traceConverter := converter.NewInternalToOCTraceConverter(nextConsumer.(TraceConsumerOld))
		return factory.CreateTraceProcessor(ctx, creationParams, traceConverter, cfg)
	}

	if factoryOld, ok := procFactory.(ProcessorFactoryOld); ok {
		// If both processor and consumer are of the old type (can manipulate on OC traces only),
		// use ProcessorFactoryOld.CreateTraceProcessor.
		if nextConsumerOld, ok := nextConsumer.(TraceConsumerOld); ok {
			return factoryOld.CreateTraceProcessor(logger, nextConsumerOld, cfg)
		}

		// If processor is of the old type, but downstream consumer is of the new type,
		// use NewInternalToOCTraceConverter compatibility shim to convert traces from internal format to OC.
		traceConverter := converter.NewOCToInternalTraceConverter(nextConsumer.(TraceConsumer))
		return factoryOld.CreateTraceProcessor(logger, traceConverter, cfg)

	}

	return nil, fmt.Errorf("is not a valid processor factory")
}

// fanout connection

// CreateFanoutConnection ...

// CreateFanOutConnector wraps multiple trace consumers in a single one.
// If any of the wrapped trace consumers are of the new type, use traceFanOutConnector,
// otherwise use the old type connector
func (f *Factory) CreateFanOutConnector(tcs []component.Consumer) component.Consumer {
	traceConsumersOld := make([]TraceConsumerOld, 0, len(tcs))
	traceConsumers := make([]TraceConsumer, 0, len(tcs))
	allTraceConsumersOld := true
	for _, tc := range tcs {
		if traceConsumer, ok := tc.(TraceConsumer); ok {
			allTraceConsumersOld = false
			traceConsumers = append(traceConsumers, traceConsumer)
		} else {
			traceConsumerOld := tc.(TraceConsumerOld)
			traceConsumersOld = append(traceConsumersOld, traceConsumerOld)
			traceConsumers = append(traceConsumers, converter.NewInternalToOCTraceConverter(traceConsumerOld))
		}
	}

	if allTraceConsumersOld {
		return newTraceFanOutConnectorOld(traceConsumersOld)
	}
	return newTraceFanOutConnector(traceConsumers)
}

func newTraceFanOutConnectorOld(tcs []consumer.TraceConsumerOld) consumer.TraceConsumerOld {
	return fanOutConnectorOld(tcs)
}

func newTraceFanOutConnector(tcs []consumer.TraceConsumer) consumer.TraceConsumer {
	return fanOutConnector(tcs)
}

type fanOutConnectorOld []consumer.TraceConsumerOld

var _ consumer.TraceConsumerOld = (*fanOutConnectorOld)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc fanOutConnectorOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	var errs []error
	for _, tc := range tfc {
		if err := tc.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

type fanOutConnector []consumer.TraceConsumer

var _ consumer.TraceConsumer = (*fanOutConnector)(nil)

func (fc fanOutConnector) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	var errs []error
	for _, tc := range fc {
		if err := tc.ConsumeTraces(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

// cloning connection
// TODO: add cloning connection

func (f *Factory) CreateCloningFanOutConnector(tcs []component.Consumer) component.Consumer {
	if len(tcs) == 1 {
		// Don't wrap if no need to do it.
		return tcs[0]
	}
	traceConsumersOld := make([]consumer.TraceConsumerOld, 0, len(tcs))
	traceConsumers := make([]consumer.TraceConsumer, 0, len(tcs))
	allTraceConsumersOld := true
	for _, tc := range tcs {
		if traceConsumer, ok := tc.(consumer.TraceConsumer); ok {
			allTraceConsumersOld = false
			traceConsumers = append(traceConsumers, traceConsumer)
		} else {
			traceConsumerOld := tc.(consumer.TraceConsumerOld)
			traceConsumersOld = append(traceConsumersOld, traceConsumerOld)
			traceConsumers = append(traceConsumers, converter.NewInternalToOCTraceConverter(traceConsumerOld))
		}
	}

	if allTraceConsumersOld {
		return newTraceCloningFanOutConnectorOld(traceConsumersOld)
	}
	return newTraceCloningFanOutConnector(traceConsumers)
}

func newTraceCloningFanOutConnectorOld(tcs []consumer.TraceConsumerOld) consumer.TraceConsumerOld {
	return cloningFanOutConnectorOld(tcs)
}

type cloningFanOutConnectorOld []consumer.TraceConsumerOld

var _ consumer.TraceConsumerOld = (*cloningFanOutConnectorOld)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc cloningFanOutConnectorOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		if err := tfc[i].ConsumeTraceData(ctx, cloneTraceDataOld(td)); err != nil {
			errs = append(errs, err)
		}
	}

	if len(tfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := tfc[len(tfc)-1]
		if err := lastTc.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

func newTraceCloningFanOutConnector(tcs []consumer.TraceConsumer) consumer.TraceConsumer {
	return cloningFanOutConnector(tcs)
}

type cloningFanOutConnector []consumer.TraceConsumer

var _ consumer.TraceConsumer = (*cloningFanOutConnector)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc cloningFanOutConnector) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		clone := td.Clone()
		if err := tfc[i].ConsumeTraces(ctx, clone); err != nil {
			errs = append(errs, err)
		}
	}

	if len(tfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := tfc[len(tfc)-1]
		if err := lastTc.ConsumeTraces(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

func cloneTraceDataOld(td consumerdata.TraceData) consumerdata.TraceData {
	clone := consumerdata.TraceData{
		SourceFormat: td.SourceFormat,
		Node:         proto.Clone(td.Node).(*commonpb.Node),
		Resource:     proto.Clone(td.Resource).(*resourcepb.Resource),
	}

	if td.Spans != nil {
		clone.Spans = make([]*tracepb.Span, 0, len(td.Spans))

		for _, span := range td.Spans {
			spanClone := proto.Clone(span).(*tracepb.Span)
			clone.Spans = append(clone.Spans, spanClone)
		}
	}

	return clone
}
