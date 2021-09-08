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

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

// builtPipeline is a pipeline that is built based on a config.
// It can have a trace and/or a metrics consumer (the consumer is either the first
// processor in the pipeline or the exporter if pipeline has no processors).
type builtPipeline struct {
	logger  *zap.Logger
	firstTC consumer.Traces
	firstMC consumer.Metrics
	firstLC consumer.Logs

	// MutatesData is set to true if any processors in the pipeline
	// can mutate the TraceData or MetricsData input argument.
	MutatesData bool

	processors []component.Processor
}

// BuiltPipelines is a map of build pipelines created from pipeline configs.
type BuiltPipelines map[*config.Pipeline]*builtPipeline

func (bps BuiltPipelines) StartProcessors(ctx context.Context, host component.Host) error {
	for _, bp := range bps {
		bp.logger.Info("Pipeline is starting...")
		hostWrapper := newHostWrapper(host, bp.logger)
		// Start in reverse order, starting from the back of processors pipeline.
		// This is important so that processors that are earlier in the pipeline and
		// reference processors that are later in the pipeline do not start sending
		// data to later pipelines which are not yet started.
		for i := len(bp.processors) - 1; i >= 0; i-- {
			if err := bp.processors[i].Start(ctx, hostWrapper); err != nil {
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

	return consumererror.Combine(errs)
}

// pipelinesBuilder builds Pipelines from config.
type pipelinesBuilder struct {
	logger         *zap.Logger
	tracerProvider trace.TracerProvider
	buildInfo      component.BuildInfo
	config         *config.Config
	exporters      Exporters
	factories      map[config.Type]component.ProcessorFactory
}

// BuildPipelines builds pipeline processors from config. Requires exporters to be already
// built via BuildExporters.
func BuildPipelines(
	logger *zap.Logger,
	tracerProvider trace.TracerProvider,
	buildInfo component.BuildInfo,
	config *config.Config,
	exporters Exporters,
	factories map[config.Type]component.ProcessorFactory,
) (BuiltPipelines, error) {
	pb := &pipelinesBuilder{logger, tracerProvider, buildInfo, config, exporters, factories}

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
func (pb *pipelinesBuilder) buildPipeline(ctx context.Context, pipelineCfg *config.Pipeline) (*builtPipeline, error) {

	// BuildProcessors the pipeline backwards.

	// First create a consumer junction point that fans out the data to all exporters.
	var tc consumer.Traces
	var mc consumer.Metrics
	var lc consumer.Logs

	switch pipelineCfg.InputType {
	case config.TracesDataType:
		tc = pb.buildFanoutExportersTracesConsumer(pipelineCfg.Exporters)
	case config.MetricsDataType:
		mc = pb.buildFanoutExportersMetricsConsumer(pipelineCfg.Exporters)
	case config.LogsDataType:
		lc = pb.buildFanoutExportersLogsConsumer(pipelineCfg.Exporters)
	}

	mutatesConsumedData := false

	processors := make([]component.Processor, len(pipelineCfg.Processors))

	// Now build the processors backwards, starting from the last one.
	// The last processor points to consumer which fans out to exporters, then
	// the processor itself becomes a consumer for the one that precedes it in
	// in the pipeline and so on.
	for i := len(pipelineCfg.Processors) - 1; i >= 0; i-- {
		procID := pipelineCfg.Processors[i]

		procCfg, existsCfg := pb.config.Processors[procID]
		if !existsCfg {
			return nil, fmt.Errorf("processor %q is not configured", procID)
		}

		factory, existsFactory := pb.factories[procID.Type()]
		if !existsFactory {
			return nil, fmt.Errorf("processor factory for type %q is not configured", procID.Type())
		}

		// This processor must point to the next consumer and then
		// it becomes the next for the previous one (previous in the pipeline,
		// which we will build in the next loop iteration).
		var err error
		set := component.ProcessorCreateSettings{
			TelemetryCreateSettings: component.TelemetryCreateSettings{
				Logger:         pb.logger.With(zap.String(zapKindKey, zapKindProcessor), zap.String(zapNameKey, procID.String())),
				TracerProvider: pb.tracerProvider,
			},
			BuildInfo: pb.buildInfo,
		}

		switch pipelineCfg.InputType {
		case config.TracesDataType:
			var proc component.TracesProcessor
			proc, err = factory.CreateTracesProcessor(ctx, set, procCfg, tc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.Capabilities().MutatesData
			}
			processors[i] = proc
			tc = proc
		case config.MetricsDataType:
			var proc component.MetricsProcessor
			proc, err = factory.CreateMetricsProcessor(ctx, set, procCfg, mc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.Capabilities().MutatesData
			}
			processors[i] = proc
			mc = proc

		case config.LogsDataType:
			var proc component.LogsProcessor
			proc, err = factory.CreateLogsProcessor(ctx, set, procCfg, lc)
			if proc != nil {
				mutatesConsumedData = mutatesConsumedData || proc.Capabilities().MutatesData
			}
			processors[i] = proc
			lc = proc

		default:
			return nil, fmt.Errorf("error creating processor %q in pipeline %q, data type %s is not supported",
				procID, pipelineCfg.Name, pipelineCfg.InputType)
		}

		if err != nil {
			return nil, fmt.Errorf("error creating processor %q in pipeline %q: %v",
				procID, pipelineCfg.Name, err)
		}

		// Check if the factory really created the processor.
		if tc == nil && mc == nil && lc == nil {
			return nil, fmt.Errorf("factory for %v produced a nil processor", procID)
		}
	}

	pipelineLogger := pb.logger.With(zap.String("pipeline_name", pipelineCfg.Name),
		zap.String("pipeline_datatype", string(pipelineCfg.InputType)))
	pipelineLogger.Info("Pipeline was built.")

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
func (pb *pipelinesBuilder) getBuiltExportersByIDs(exporterIDs []config.ComponentID) []*builtExporter {
	var result []*builtExporter
	for _, expID := range exporterIDs {
		exporter := pb.exporters[expID]
		result = append(result, exporter)
	}

	return result
}

func (pb *pipelinesBuilder) buildFanoutExportersTracesConsumer(exporterIDs []config.ComponentID) consumer.Traces {
	builtExporters := pb.getBuiltExportersByIDs(exporterIDs)

	var exporters []consumer.Traces
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.getTracesExporter())
	}

	// Create a junction point that fans out to all exporters.
	return fanoutconsumer.NewTraces(exporters)
}

func (pb *pipelinesBuilder) buildFanoutExportersMetricsConsumer(exporterIDs []config.ComponentID) consumer.Metrics {
	builtExporters := pb.getBuiltExportersByIDs(exporterIDs)

	var exporters []consumer.Metrics
	for _, builtExp := range builtExporters {
		exporters = append(exporters, builtExp.getMetricExporter())
	}

	// Create a junction point that fans out to all exporters.
	return fanoutconsumer.NewMetrics(exporters)
}

func (pb *pipelinesBuilder) buildFanoutExportersLogsConsumer(exporterIDs []config.ComponentID) consumer.Logs {
	builtExporters := pb.getBuiltExportersByIDs(exporterIDs)

	exporters := make([]consumer.Logs, len(builtExporters))
	for i, builtExp := range builtExporters {
		exporters[i] = builtExp.getLogExporter()
	}

	// Create a junction point that fans out to all exporters.
	return fanoutconsumer.NewLogs(exporters)
}
