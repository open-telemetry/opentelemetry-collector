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
	"go.opentelemetry.io/collector/processor"
)

// builtPipeline is a pipeline that is built based on a config.
// It can have a trace and/or a metrics consumer (the consumer is either the first
// processor in the pipeline or the exporter if pipeline has no processors).
type builtPipeline struct {
	logger  *zap.Logger
	firstTC consumer.TracesConsumer
	firstMC consumer.MetricsConsumer
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
	appInfo   component.ApplicationStartInfo
	config    *configmodels.Config
	exporters Exporters
	factories map[configmodels.Type]component.ProcessorFactory
}

// NewPipelinesBuilder creates a new PipelinesBuilder. Requires exporters to be already
// built via ExportersBuilder. Call BuildProcessors() on the returned value.
func NewPipelinesBuilder(
	logger *zap.Logger,
	appInfo component.ApplicationStartInfo,
	config *configmodels.Config,
	exporters Exporters,
	factories map[configmodels.Type]component.ProcessorFactory,
) *PipelinesBuilder {
	return &PipelinesBuilder{logger, appInfo, config, exporters, factories}
}

// BuildProcessors pipeline processors from config.
func (pb *PipelinesBuilder) Build() (BuiltPipelines, error) {
	pipelineProcessors := make(BuiltPipelines)

	for _, pipeline := range pb.config.Service.Pipelines {
		firstProcessor, err := pb.buildPipeline(context.Background(), pipeline)
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
func (pb *PipelinesBuilder) buildPipeline(ctx context.Context, pipelineCfg *configmodels.Pipeline) (*builtPipeline, error) {

	// BuildProcessors the pipeline backwards.

	// First create a consumer junction point that fans out the data to all exporters.
	var tc consumer.TracesConsumer
	var mc consumer.MetricsConsumer
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
		creationParams := component.ProcessorCreateParams{
			Logger:               componentLogger,
			ApplicationStartInfo: pb.appInfo,
		}

		switch pipelineCfg.InputType {
		case configmodels.TracesDataType:
			var proc component.TracesProcessor
			proc, err = factory.CreateTracesProcessor(ctx, creationParams, procCfg, tc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			tc = proc
		case configmodels.MetricsDataType:
			var proc component.MetricsProcessor
			proc, err = factory.CreateMetricsProcessor(ctx, creationParams, procCfg, mc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
			}
			processors[i] = proc
			mc = proc

		case configmodels.LogsDataType:
			var proc component.LogsProcessor
			proc, err = factory.CreateLogsProcessor(ctx, creationParams, procCfg, lc)
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

func (pb *PipelinesBuilder) buildFanoutExportersTraceConsumer(exporterNames []string) consumer.TracesConsumer {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	var exporters []consumer.TracesConsumer
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.getTraceExporter())
	}

	// Create a junction point that fans out to all exporters.
	return processor.NewTracesFanOutConnector(exporters)
}

func (pb *PipelinesBuilder) buildFanoutExportersMetricsConsumer(exporterNames []string) consumer.MetricsConsumer {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	var exporters []consumer.MetricsConsumer
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.getMetricExporter())
	}

	// Create a junction point that fans out to all exporters.
	return processor.NewMetricsFanOutConnector(exporters)
}

func (pb *PipelinesBuilder) buildFanoutExportersLogConsumer(exporterNames []string) consumer.LogsConsumer {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	exporters := make([]consumer.LogsConsumer, len(builtExporters))
	for i, builtExp := range builtExporters {
		exporters[i] = builtExp.getLogExporter()
	}

	// Create a junction point that fans out to all exporters.
	return processor.NewLogsFanOutConnector(exporters)
}
