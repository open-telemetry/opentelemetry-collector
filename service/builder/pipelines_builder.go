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
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/processor"
	"github.com/open-telemetry/opentelemetry-service/processor/fanoutprocessor"
)

// builtProcessor is a processor that is built based on a config.
// It can have a trace and/or a metrics consumer.
type builtProcessor struct {
	tc consumer.TraceConsumer
	mc consumer.MetricsConsumer
}

// PipelineProcessors is a map of entry-point processors created from pipeline configs.
// Each element of the map points to the first processor of the pipeline.
type PipelineProcessors map[*configmodels.Pipeline]*builtProcessor

// PipelinesBuilder builds pipelines from config.
type PipelinesBuilder struct {
	logger    *zap.Logger
	config    *configmodels.Config
	exporters Exporters
	factories map[string]processor.Factory
}

// NewPipelinesBuilder creates a new PipelinesBuilder. Requires exporters to be already
// built via ExportersBuilder. Call Build() on the returned value.
func NewPipelinesBuilder(
	logger *zap.Logger,
	config *configmodels.Config,
	exporters Exporters,
	factories map[string]processor.Factory,
) *PipelinesBuilder {
	return &PipelinesBuilder{logger, config, exporters, factories}
}

// Build pipeline processors from config.
func (pb *PipelinesBuilder) Build() (PipelineProcessors, error) {
	pipelineProcessors := make(PipelineProcessors)

	for _, pipeline := range pb.config.Pipelines {
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
) (*builtProcessor, error) {

	// Build the pipeline backwards.

	// First create a consumer junction point that fans out the data to all exporters.
	var tc consumer.TraceConsumer
	var mc consumer.MetricsConsumer

	switch pipelineCfg.InputType {
	case configmodels.TracesDataType:
		tc = pb.buildFanoutExportersTraceConsumer(pipelineCfg.Exporters)
	case configmodels.MetricsDataType:
		mc = pb.buildFanoutExportersMetricsConsumer(pipelineCfg.Exporters)
	}

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
			tc, err = factory.CreateTraceProcessor(pb.logger, tc, procCfg)
		case configmodels.MetricsDataType:
			mc, err = factory.CreateMetricsProcessor(pb.logger, mc, procCfg)
		}

		if err != nil {
			return nil, fmt.Errorf("error creating processor %q in pipeline %q: %v",
				procName, pipelineCfg.Name, err)
		}
	}

	pb.logger.Info("Pipeline is enabled.", zap.String("pipelines", pipelineCfg.Name))

	return &builtProcessor{tc, mc}, nil
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

func (pb *PipelinesBuilder) buildFanoutExportersTraceConsumer(exporterNames []string) consumer.TraceConsumer {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	// Optimize for the case when there is only one exporter, no need to create junction point.
	if len(builtExporters) == 1 {
		return builtExporters[0].tc
	}

	var exporters []consumer.TraceConsumer
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.tc)
	}

	// Create a junction point that fans out to all exporters.
	return fanoutprocessor.NewTraceProcessor(exporters)
}

func (pb *PipelinesBuilder) buildFanoutExportersMetricsConsumer(exporterNames []string) consumer.MetricsConsumer {
	builtExporters := pb.getBuiltExportersByNames(exporterNames)

	// Optimize for the case when there is only one exporter, no need to create junction point.
	if len(builtExporters) == 1 {
		return builtExporters[0].mc
	}

	var exporters []consumer.MetricsConsumer
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.mc)
	}

	// Create a junction point that fans out to all exporters.
	return fanoutprocessor.NewMetricsProcessor(exporters)
}
