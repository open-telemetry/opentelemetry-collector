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

package service // import "go.opentelemetry.io/collector/service"

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

const (
	zPipelineName  = "zpipelinename"
	zComponentName = "zcomponentname"
	zComponentKind = "zcomponentkind"
)

type pipelines interface {
	StartAll(ctx context.Context, host component.Host) error
	ShutdownAll(ctx context.Context) error
	GetExporters() map[component.DataType]map[component.ID]component.Component
	HandleZPages(w http.ResponseWriter, r *http.Request)
}

var _ pipelines = (*builtPipelines)(nil)

// baseConsumer redeclared here since not public in consumer package. May consider to make that public.
type baseConsumer interface {
	Capabilities() consumer.Capabilities
}

type builtComponent struct {
	id   component.ID
	comp component.Component
}

type builtPipeline struct {
	lastConsumer baseConsumer

	receivers  []builtComponent
	processors []builtComponent
	exporters  []builtComponent
}

// builtPipelines is set of all pipelines created from exporter configs.
type builtPipelines struct {
	telemetry component.TelemetrySettings

	allReceivers map[component.DataType]map[component.ID]component.Component
	allExporters map[component.DataType]map[component.ID]component.Component

	pipelines map[component.ID]*builtPipeline
}

// StartAll starts all pipelines.
//
// Start with exporters, processors (in reverse configured order), then receivers.
// This is important so that components that are earlier in the pipeline and reference components that are
// later in the pipeline do not start sending data to later components which are not yet started.
func (bps *builtPipelines) StartAll(ctx context.Context, host component.Host) error {
	bps.telemetry.Logger.Info("Starting exporters...")
	for dt, expByID := range bps.allExporters {
		for expID, exp := range expByID {
			expLogger := components.ExporterLogger(bps.telemetry.Logger, expID, dt)
			expLogger.Info("Exporter is starting...")
			if err := exp.Start(ctx, components.NewHostWrapper(host, expLogger)); err != nil {
				return err
			}
			expLogger.Info("Exporter started.")
		}
	}

	bps.telemetry.Logger.Info("Starting processors...")
	for pipelineID, bp := range bps.pipelines {
		for i := len(bp.processors) - 1; i >= 0; i-- {
			procLogger := components.ProcessorLogger(bps.telemetry.Logger, bp.processors[i].id, pipelineID)
			procLogger.Info("Processor is starting...")
			if err := bp.processors[i].comp.Start(ctx, components.NewHostWrapper(host, procLogger)); err != nil {
				return err
			}
			procLogger.Info("Processor started.")
		}
	}

	bps.telemetry.Logger.Info("Starting receivers...")
	for dt, recvByID := range bps.allReceivers {
		for recvID, recv := range recvByID {
			recvLogger := components.ReceiverLogger(bps.telemetry.Logger, recvID, dt)
			recvLogger.Info("Receiver is starting...")
			if err := recv.Start(ctx, components.NewHostWrapper(host, recvLogger)); err != nil {
				return err
			}
			recvLogger.Info("Receiver started.")
		}
	}
	return nil
}

// ShutdownAll stops all pipelines.
//
// Shutdown order is the reverse of starting: receivers, processors, then exporters.
// This gives senders a chance to send all their data to a not "shutdown" component.
func (bps *builtPipelines) ShutdownAll(ctx context.Context) error {
	var errs error
	bps.telemetry.Logger.Info("Stopping receivers...")
	for _, recvByID := range bps.allReceivers {
		for _, recv := range recvByID {
			errs = multierr.Append(errs, recv.Shutdown(ctx))
		}
	}

	bps.telemetry.Logger.Info("Stopping processors...")
	for _, bp := range bps.pipelines {
		for _, p := range bp.processors {
			errs = multierr.Append(errs, p.comp.Shutdown(ctx))
		}
	}

	bps.telemetry.Logger.Info("Stopping exporters...")
	for _, expByID := range bps.allExporters {
		for _, exp := range expByID {
			errs = multierr.Append(errs, exp.Shutdown(ctx))
		}
	}

	return errs
}

func (bps *builtPipelines) GetExporters() map[component.DataType]map[component.ID]component.Component {
	exportersMap := make(map[component.DataType]map[component.ID]component.Component)

	exportersMap[component.DataTypeTraces] = make(map[component.ID]component.Component, len(bps.allExporters[component.DataTypeTraces]))
	exportersMap[component.DataTypeMetrics] = make(map[component.ID]component.Component, len(bps.allExporters[component.DataTypeMetrics]))
	exportersMap[component.DataTypeLogs] = make(map[component.ID]component.Component, len(bps.allExporters[component.DataTypeLogs]))

	for dt, expByID := range bps.allExporters {
		for expID, exp := range expByID {
			exportersMap[dt][expID] = exp
		}
	}

	return exportersMap
}

// pipelinesSettings holds configuration for building builtPipelines.
type pipelinesSettings struct {
	Telemetry component.TelemetrySettings
	BuildInfo component.BuildInfo

	ReceiverBuilder  *receiver.Builder
	ProcessorBuilder *processor.Builder
	ExporterBuilder  *exporter.Builder
	ConnectorBuilder *connector.Builder

	// PipelineConfigs is a map of component.ID to PipelineConfig.
	PipelineConfigs map[component.ID]*PipelineConfig
}

// buildPipelines builds all pipelines from config.
func buildPipelines(ctx context.Context, set pipelinesSettings) (pipelines, error) {
	exps := &builtPipelines{
		telemetry:    set.Telemetry,
		allReceivers: make(map[component.DataType]map[component.ID]component.Component),
		allExporters: make(map[component.DataType]map[component.ID]component.Component),
		pipelines:    make(map[component.ID]*builtPipeline, len(set.PipelineConfigs)),
	}

	receiversConsumers := make(map[component.DataType]map[component.ID][]baseConsumer)

	// Iterate over all pipelines, and create exporters, then processors.
	// Receivers cannot be created since we need to know all consumers, a.k.a. we need all pipelines build up to the
	// first processor.
	for pipelineID, pipeline := range set.PipelineConfigs {
		// The data type of the pipeline defines what data type each exporter is expected to receive.
		if _, ok := exps.allExporters[pipelineID.Type()]; !ok {
			exps.allExporters[pipelineID.Type()] = make(map[component.ID]component.Component)
		}
		expByID := exps.allExporters[pipelineID.Type()]

		bp := &builtPipeline{
			receivers:  make([]builtComponent, len(pipeline.Receivers)),
			processors: make([]builtComponent, len(pipeline.Processors)),
			exporters:  make([]builtComponent, len(pipeline.Exporters)),
		}
		exps.pipelines[pipelineID] = bp

		// Iterate over all Exporters for this pipeline.
		for i, expID := range pipeline.Exporters {
			// If already created an exporter for this [DataType, ComponentID] nothing to do, will reuse this instance.
			if exp, ok := expByID[expID]; ok {
				bp.exporters[i] = builtComponent{id: expID, comp: exp}
				continue
			}

			exp, err := buildExporter(ctx, expID, set.Telemetry, set.BuildInfo, set.ExporterBuilder, pipelineID)
			if err != nil {
				return nil, err
			}

			bp.exporters[i] = builtComponent{id: expID, comp: exp}
			expByID[expID] = exp
		}

		// Build a fan out consumer to all exporters.
		switch pipelineID.Type() {
		case component.DataTypeTraces:
			bp.lastConsumer = buildFanOutExportersTracesConsumer(bp.exporters)
		case component.DataTypeMetrics:
			bp.lastConsumer = buildFanOutExportersMetricsConsumer(bp.exporters)
		case component.DataTypeLogs:
			bp.lastConsumer = buildFanOutExportersLogsConsumer(bp.exporters)
		default:
			return nil, fmt.Errorf("create fan-out exporter in pipeline %q, data type %q is not supported", pipelineID, pipelineID.Type())
		}

		mutatesConsumedData := bp.lastConsumer.Capabilities().MutatesData
		// Build the processors backwards, starting from the last one.
		// The last processor points to fan out consumer to all Exporters, then the processor itself becomes a
		// consumer for the one that precedes it in the pipeline and so on.
		for i := len(pipeline.Processors) - 1; i >= 0; i-- {
			procID := pipeline.Processors[i]
			proc, err := buildProcessor(ctx, procID, set.Telemetry, set.BuildInfo, set.ProcessorBuilder, pipelineID, bp.lastConsumer)
			if err != nil {
				return nil, err
			}

			bp.processors[i] = builtComponent{id: procID, comp: proc}
			bp.lastConsumer = proc.(baseConsumer)
			mutatesConsumedData = mutatesConsumedData || bp.lastConsumer.Capabilities().MutatesData
		}

		// Some consumers may not correctly implement the Capabilities, and ignore the next consumer when calculated the Capabilities.
		// Because of this wrap the first consumer if any consumers in the pipeline mutate the data and the first says that it doesn't.
		switch pipelineID.Type() {
		case component.DataTypeTraces:
			bp.lastConsumer = capabilityconsumer.NewTraces(bp.lastConsumer.(consumer.Traces), consumer.Capabilities{MutatesData: mutatesConsumedData})
		case component.DataTypeMetrics:
			bp.lastConsumer = capabilityconsumer.NewMetrics(bp.lastConsumer.(consumer.Metrics), consumer.Capabilities{MutatesData: mutatesConsumedData})
		case component.DataTypeLogs:
			bp.lastConsumer = capabilityconsumer.NewLogs(bp.lastConsumer.(consumer.Logs), consumer.Capabilities{MutatesData: mutatesConsumedData})
		default:
			return nil, fmt.Errorf("create cap consumer in pipeline %q, data type %q is not supported", pipelineID, pipelineID.Type())
		}

		// The data type of the pipeline defines what data type each exporter is expected to receive.
		if _, ok := receiversConsumers[pipelineID.Type()]; !ok {
			receiversConsumers[pipelineID.Type()] = make(map[component.ID][]baseConsumer)
		}
		recvConsByID := receiversConsumers[pipelineID.Type()]
		// Iterate over all Receivers for this pipeline and just append the lastConsumer as a consumer for the receiver.
		for _, recvID := range pipeline.Receivers {
			recvConsByID[recvID] = append(recvConsByID[recvID], bp.lastConsumer)
		}
	}

	// Now that we built the `receiversConsumers` map, we can build the receivers as well.
	for pipelineID, pipeline := range set.PipelineConfigs {
		// The data type of the pipeline defines what data type each exporter is expected to receive.
		if _, ok := exps.allReceivers[pipelineID.Type()]; !ok {
			exps.allReceivers[pipelineID.Type()] = make(map[component.ID]component.Component)
		}
		recvByID := exps.allReceivers[pipelineID.Type()]
		bp := exps.pipelines[pipelineID]

		// Iterate over all Receivers for this pipeline.
		for i, recvID := range pipeline.Receivers {
			// If already created a receiver for this [DataType, ComponentID] nothing to do.
			if exp, ok := recvByID[recvID]; ok {
				bp.receivers[i] = builtComponent{id: recvID, comp: exp}
				continue
			}

			recv, err := buildReceiver(ctx, recvID, set.Telemetry, set.BuildInfo, set.ReceiverBuilder, pipelineID, receiversConsumers[pipelineID.Type()][recvID])
			if err != nil {
				return nil, err
			}

			bp.receivers[i] = builtComponent{id: recvID, comp: recv}
			recvByID[recvID] = recv
		}
	}
	return exps, nil
}

func buildFanOutExportersTracesConsumer(exporters []builtComponent) consumer.Traces {
	consumers := make([]consumer.Traces, 0, len(exporters))
	for _, exp := range exporters {
		consumers = append(consumers, exp.comp.(consumer.Traces))
	}
	// Create a junction point that fans out to all allExporters.
	return fanoutconsumer.NewTraces(consumers)
}

func buildFanOutExportersMetricsConsumer(exporters []builtComponent) consumer.Metrics {
	consumers := make([]consumer.Metrics, 0, len(exporters))
	for _, exp := range exporters {
		consumers = append(consumers, exp.comp.(consumer.Metrics))
	}
	// Create a junction point that fans out to all allExporters.
	return fanoutconsumer.NewMetrics(consumers)
}

func buildFanOutExportersLogsConsumer(exporters []builtComponent) consumer.Logs {
	consumers := make([]consumer.Logs, 0, len(exporters))
	for _, exp := range exporters {
		consumers = append(consumers, exp.comp.(consumer.Logs))
	}
	// Create a junction point that fans out to all allExporters.
	return fanoutconsumer.NewLogs(consumers)
}

func buildExporter(
	ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *exporter.Builder,
	pipelineID component.ID,
) (exp component.Component, err error) {
	set := exporter.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ExporterLogger(set.TelemetrySettings.Logger, componentID, pipelineID.Type())
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		exp, err = builder.CreateTraces(ctx, set)
	case component.DataTypeMetrics:
		exp, err = builder.CreateMetrics(ctx, set)
	case component.DataTypeLogs:
		exp, err = builder.CreateLogs(ctx, set)
	default:
		return nil, fmt.Errorf("error creating exporter %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q exporter, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return exp, nil
}

func buildProcessor(ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *processor.Builder,
	pipelineID component.ID,
	next baseConsumer,
) (proc component.Component, err error) {
	set := processor.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ProcessorLogger(set.TelemetrySettings.Logger, componentID, pipelineID)
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		proc, err = builder.CreateTraces(ctx, set, next.(consumer.Traces))
	case component.DataTypeMetrics:
		proc, err = builder.CreateMetrics(ctx, set, next.(consumer.Metrics))
	case component.DataTypeLogs:
		proc, err = builder.CreateLogs(ctx, set, next.(consumer.Logs))
	default:
		return nil, fmt.Errorf("error creating processor %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q processor, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return proc, nil
}

func buildReceiver(ctx context.Context,
	componentID component.ID,
	tel component.TelemetrySettings,
	info component.BuildInfo,
	builder *receiver.Builder,
	pipelineID component.ID,
	nexts []baseConsumer,
) (recv component.Component, err error) {
	set := receiver.CreateSettings{ID: componentID, TelemetrySettings: tel, BuildInfo: info}
	set.TelemetrySettings.Logger = components.ReceiverLogger(tel.Logger, componentID, pipelineID.Type())
	switch pipelineID.Type() {
	case component.DataTypeTraces:
		var consumers []consumer.Traces
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		recv, err = builder.CreateTraces(ctx, set, fanoutconsumer.NewTraces(consumers))
	case component.DataTypeMetrics:
		var consumers []consumer.Metrics
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		recv, err = builder.CreateMetrics(ctx, set, fanoutconsumer.NewMetrics(consumers))
	case component.DataTypeLogs:
		var consumers []consumer.Logs
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		recv, err = builder.CreateLogs(ctx, set, fanoutconsumer.NewLogs(consumers))
	default:
		return nil, fmt.Errorf("error creating receiver %q in pipeline %q, data type %q is not supported", set.ID, pipelineID, pipelineID.Type())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %q receiver, in pipeline %q: %w", set.ID, pipelineID, err)
	}
	return recv, nil
}

func (bps *builtPipelines) HandleZPages(w http.ResponseWriter, r *http.Request) {
	handleZPages(w, r, bps.pipelines)
}

func (bp *builtPipeline) receiverIDs() []string {
	ids := make([]string, 0, len(bp.receivers))
	for _, bc := range bp.receivers {
		ids = append(ids, bc.id.String())
	}
	return ids
}

func (bp *builtPipeline) processorIDs() []string {
	ids := make([]string, 0, len(bp.processors))
	for _, bc := range bp.processors {
		ids = append(ids, bc.id.String())
	}
	return ids
}

func (bp *builtPipeline) exporterIDs() []string {
	ids := make([]string, 0, len(bp.exporters))
	for _, bc := range bp.exporters {
		ids = append(ids, bc.id.String())
	}
	return ids
}

func (bp *builtPipeline) mutatesData() bool {
	return bp.lastConsumer.Capabilities().MutatesData
}
