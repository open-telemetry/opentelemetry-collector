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

package builder

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/converter"
	"go.opentelemetry.io/collector/processor"
)

// builtPipeline is a pipeline that is built based on a config.
// It can have a trace and/or a metrics consumer (the consumer is either the first
// processor in the pipeline or the exporter if pipeline has no processors).
type builtPipeline struct {
	logger  *zap.Logger
	firstTC consumer.TraceConsumerBase
	firstMC consumer.MetricsConsumerBase
	firstLC consumer.LogsConsumer

	// MutatesConsumedData is set to true if any processors in the pipeline
	// can mutate the TraceData or MetricsData input argument.
	MutatesConsumedData bool

	processors []component.Processor
}

// BuiltPipelines is a map of build pipelines created from pipeline configs.
type BuiltPipelines map[*configmodels.Pipeline]*builtPipeline

func (bps BuiltPipelines) StartProcessors(ctx context.Context, host component.Host) error {
	for _, bp := range bps {
		bp.logger.Info("Pipeline is starting...")
		// Start in reverse order, starting from the back of processors pipeline.
		// This is important so that processors that are earlier in the pipeline and
		// reference processors that are later in the pipeline do not start sending
		// data to later pipelines which are not yet started.
		for i := len(bp.processors) - 1; i >= 0; i-- {
			if err := bp.processors[i].Start(ctx, host); err != nil {
				return err
			}
		}
		bp.logger.Info("Pipeline is started.")
	}
	return nil
}

func (bps BuiltPipelines) ShutdownProcessors(ctx context.Context) error {
	var errs []error
	for _, bp := range bps {
		bp.logger.Info("Pipeline is shutting down...")
		for _, p := range bp.processors {
			if err := p.Shutdown(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		bp.logger.Info("Pipeline is shutdown.")
	}

	return componenterror.CombineErrors(errs)
}

// PipelinesBuilder builds pipelines from config.
type PipelinesBuilder struct {
	logger    *zap.Logger
	config    *configmodels.Config
	exporters Exporters
	factories map[configmodels.Type]component.ProcessorFactoryBase
}

// NewPipelinesBuilder creates a new PipelinesBuilder. Requires exporters to be already
// built via ExportersBuilder. Call BuildProcessors() on the returned value.
func NewPipelinesBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	exporters Exporters,
	factories map[configmodels.Type]component.ProcessorFactoryBase,
) *PipelinesBuilder {
	return &PipelinesBuilder{logger, config, exporters, factories}
}

// BuildProcessors pipeline processors from config.
func (pb *PipelinesBuilder) Build() (BuiltPipelines, error) {
	pipelineProcessors := make(BuiltPipelines)

	for _, pipeline := range pb.config.Service.Pipelines {
		firstProcessor, err := pb.buildPipeline(pipeline)
		if err != nil {
			return nil, err
		}
		pipelineProcessors[pipeline] = firstProcessor
	}

	return pipelineProcessors, nil
}

// Builds a pipeline of processors. Returns the first processor in the pipeline.
// The last processor in the pipeline will be plugged to fan out the data into exporters
// that are configured for this pipeline.
func (pb *PipelinesBuilder) buildPipeline(pipelineCfg *configmodels.Pipeline,
) (*builtPipeline, error) {

	// BuildProcessors the pipeline backwards.

	// First create a consumer junction point that fans out the data to all exporters.
	var tc consumer.TraceConsumerBase
	var mc consumer.MetricsConsumerBase
	var lc consumer.LogsConsumer

	switch pipelineCfg.InputType {
	case configmodels.TracesDataType:
		tc = pb.buildFanoutExportersTraceConsumer(pipelineCfg.Exporters)
	case configmodels.MetricsDataType:
		mc = pb.buildFanoutExportersMetricsConsumer(pipelineCfg.Exporters)
	case configmodels.LogsDataType:
		lc = pb.buildFanoutExportersLogConsumer(pipelineCfg.Exporters)
	}

	mutatesConsumedData := false

	processors := make([]component.Processor, len(pipelineCfg.Processors))

	// Now build the processors backwards, starting from the last one.
	// The last processor points to consumer which fans out to exporters, then
	// the processor itself becomes a consumer for the one that precedes it in
	// in the pipeline and so on.
	for i := len(pipelineCfg.Processors) - 1; i >= 0; i-- {
		procName := pipelineCfg.Processors[i]
		procCfg := pb.config.Processors[procName]

		factory := pb.factories[procCfg.Type()]

		// This processor must point to the next consumer and then
		// it becomes the next for the previous one (previous in the pipeline,
		// which we will build in the next loop iteration).
		var err error
		componentLogger := pb.logger.With(zap.String(kindLogKey, kindLogsProcessor), zap.String(typeLogKey, string(procCfg.Type())), zap.String(nameLogKey, procCfg.Name()))
		switch pipelineCfg.InputType {
		case configmodels.TracesDataType:
			var proc component.TraceProcessorBase
			proc, err = createTraceProcessor(factory, componentLogger, procCfg, tc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			tc = proc
		case configmodels.MetricsDataType:
			var proc component.MetricsProcessorBase
			proc, err = createMetricsProcessor(factory, componentLogger, procCfg, mc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			mc = proc

		case configmodels.LogsDataType:
			var proc component.LogsProcessor
			proc, err = createLogsProcessor(factory, componentLogger, procCfg, lc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			lc = proc

		default:
			return nil, fmt.Errorf("error creating processor %q in pipeline %q, data type %s is not supported",
				procName, pipelineCfg.Name, pipelineCfg.InputType)
		}

		if err != nil {
			return nil, fmt.Errorf("error creating processor %q in pipeline %q: %v",
				procName, pipelineCfg.Name, err)
		}

		// Check if the factory really created the processor.
		if tc == nil && mc == nil && lc == nil {
			return nil, fmt.Errorf("factory for %q produced a nil processor", procCfg.Name())
		}
	}

	pipelineLogger := pb.logger.With(zap.String("pipeline_name", pipelineCfg.Name),
		zap.String("pipeline_datatype", string(pipelineCfg.InputType)))
	pipelineLogger.Info("Pipeline is enabled.")

	bp := &builtPipeline{
		pipelineLogger,
		tc,
		mc,
		lc,
		mutatesConsumedData,
		processors,
	}

	return bp, nil
}

// Converts the list of exporter names to a list of corresponding builtExporters.
func (pb *PipelinesBuilder) getBuiltExportersByNames(exporterNames []string) []*builtExporter {
	var result []*builtExporter
	for _, name := range exporterNames {
		exporter := pb.exporters[pb.config.Exporters[name]]
		result = append(result, exporter)
	}

	return result
}

func (pb *PipelinesBuilder) buildFanoutExportersTraceConsumer(exporterNames []string) consumer.TraceConsumerBase {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	// Optimize for the case when there is only one exporter, no need to create junction point.
	if len(builtExporters) == 1 {
		return builtExporters[0].te
	}

	var exporters []consumer.TraceConsumerBase
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.te)
	}

	// Create a junction point that fans out to all exporters.
	return processor.CreateTraceFanOutConnector(exporters)
}

func (pb *PipelinesBuilder) buildFanoutExportersMetricsConsumer(exporterNames []string) consumer.MetricsConsumerBase {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	// Optimize for the case when there is only one exporter, no need to create junction point.
	if len(builtExporters) == 1 {
		return builtExporters[0].me
	}

	var exporters []consumer.MetricsConsumerBase
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.me)
	}

	// Create a junction point that fans out to all exporters.
	return processor.CreateMetricsFanOutConnector(exporters)
}

func (pb *PipelinesBuilder) buildFanoutExportersLogConsumer(
	exporterNames []string,
) consumer.LogsConsumer {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	// Optimize for the case when there is only one exporter, no need to create junction point.
	if len(builtExporters) == 1 {
		return builtExporters[0].le
	}

	exporters := make([]consumer.LogsConsumer, len(builtExporters))
	for i, builtExp := range builtExporters {
		exporters[i] = builtExp.le
	}

	// Create a junction point that fans out to all exporters.
	return processor.NewLogFanOutConnector(exporters)
}

// createTraceProcessor creates trace processor based on type of the current processor
// and type of the downstream consumer.
func createTraceProcessor(
	factoryBase component.ProcessorFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Processor,
	nextConsumer consumer.TraceConsumerBase,
) (component.TraceProcessorBase, error) {
	if factory, ok := factoryBase.(component.ProcessorFactory); ok {
		creationParams := component.ProcessorCreateParams{Logger: logger}
		ctx := context.Background()

		// If both processor and consumer are of the new type (can manipulate on internal data structure),
		// use ProcessorFactory.CreateTraceProcessor.
		if nextConsumer, ok := nextConsumer.(consumer.TraceConsumer); ok {
			return factory.CreateTraceProcessor(ctx, creationParams, nextConsumer, cfg)
		}

		// If processor is of the new type, but downstream consumer is of the old type,
		// use internalToOCTraceConverter compatibility shim.
		traceConverter := converter.NewInternalToOCTraceConverter(nextConsumer.(consumer.TraceConsumerOld))
		return factory.CreateTraceProcessor(ctx, creationParams, traceConverter, cfg)
	}

	factoryOld := factoryBase.(component.ProcessorFactoryOld)

	// If both processor and consumer are of the old type (can manipulate on OC traces only),
	// use ProcessorFactoryOld.CreateTraceProcessor.
	if nextConsumerOld, ok := nextConsumer.(consumer.TraceConsumerOld); ok {
		return factoryOld.CreateTraceProcessor(logger, nextConsumerOld, cfg)
	}

	// If processor is of the old type, but downstream consumer is of the new type,
	// use NewInternalToOCTraceConverter compatibility shim to convert traces from internal format to OC.
	traceConverter := converter.NewOCToInternalTraceConverter(nextConsumer.(consumer.TraceConsumer))
	return factoryOld.CreateTraceProcessor(logger, traceConverter, cfg)
}

// createMetricsProcessor creates metric processor based on type of the current processor
// and type of the downstream consumer.
func createMetricsProcessor(
	factoryBase component.ProcessorFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Processor,
	nextConsumer consumer.MetricsConsumerBase,
) (component.MetricsProcessorBase, error) {
	if factory, ok := factoryBase.(component.ProcessorFactory); ok {
		creationParams := component.ProcessorCreateParams{Logger: logger}
		ctx := context.Background()

		// If both processor and consumer are of the new type (can manipulate on internal data structure),
		// use ProcessorFactory.CreateMetricsProcessor.
		if nextConsumer, ok := nextConsumer.(consumer.MetricsConsumer); ok {
			return factory.CreateMetricsProcessor(ctx, creationParams, nextConsumer, cfg)
		}

		// If processor is of the new type, but downstream consumer is of the old type,
		// use internalToOCMetricsConverter compatibility shim.
		metricsConverter := converter.NewInternalToOCMetricsConverter(nextConsumer.(consumer.MetricsConsumerOld))
		return factory.CreateMetricsProcessor(ctx, creationParams, metricsConverter, cfg)
	}

	factoryOld := factoryBase.(component.ProcessorFactoryOld)

	// If both processor and consumer are of the old type (can manipulate on OC metrics only),
	// use ProcessorFactoryOld.CreateMetricsProcessor.
	if nextConsumerOld, ok := nextConsumer.(consumer.MetricsConsumerOld); ok {
		return factoryOld.CreateMetricsProcessor(logger, nextConsumerOld, cfg)
	}

	// If processor is of the old type, but downstream consumer is of the new type,
	// use NewInternalToOCMetricsConverter compatibility shim to convert metrics from internal format to OC.
	metricsConverter := converter.NewOCToInternalMetricsConverter(nextConsumer.(consumer.MetricsConsumer))
	return factoryOld.CreateMetricsProcessor(logger, metricsConverter, cfg)
}

// createLogsProcessor creates a log processor using given factory and next consumer.
func createLogsProcessor(
	factoryBase component.ProcessorFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Processor,
	nextConsumer consumer.LogsConsumer,
) (component.LogsProcessor, error) {
	factory, ok := factoryBase.(component.LogsProcessorFactory)
	if !ok {
		return nil, fmt.Errorf("processor %q does support data type %q",
			cfg.Name(), configmodels.LogsDataType)
	}
	creationParams := component.ProcessorCreateParams{Logger: logger}
	ctx := context.Background()
	return factory.CreateLogsProcessor(ctx, creationParams, cfg, nextConsumer)
}
