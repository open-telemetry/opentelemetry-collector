// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

// builtPipeline is a pipeline that is built based on a config.
// It can have a trace and/or a metrics consumer (the consumer is either the first
// processor in the pipeline or the exporter if pipeline has no processors).
type builtPipeline struct {
	firstTC consumer.TraceConsumerBase
	firstMC consumer.MetricsConsumerBase

	// MutatesConsumedData is set to true if any processors in the pipeline
	// can mutate the TraceData or MetricsData input argument.
	MutatesConsumedData bool

	processors []component.Processor
}

// BuiltPipelines is a map of build pipelines created from pipeline configs.
type BuiltPipelines map[*configmodels.Pipeline]*builtPipeline

func (bps BuiltPipelines) StartProcessors(logger *zap.Logger, host component.Host) error {
	for cfg, bp := range bps {
		logger.Info("Pipeline is starting...", zap.String("pipeline", cfg.Name))
		// Start in reverse order, starting from the back of processors pipeline.
		// This is important so that processors that are earlier in the pipeline and
		// reference processors that are later in the pipeline do not start sending
		// data to later pipelines which are not yet started.
		for i := len(bp.processors) - 1; i >= 0; i-- {
			if err := bp.processors[i].Start(host); err != nil {
				return err
			}
		}
		logger.Info("Pipeline is started.", zap.String("pipeline", cfg.Name))
	}
	return nil
}

func (bps BuiltPipelines) ShutdownProcessors(logger *zap.Logger) error {
	var errs []error
	for cfg, bp := range bps {
		logger.Info("Pipeline is shutting down...", zap.String("pipeline", cfg.Name))
		for _, p := range bp.processors {
			if err := p.Shutdown(); err != nil {
				errs = append(errs, err)
			}
		}
		logger.Info("Pipeline is shutdown.", zap.String("pipeline", cfg.Name))
	}

	if len(errs) != 0 {
		return oterr.CombineErrors(errs)
	}
	return nil
}

// PipelinesBuilder builds pipelines from config.
type PipelinesBuilder struct {
	logger    *zap.Logger
	config    *configmodels.Config
	exporters Exporters
	factories map[string]component.ProcessorFactoryBase
}

// NewPipelinesBuilder creates a new PipelinesBuilder. Requires exporters to be already
// built via ExportersBuilder. Call BuildProcessors() on the returned value.
func NewPipelinesBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	exporters Exporters,
	factories map[string]component.ProcessorFactoryBase,
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
func (pb *PipelinesBuilder) buildPipeline(
	pipelineCfg *configmodels.Pipeline,
) (*builtPipeline, error) {

	// BuildProcessors the pipeline backwards.

	// First create a consumer junction point that fans out the data to all exporters.
	var tc consumer.TraceConsumerBase
	var mc consumer.MetricsConsumerBase

	switch pipelineCfg.InputType {
	case configmodels.TracesDataType:
		tc = pb.buildFanoutExportersTraceConsumer(pipelineCfg.Exporters)
	case configmodels.MetricsDataType:
		mc = pb.buildFanoutExportersMetricsConsumer(pipelineCfg.Exporters)
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
		switch pipelineCfg.InputType {
		case configmodels.TracesDataType:
			var proc component.TraceProcessorBase
			proc, err = createTraceProcessor(factory, pb.logger, procCfg, tc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			tc = proc
		case configmodels.MetricsDataType:
			var proc component.MetricsProcessorBase
			proc, err = createMetricsProcessor(factory, pb.logger, procCfg, mc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			mc = proc
		}

		if err != nil {
			return nil, fmt.Errorf("error creating processor %q in pipeline %q: %v",
				procName, pipelineCfg.Name, err)
		}

		// Check if the factory really created the processor.
		if tc == nil && mc == nil {
			return nil, fmt.Errorf("factory for %q produced a nil processor", procCfg.Name())
		}
	}

	pb.logger.Info("Pipeline is enabled.", zap.String("pipelines", pipelineCfg.Name))

	bp := &builtPipeline{
		tc,
		mc,
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
		traceConverter := consumer.NewInternalToOCTraceConverter(nextConsumer.(consumer.TraceConsumerOld))
		return factory.CreateTraceProcessor(ctx, creationParams, traceConverter, cfg)
	}

	// If both processor and consumer are of the old type (can manipulate on OC traces only),
	// use ProcessorFactoryOld.CreateTraceProcessor.
	if nextConsumerOld, ok := nextConsumer.(consumer.TraceConsumerOld); ok {
		return factoryBase.(component.ProcessorFactoryOld).CreateTraceProcessor(logger, nextConsumerOld, cfg)
	}

	// Old type processor and a new type consumer usecase is not supported.
	// TODO: This case can be supported since we have OC->internal traces translation function
	return nil, errors.New("OC Traces -> internal data format translation is not supported")
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
		metricsConverter := consumer.NewInternalToOCMetricsConverter(nextConsumer.(consumer.MetricsConsumerOld))
		return factory.CreateMetricsProcessor(ctx, creationParams, metricsConverter, cfg)
	}

	// If both processor and consumer are of the old type (can manipulate on OC metrics only),
	// use ProcessorFactoryOld.CreateMetricsProcessor.
	if nextConsumerOld, ok := nextConsumer.(consumer.MetricsConsumerOld); ok {
		return factoryBase.(component.ProcessorFactoryOld).CreateMetricsProcessor(logger, nextConsumerOld, cfg)
	}

	// Old type processor and a new type consumer usecase is not supported.
	// TODO: This case can be supported once we have OC->internal metrics translation function
	return nil, errors.New("OC Metrics -> internal data format translation is not supported")
}
