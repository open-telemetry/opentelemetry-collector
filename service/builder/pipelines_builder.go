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
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

// builtPipeline is a pipeline that is built based on a config.
// It can have a trace and/or a metrics consumer (the consumer is either the first
// processor in the pipeline or the exporter if pipeline has no processors).
type builtPipeline struct {
	logger        *zap.Logger
	firstConsumer component.Consumer

	// MutatesConsumedData is set to true if any processors in the pipeline
	// can mutate the TraceData or MetricsData input argument.
	MutatesConsumedData bool

	processors []component.Processor

	factory component.PipelineFactory
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

	if len(errs) != 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

// PipelinesBuilder builds pipelines from config.
type PipelinesBuilder struct {
	logger    *zap.Logger
	config    *configmodels.Config
	exporters Exporters
	factories config.Factories
}

// NewPipelinesBuilder creates a new PipelinesBuilder. Requires exporters to be already
// built via ExportersBuilder. Call BuildProcessors() on the returned value.
func NewPipelinesBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	exporters Exporters,
	factories config.Factories,
) *PipelinesBuilder {
	return &PipelinesBuilder{logger, config, exporters, factories}
}

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

func (pb *PipelinesBuilder) buildPipeline(pipelineCfg *configmodels.Pipeline,
) (*builtPipeline, error) {
	pipelineFactory := pb.factories.Pipelines[pipelineCfg.Type()]

	exporters := pb.getExportersForPipeline(pipelineCfg)
	consumers := []component.Consumer{}
	for _, e := range exporters {
		consumers = append(consumers, e)
	}
	consumer := pipelineFactory.CreateFanOutConnector(consumers)

	mutatesConsumedData := false

	processors := make([]component.Processor, len(pipelineCfg.Processors))
	ctx := context.Background()

	// Now build the processors backwards, starting from the last one.
	// The last processor points to consumer which fans out to exporters, then
	// the processor itself becomes a consumer for the one that precedes it in
	// in the pipeline and so on.
	for i := len(pipelineCfg.Processors) - 1; i >= 0; i-- {
		procName := pipelineCfg.Processors[i]
		procCfg := pb.config.Processors[procName]

		factory := pb.factories.Processors[procCfg.Type()]

		// This processor must point to the next consumer and then
		// it becomes the next for the previous one (previous in the pipeline,
		// which we will build in the next loop iteration).
		var err error
		componentLogger := pb.logger.With(zap.String(kindLogKey, kindLogProcessor), zap.String(typeLogKey, string(procCfg.Type())), zap.String(nameLogKey, procCfg.Name()))

		proc, err := pipelineFactory.CreateProcessor(ctx, factory, componentLogger, consumer, procCfg)

		// proc, err = createProcessor(factory, componentLogger, procCfg, consumer)
		if proc != nil {
			mutatesConsumedData = mutatesConsumedData || proc.GetCapabilities().MutatesConsumedData
		}
		processors[i] = proc
		consumer = proc

		if err != nil {
			return nil, fmt.Errorf("error creating processor %q in pipeline %q: %v",
				procName, pipelineCfg.Name, err)
		}

		// Check if the factory really created the processor.
		if consumer == nil {
			return nil, fmt.Errorf("factory for %q produced a nil processor", procCfg.Name())
		}
	}

	pipelineLogger := pb.logger.With(zap.String("pipeline_name", pipelineCfg.Name),
		zap.String("pipeline_datatype", pipelineCfg.InputType.GetString()))
	pipelineLogger.Info("Pipeline is enabled.")

	bp := &builtPipeline{
		pipelineLogger,
		consumer,
		mutatesConsumedData,
		processors,
		pipelineFactory,
	}

	return bp, nil
}

func (pb *PipelinesBuilder) getExportersForPipeline(cfg *configmodels.Pipeline) []component.Exporter {
	exporters := []component.Exporter{}
	for _, name := range cfg.Exporters {
		builtExporter := pb.exporters[pb.config.Exporters[name]]
		if pexp, ok := builtExporter.byPipeline[cfg.Name]; ok {
			for _, e := range pexp {
				exporters = append(exporters, e)
			}
		}
	}
	return exporters
}
