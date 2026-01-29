// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package graph contains the internal graph representation of the pipelines.
//
// [Build] is the constructor for a [Graph] object.  The method calls out to helpers that transform the graph from a config
// to a DAG of components.  The configuration undergoes additional validation here as well, and is used to instantiate
// the components of the pipeline.
//
// [Graph.StartAll] starts all components in each pipeline.
//
// [Graph.ShutdownAll] stops all components in each pipeline.
package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/hostcapabilities"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/status"
	"go.opentelemetry.io/collector/service/pipelines"
)

// Settings holds configuration for building builtPipelines.
type Settings struct {
	Telemetry component.TelemetrySettings
	BuildInfo component.BuildInfo

	ReceiverBuilder  *builders.ReceiverBuilder
	ProcessorBuilder *builders.ProcessorBuilder
	ExporterBuilder  *builders.ExporterBuilder
	ConnectorBuilder *builders.ConnectorBuilder

	// PipelineConfigs is a map of component.ID to PipelineConfig.
	PipelineConfigs pipelines.Config

	ReportStatus status.ServiceStatusFunc
}

type Graph struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelines map[pipeline.ID]*pipelineNodes

	// Keep track of status source per node
	instanceIDs map[int64]*componentstatus.InstanceID

	telemetry component.TelemetrySettings
}

// Build builds a full pipeline graph.
// Build also validates the configuration of the pipelines and does the actual initialization of each Component in the Graph.
func Build(ctx context.Context, set Settings) (*Graph, error) {
	pipelines := &Graph{
		componentGraph: simple.NewDirectedGraph(),
		pipelines:      make(map[pipeline.ID]*pipelineNodes, len(set.PipelineConfigs)),
		instanceIDs:    make(map[int64]*componentstatus.InstanceID),
		telemetry:      set.Telemetry,
	}
	for pipelineID := range set.PipelineConfigs {
		pipelines.pipelines[pipelineID] = &pipelineNodes{
			receivers: make(map[int64]graph.Node),
			exporters: make(map[int64]graph.Node),
		}
	}
	if err := pipelines.createNodes(set); err != nil {
		return nil, err
	}
	pipelines.createEdges()
	err := pipelines.buildComponents(ctx, set)
	return pipelines, err
}

// Creates a node for each instance of a component and adds it to the graph.
// Validates that connectors are configured to export and receive correctly.
func (g *Graph) createNodes(set Settings) error {
	// Build a list of all connectors for easy reference.
	connectors := make(map[component.ID]struct{})

	// Keep track of connectors and where they are used. (map[connectorID][]pipelineID).
	connectorsAsExporter := make(map[component.ID][]pipeline.ID)
	connectorsAsReceiver := make(map[component.ID][]pipeline.ID)

	// Build each pipelineNodes struct for each pipeline by parsing the pipelineCfg.
	// Also populates the connectors, connectorsAsExporter and connectorsAsReceiver maps.
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		pipe := g.pipelines[pipelineID]
		for _, recvID := range pipelineCfg.Receivers {
			// Checks if this receiver is a connector or a regular receiver.
			if set.ConnectorBuilder.IsConfigured(recvID) {
				connectors[recvID] = struct{}{}
				connectorsAsReceiver[recvID] = append(connectorsAsReceiver[recvID], pipelineID)
				continue
			}
			rcvrNode := g.createReceiver(pipelineID, recvID)
			pipe.receivers[rcvrNode.ID()] = rcvrNode
		}

		pipe.capabilitiesNode = newCapabilitiesNode(pipelineID)

		for _, procID := range pipelineCfg.Processors {
			procNode := g.createProcessor(pipelineID, procID)
			pipe.processors = append(pipe.processors, procNode)
		}

		pipe.fanOutNode = newFanOutNode(pipelineID)

		for _, exprID := range pipelineCfg.Exporters {
			if set.ConnectorBuilder.IsConfigured(exprID) {
				connectors[exprID] = struct{}{}
				connectorsAsExporter[exprID] = append(connectorsAsExporter[exprID], pipelineID)
				continue
			}
			expNode := g.createExporter(pipelineID, exprID)
			pipe.exporters[expNode.ID()] = expNode
		}
	}

	for connID := range connectors {
		factory := set.ConnectorBuilder.Factory(connID.Type())
		if factory == nil {
			return fmt.Errorf("connector factory not available for: %q", connID.Type())
		}
		connFactory := factory.(connector.Factory)

		expTypes := make(map[pipeline.Signal]bool)
		for _, pipelineID := range connectorsAsExporter[connID] {
			// The presence of each key indicates how the connector is used as an exporter.
			// The value is initially set to false. Later we will set the value to true *if* we
			// confirm that there is a supported corresponding use as a receiver.
			expTypes[pipelineID.Signal()] = false
		}
		recTypes := make(map[pipeline.Signal]bool)
		for _, pipelineID := range connectorsAsReceiver[connID] {
			// The presence of each key indicates how the connector is used as a receiver.
			// The value is initially set to false. Later we will set the value to true *if* we
			// confirm that there is a supported corresponding use as an exporter.
			recTypes[pipelineID.Signal()] = false
		}

		for expType := range expTypes {
			for recType := range recTypes {
				// Typechecks the connector's receiving and exporting datatypes.
				if connectorStability(connFactory, expType, recType) != component.StabilityLevelUndefined {
					expTypes[expType] = true
					recTypes[recType] = true
				}
			}
		}

		for expType, supportedUse := range expTypes {
			if supportedUse {
				continue
			}
			return fmt.Errorf("connector %q used as exporter in %v pipeline but not used in any supported receiver pipeline", connID, formatPipelineNamesWithSignal(connectorsAsExporter[connID], expType))
		}
		for recType, supportedUse := range recTypes {
			if supportedUse {
				continue
			}
			return fmt.Errorf("connector %q used as receiver in %v pipeline but not used in any supported exporter pipeline", connID, formatPipelineNamesWithSignal(connectorsAsReceiver[connID], recType))
		}

		for _, eID := range connectorsAsExporter[connID] {
			for _, rID := range connectorsAsReceiver[connID] {
				if connectorStability(connFactory, eID.Signal(), rID.Signal()) == component.StabilityLevelUndefined {
					// Connector is not supported for this combination, but we know it is used correctly elsewhere
					continue
				}
				connNode := g.createConnector(eID, rID, connID)

				g.pipelines[eID].exporters[connNode.ID()] = connNode
				g.pipelines[rID].receivers[connNode.ID()] = connNode
			}
		}
	}
	return nil
}

// formatPipelineNamesWithSignal formats pipeline name with signal as "signal[/name]" format.
func formatPipelineNamesWithSignal(pipelineIDs []pipeline.ID, signal pipeline.Signal) []string {
	var formatted []string
	for _, pid := range pipelineIDs {
		if pid.Signal() == signal {
			formatted = append(formatted, pid.String())
		}
	}
	return formatted
}

func (g *Graph) createReceiver(pipelineID pipeline.ID, recvID component.ID) *receiverNode {
	rcvrNode := newReceiverNode(pipelineID.Signal(), recvID)
	if node := g.componentGraph.Node(rcvrNode.ID()); node != nil {
		instanceID := g.instanceIDs[node.ID()]
		g.instanceIDs[node.ID()] = instanceID.WithPipelines(pipelineID)
		return node.(*receiverNode)
	}
	g.componentGraph.AddNode(rcvrNode)
	g.instanceIDs[rcvrNode.ID()] = componentstatus.NewInstanceID(
		recvID, component.KindReceiver, pipelineID,
	)
	return rcvrNode
}

func (g *Graph) createProcessor(pipelineID pipeline.ID, procID component.ID) *processorNode {
	procNode := newProcessorNode(pipelineID, procID)
	g.componentGraph.AddNode(procNode)
	g.instanceIDs[procNode.ID()] = componentstatus.NewInstanceID(
		procID, component.KindProcessor, pipelineID,
	)
	return procNode
}

func (g *Graph) createExporter(pipelineID pipeline.ID, exprID component.ID) *exporterNode {
	expNode := newExporterNode(pipelineID.Signal(), exprID)
	if node := g.componentGraph.Node(expNode.ID()); node != nil {
		instanceID := g.instanceIDs[expNode.ID()]
		g.instanceIDs[expNode.ID()] = instanceID.WithPipelines(pipelineID)
		return node.(*exporterNode)
	}
	g.componentGraph.AddNode(expNode)
	g.instanceIDs[expNode.ID()] = componentstatus.NewInstanceID(
		expNode.componentID, component.KindExporter, pipelineID,
	)
	return expNode
}

func (g *Graph) createConnector(exprPipelineID, rcvrPipelineID pipeline.ID, connID component.ID) *connectorNode {
	connNode := newConnectorNode(exprPipelineID.Signal(), rcvrPipelineID.Signal(), connID)
	if node := g.componentGraph.Node(connNode.ID()); node != nil {
		instanceID := g.instanceIDs[connNode.ID()]
		g.instanceIDs[connNode.ID()] = instanceID.WithPipelines(exprPipelineID, rcvrPipelineID)
		return node.(*connectorNode)
	}
	g.componentGraph.AddNode(connNode)
	g.instanceIDs[connNode.ID()] = componentstatus.NewInstanceID(
		connNode.componentID, component.KindConnector, exprPipelineID, rcvrPipelineID,
	)
	return connNode
}

// Iterates through the pipelines and creates edges between components.
func (g *Graph) createEdges() {
	for _, pg := range g.pipelines {
		// Draw edges from each receiver to the capability node.
		for _, receiver := range pg.receivers {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(receiver, pg.capabilitiesNode))
		}

		// Iterates through processors, chaining them together.  starts with the capabilities node.
		var from, to graph.Node
		from = pg.capabilitiesNode
		for _, processor := range pg.processors {
			to = processor
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, to))
			from = processor
		}
		// Always inserts a fanout node before any exporters. If there is only one
		// exporter, the fanout node is still created and acts as a noop.
		to = pg.fanOutNode
		g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, to))

		for _, exporter := range pg.exporters {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(pg.fanOutNode, exporter))
		}
	}
}

// Uses the already built graph g to instantiate the actual components for each component of each pipeline.
// Handles calling the factories for each component - and hooking up each component to the next.
// Also calculates whether each pipeline mutates data so the receiver can know whether it needs to clone the data.
func (g *Graph) buildComponents(ctx context.Context, set Settings) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return cycleErr(err, topo.DirectedCyclesIn(g.componentGraph))
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]

		switch n := node.(type) {
		case *receiverNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ReceiverBuilder, g.nextConsumers(n.ID()))
		case *processorNode:
			// nextConsumers is guaranteed to be length 1.  Either it is the next processor or it is the fanout node for the exporters.
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ProcessorBuilder, g.nextConsumers(n.ID())[0])
		case *exporterNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ExporterBuilder)
		case *connectorNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ConnectorBuilder, g.nextConsumers(n.ID()))
		case *capabilitiesNode:
			capability := consumer.Capabilities{
				// The fanOutNode represents the aggregate capabilities of the exporters in the pipeline.
				MutatesData: g.pipelines[n.pipelineID].fanOutNode.getConsumer().Capabilities().MutatesData,
			}
			for _, proc := range g.pipelines[n.pipelineID].processors {
				capability.MutatesData = capability.MutatesData || proc.(*processorNode).getConsumer().Capabilities().MutatesData
			}
			next := g.nextConsumers(n.ID())[0]
			switch n.pipelineID.Signal() {
			case pipeline.SignalTraces:
				cc := capabilityconsumer.NewTraces(next.(consumer.Traces), capability)
				n.baseConsumer = cc
				n.ConsumeTracesFunc = cc.ConsumeTraces
			case pipeline.SignalMetrics:
				cc := capabilityconsumer.NewMetrics(next.(consumer.Metrics), capability)
				n.baseConsumer = cc
				n.ConsumeMetricsFunc = cc.ConsumeMetrics
			case pipeline.SignalLogs:
				cc := capabilityconsumer.NewLogs(next.(consumer.Logs), capability)
				n.baseConsumer = cc
				n.ConsumeLogsFunc = cc.ConsumeLogs
			case xpipeline.SignalProfiles:
				cc := capabilityconsumer.NewProfiles(next.(xconsumer.Profiles), capability)
				n.baseConsumer = cc
				n.ConsumeProfilesFunc = cc.ConsumeProfiles
			}
		case *fanOutNode:
			nexts := g.nextConsumers(n.ID())
			switch n.pipelineID.Signal() {
			case pipeline.SignalTraces:
				consumers := make([]consumer.Traces, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(consumer.Traces))
				}
				n.baseConsumer = fanoutconsumer.NewTraces(consumers)
			case pipeline.SignalMetrics:
				consumers := make([]consumer.Metrics, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(consumer.Metrics))
				}
				n.baseConsumer = fanoutconsumer.NewMetrics(consumers)
			case pipeline.SignalLogs:
				consumers := make([]consumer.Logs, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(consumer.Logs))
				}
				n.baseConsumer = fanoutconsumer.NewLogs(consumers)
			case xpipeline.SignalProfiles:
				consumers := make([]xconsumer.Profiles, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(xconsumer.Profiles))
				}
				n.baseConsumer = fanoutconsumer.NewProfiles(consumers)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Find all nodes
func (g *Graph) nextConsumers(nodeID int64) []baseConsumer {
	nextNodes := g.componentGraph.From(nodeID)
	nexts := make([]baseConsumer, 0, nextNodes.Len())
	for nextNodes.Next() {
		nexts = append(nexts, nextNodes.Node().(consumerNode).getConsumer())
	}
	return nexts
}

// A node-based representation of a pipeline configuration.
type pipelineNodes struct {
	// Use map to assist with deduplication of connector instances.
	receivers map[int64]graph.Node

	// The node to which receivers emit. Passes through to processors.
	// Easily accessible as the first node in a pipeline.
	*capabilitiesNode

	// The order of processors is very important. Therefore use a slice for processors.
	processors []graph.Node

	// Emits to exporters.
	*fanOutNode

	// Use map to assist with deduplication of connector instances.
	exporters map[int64]graph.Node
}

func (g *Graph) StartAll(ctx context.Context, host *Host) error {
	if host == nil {
		return errors.New("host cannot be nil")
	}

	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return err
	}

	// Start in reverse topological order so that downstream components
	// are started before upstream components. This ensures that each
	// component's consumer is ready to consume.
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		comp, ok := node.(component.Component)

		if !ok {
			// Skip capabilities/fanout nodes
			continue
		}

		instanceID := g.instanceIDs[node.ID()]
		host.Reporter.ReportStatus(
			instanceID,
			componentstatus.NewEvent(componentstatus.StatusStarting),
		)

		if compErr := comp.Start(ctx, &HostWrapper{Host: host, InstanceID: instanceID}); compErr != nil {
			host.Reporter.ReportStatus(
				instanceID,
				componentstatus.NewPermanentErrorEvent(compErr),
			)
			// We log with zap.AddStacktrace(zap.DPanicLevel) to avoid adding the stack trace to the error log
			g.telemetry.Logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).
				Error("Failed to start component",
					zap.Error(compErr),
					zap.String("type", instanceID.Kind().String()),
					zap.String("id", instanceID.ComponentID().String()),
				)
			return fmt.Errorf("failed to start %q %s: %w", instanceID.ComponentID().String(), strings.ToLower(instanceID.Kind().String()), compErr)
		}

		host.Reporter.ReportOKIfStarting(instanceID)
	}
	return nil
}

func (g *Graph) ShutdownAll(ctx context.Context, reporter status.Reporter) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return err
	}

	// Stop in topological order so that upstream components
	// are stopped before downstream components.  This ensures
	// that each component has a chance to drain to its consumer
	// before the consumer is stopped.
	var errs error
	for i := range nodes {
		node := nodes[i]
		comp, ok := node.(component.Component)

		if !ok {
			// Skip capabilities/fanout nodes
			continue
		}

		instanceID := g.instanceIDs[node.ID()]
		reporter.ReportStatus(
			instanceID,
			componentstatus.NewEvent(componentstatus.StatusStopping),
		)

		if compErr := comp.Shutdown(ctx); compErr != nil {
			errs = multierr.Append(errs, compErr)
			reporter.ReportStatus(
				instanceID,
				componentstatus.NewPermanentErrorEvent(compErr),
			)
			continue
		}

		reporter.ReportStatus(
			instanceID,
			componentstatus.NewEvent(componentstatus.StatusStopped),
		)
	}
	return errs
}

func (g *Graph) shutdownReceiverNode(ctx context.Context, nodeID int64, rn *receiverNode, host *Host) error {
	instanceID := g.instanceIDs[nodeID]
	host.Reporter.ReportStatus(
		instanceID,
		componentstatus.NewEvent(componentstatus.StatusStopping),
	)
	if err := rn.Shutdown(ctx); err != nil {
		host.Reporter.ReportStatus(
			instanceID,
			componentstatus.NewPermanentErrorEvent(err),
		)
		return fmt.Errorf("failed to shutdown receiver %q: %w", rn.componentID, err)
	}
	host.Reporter.ReportStatus(
		instanceID,
		componentstatus.NewEvent(componentstatus.StatusStopped),
	)
	return nil
}

// reloadState holds all the state needed during a partial reload operation.
type reloadState struct {
	// Pipelines affected by processor changes.
	affectedPipelines map[pipeline.ID]bool
	// Pipelines needing fanOutNode rebuild due to exporter/connector changes.
	pipelinesNeedingFanOutRebuild map[pipeline.ID]bool

	// Receiver changes.
	receiversToRemove  map[int64]*receiverNode
	receiversToRebuild map[int64]*receiverNode
	receiversToAdd     map[int64]bool

	// Exporter/connector nodes to rebuild.
	exporterNodesToRebuild  map[int64]*exporterNode
	connectorNodesToRebuild map[int64]*connectorNode

	// Receivers that were built (for starting).
	builtReceivers map[int64]bool
}

// Reload performs a partial reload of the graph.
//
// Pipeline structure: receiver -> capabilitiesNode -> processors -> fanOutNode -> exporters
//
// When a component is recreated, all components upstream of it must also be recreated
// because each component stores a reference to its next consumer. For example:
//   - If an exporter config changes, the exporter is recreated
//   - The fanOutNode must be rebuilt to reference the new exporter
//   - All processors must be recreated to get new references to the rebuilt fanOutNode
//   - The capabilitiesNode must be rebuilt to reference the new first processor
//   - All receivers must be recreated to get new references to the rebuilt capabilitiesNode
//
// The reload process has five phases:
//  1. Identify changes: Determine which components changed and which pipelines are affected
//  2. Shutdown: Stop all components that need to be recreated (receivers first, then downstream)
//  3. Update graph: Remove old nodes, create new nodes, wire edges
//  4. Build: Create new component instances (downstream first, then upstream)
//  5. Start: Start all recreated components (downstream first, then receivers last)
//
// Returns true if changes were detected and a partial reload was performed,
// false if no changes were detected. An error is returned if the reload fails.
func (g *Graph) Reload(ctx context.Context, set Settings,
	oldReceiverCfgs, newReceiverCfgs map[component.ID]component.Config,
	receiverFactories map[component.Type]receiver.Factory,
	oldProcessorCfgs, newProcessorCfgs map[component.ID]component.Config,
	processorFactories map[component.Type]processor.Factory,
	oldExporterCfgs, newExporterCfgs map[component.ID]component.Config,
	exporterFactories map[component.Type]exporter.Factory,
	oldConnectorCfgs, newConnectorCfgs map[component.ID]component.Config,
	connectorFactories map[component.Type]connector.Factory,
	host *Host,
) (bool, error) {
	if host == nil {
		return false, errors.New("host cannot be nil")
	}

	// Phase 1: Identify all changes.
	state := g.identifyChanges(set,
		oldReceiverCfgs, newReceiverCfgs,
		oldProcessorCfgs, newProcessorCfgs,
		oldExporterCfgs, newExporterCfgs,
		oldConnectorCfgs, newConnectorCfgs)

	// Check if there's anything to do.
	if len(state.affectedPipelines) == 0 && len(state.receiversToRemove) == 0 &&
		len(state.receiversToRebuild) == 0 && len(state.receiversToAdd) == 0 &&
		len(state.exporterNodesToRebuild) == 0 && len(state.connectorNodesToRebuild) == 0 {
		g.telemetry.Logger.Info("Partial reload: no changes detected")
		return false, nil
	}

	// Phase 2: Shutdown components.
	if err := g.reloadShutdown(ctx, state, host); err != nil {
		return false, err
	}

	// Phase 3: Update graph (remove old nodes, create new nodes, wire edges).
	g.reloadUpdateGraph(set, state)

	// Phase 4: Build components.
	if err := g.reloadBuild(ctx, set, state,
		newReceiverCfgs, receiverFactories,
		newProcessorCfgs, processorFactories,
		newExporterCfgs, exporterFactories,
		newConnectorCfgs, connectorFactories); err != nil {
		return false, err
	}

	// Phase 5: Start components.
	if err := g.reloadStart(ctx, state, host); err != nil {
		return false, err
	}

	g.telemetry.Logger.Info("Partial reload completed successfully",
		zap.Int("affected_pipelines", len(state.affectedPipelines)),
		zap.Int("receivers_added", len(state.receiversToAdd)),
		zap.Int("receivers_rebuilt", len(state.receiversToRebuild)),
		zap.Int("receivers_removed", len(state.receiversToRemove)),
		zap.Int("exporters_rebuilt", len(state.exporterNodesToRebuild)),
		zap.Int("connectors_rebuilt", len(state.connectorNodesToRebuild)))
	return true, nil
}

// identifyChanges analyzes config differences and returns the reload state.
func (g *Graph) identifyChanges(set Settings,
	oldReceiverCfgs, newReceiverCfgs map[component.ID]component.Config,
	oldProcessorCfgs, newProcessorCfgs map[component.ID]component.Config,
	oldExporterCfgs, newExporterCfgs map[component.ID]component.Config,
	oldConnectorCfgs, newConnectorCfgs map[component.ID]component.Config,
) *reloadState {
	state := &reloadState{
		affectedPipelines:             make(map[pipeline.ID]bool),
		pipelinesNeedingFanOutRebuild: make(map[pipeline.ID]bool),
		receiversToRemove:             make(map[int64]*receiverNode),
		receiversToRebuild:            make(map[int64]*receiverNode),
		receiversToAdd:                make(map[int64]bool),
		exporterNodesToRebuild:        make(map[int64]*exporterNode),
		connectorNodesToRebuild:       make(map[int64]*connectorNode),
		builtReceivers:                make(map[int64]bool),
	}

	// Identify pipelines with processor changes.
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		oldPipe := g.pipelines[pipelineID]
		oldProcIDs := make([]component.ID, 0, len(oldPipe.processors))
		for _, node := range oldPipe.processors {
			oldProcIDs = append(oldProcIDs, node.(*processorNode).componentID)
		}
		if !slices.Equal(oldProcIDs, pipelineCfg.Processors) {
			state.affectedPipelines[pipelineID] = true
			continue
		}
		for _, procID := range pipelineCfg.Processors {
			if !reflect.DeepEqual(oldProcessorCfgs[procID], newProcessorCfgs[procID]) {
				state.affectedPipelines[pipelineID] = true
				break
			}
		}
	}

	// Categorize receiver changes.
	currentReceivers := make(map[int64]*receiverNode)
	currentReceiverPipelines := make(map[int64]map[pipeline.ID]bool)
	for pipelineID, pipe := range g.pipelines {
		for nodeID, node := range pipe.receivers {
			if rn, ok := node.(*receiverNode); ok {
				currentReceivers[nodeID] = rn
				if currentReceiverPipelines[nodeID] == nil {
					currentReceiverPipelines[nodeID] = make(map[pipeline.ID]bool)
				}
				currentReceiverPipelines[nodeID][pipelineID] = true
			}
		}
	}

	desiredReceiverPipelines := make(map[int64]map[pipeline.ID]bool)
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		for _, recvID := range pipelineCfg.Receivers {
			if set.ConnectorBuilder.IsConfigured(recvID) {
				continue
			}
			nodeID := newReceiverNode(pipelineID.Signal(), recvID).ID()
			if desiredReceiverPipelines[nodeID] == nil {
				desiredReceiverPipelines[nodeID] = make(map[pipeline.ID]bool)
			}
			desiredReceiverPipelines[nodeID][pipelineID] = true
		}
	}

	for nodeID, rn := range currentReceivers {
		if _, desired := desiredReceiverPipelines[nodeID]; !desired {
			state.receiversToRemove[nodeID] = rn
		} else if !maps.Equal(currentReceiverPipelines[nodeID], desiredReceiverPipelines[nodeID]) ||
			!reflect.DeepEqual(oldReceiverCfgs[rn.componentID], newReceiverCfgs[rn.componentID]) {
			state.receiversToRebuild[nodeID] = rn
		}
	}
	for nodeID := range desiredReceiverPipelines {
		if _, exists := currentReceivers[nodeID]; !exists {
			state.receiversToAdd[nodeID] = true
		}
	}

	// Identify exporter/connector changes.
	exportersToRebuild := make(map[component.ID]bool)
	for expID := range newExporterCfgs {
		if !reflect.DeepEqual(oldExporterCfgs[expID], newExporterCfgs[expID]) {
			exportersToRebuild[expID] = true
		}
	}
	connectorsToRebuild := make(map[component.ID]bool)
	for connID := range newConnectorCfgs {
		if !reflect.DeepEqual(oldConnectorCfgs[connID], newConnectorCfgs[connID]) {
			connectorsToRebuild[connID] = true
		}
	}

	for pipelineID, pipe := range g.pipelines {
		for _, node := range pipe.exporters {
			if en, ok := node.(*exporterNode); ok && exportersToRebuild[en.componentID] {
				state.exporterNodesToRebuild[en.ID()] = en
				state.pipelinesNeedingFanOutRebuild[pipelineID] = true
				// Mark pipeline as affected so processors get rebuilt with the new consumer reference.
				// Processors store a reference to their next consumer (the fanOutNode), so when
				// the fanOutNode is rebuilt, processors need to be rebuilt to get the new reference.
				state.affectedPipelines[pipelineID] = true
			}
			if cn, ok := node.(*connectorNode); ok && connectorsToRebuild[cn.componentID] {
				state.connectorNodesToRebuild[cn.ID()] = cn
				state.pipelinesNeedingFanOutRebuild[pipelineID] = true
				// Mark pipeline as affected so processors get rebuilt with the new consumer reference.
				state.affectedPipelines[pipelineID] = true
			}
		}
	}

	// Receivers in affected pipelines must be rebuilt (not just restarted) because they
	// store consumer references from when they were built. When downstream components
	// (processors, exporters, connectors) are rebuilt, receivers need new consumer references.
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		for nodeID, node := range pipe.receivers {
			rn, ok := node.(*receiverNode)
			if !ok {
				continue
			}
			if state.receiversToRebuild[nodeID] != nil || state.receiversToRemove[nodeID] != nil {
				continue
			}
			state.receiversToRebuild[nodeID] = rn
		}
	}
	for pipelineID := range state.pipelinesNeedingFanOutRebuild {
		if state.affectedPipelines[pipelineID] {
			continue
		}
		pipe := g.pipelines[pipelineID]
		for nodeID, node := range pipe.receivers {
			rn, ok := node.(*receiverNode)
			if !ok {
				continue
			}
			if state.receiversToRebuild[nodeID] != nil || state.receiversToRemove[nodeID] != nil {
				continue
			}
			state.receiversToRebuild[nodeID] = rn
		}
	}

	return state
}

// reloadShutdown stops all components that need to be stopped.
func (g *Graph) reloadShutdown(ctx context.Context, state *reloadState, host *Host) error {
	// Shutdown all receivers first (ensures no data in flight).
	for nodeID, rn := range state.receiversToRemove {
		if err := g.shutdownReceiverNode(ctx, nodeID, rn, host); err != nil {
			return err
		}
	}
	for nodeID, rn := range state.receiversToRebuild {
		if err := g.shutdownReceiverNode(ctx, nodeID, rn, host); err != nil {
			return err
		}
	}

	// Shutdown processors in affected pipelines.
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		for _, node := range pipe.processors {
			pn := node.(*processorNode)
			instanceID := g.instanceIDs[pn.ID()]
			host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
			if err := pn.Shutdown(ctx); err != nil {
				host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
				return fmt.Errorf("failed to shutdown processor %q: %w", pn.componentID, err)
			}
			host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
		}
	}

	// Shutdown exporters being rebuilt.
	for nodeID, en := range state.exporterNodesToRebuild {
		instanceID := g.instanceIDs[nodeID]
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
		if err := en.Shutdown(ctx); err != nil {
			host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
			return fmt.Errorf("failed to shutdown exporter %q: %w", en.componentID, err)
		}
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	}

	// Shutdown connectors being rebuilt.
	for nodeID, cn := range state.connectorNodesToRebuild {
		instanceID := g.instanceIDs[nodeID]
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
		if err := cn.Shutdown(ctx); err != nil {
			host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
			return fmt.Errorf("failed to shutdown connector %q: %w", cn.componentID, err)
		}
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	}

	return nil
}

// reloadUpdateGraph removes old nodes, creates new nodes, and wires edges.
func (g *Graph) reloadUpdateGraph(set Settings, state *reloadState) {
	// Remove receivers being removed or rebuilt.
	for nodeID := range state.receiversToRemove {
		g.componentGraph.RemoveNode(nodeID)
		delete(g.instanceIDs, nodeID)
	}
	for nodeID := range state.receiversToRebuild {
		g.componentGraph.RemoveNode(nodeID)
		delete(g.instanceIDs, nodeID)
	}
	for _, pipe := range g.pipelines {
		for nodeID, node := range pipe.receivers {
			if _, ok := node.(*receiverNode); !ok {
				continue
			}
			if _, rm := state.receiversToRemove[nodeID]; rm {
				delete(pipe.receivers, nodeID)
			}
			if _, rb := state.receiversToRebuild[nodeID]; rb {
				delete(pipe.receivers, nodeID)
			}
		}
	}

	// Remove processors in affected pipelines.
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		for _, node := range pipe.processors {
			pn := node.(*processorNode)
			g.componentGraph.RemoveNode(pn.ID())
			delete(g.instanceIDs, pn.ID())
		}
		pipe.processors = nil
	}

	// Create new processor nodes.
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		pipelineCfg := set.PipelineConfigs[pipelineID]
		for _, procID := range pipelineCfg.Processors {
			procNode := g.createProcessor(pipelineID, procID)
			pipe.processors = append(pipe.processors, procNode)
		}
	}

	// Create new receiver nodes.
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		pipe := g.pipelines[pipelineID]
		for _, recvID := range pipelineCfg.Receivers {
			if set.ConnectorBuilder.IsConfigured(recvID) {
				continue
			}
			nodeID := newReceiverNode(pipelineID.Signal(), recvID).ID()
			if !state.receiversToAdd[nodeID] && state.receiversToRebuild[nodeID] == nil {
				continue
			}
			rcvrNode := g.createReceiver(pipelineID, recvID)
			pipe.receivers[rcvrNode.ID()] = rcvrNode
		}
	}

	// Wire edges for affected pipelines.
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		var from graph.Node = pipe.capabilitiesNode
		for _, proc := range pipe.processors {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, proc))
			from = proc
		}
		g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, pipe.fanOutNode))
	}

	// Wire edges for new/rebuilt receivers.
	for _, pipe := range g.pipelines {
		for _, node := range pipe.receivers {
			rn, ok := node.(*receiverNode)
			if !ok {
				continue
			}
			nodeID := rn.ID()
			if !state.receiversToAdd[nodeID] && state.receiversToRebuild[nodeID] == nil {
				continue
			}
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(node, pipe.capabilitiesNode))
		}
	}
}

// reloadBuild builds all components that need to be built.
func (g *Graph) reloadBuild(ctx context.Context, set Settings, state *reloadState,
	newReceiverCfgs map[component.ID]component.Config,
	receiverFactories map[component.Type]receiver.Factory,
	newProcessorCfgs map[component.ID]component.Config,
	processorFactories map[component.Type]processor.Factory,
	newExporterCfgs map[component.ID]component.Config,
	exporterFactories map[component.Type]exporter.Factory,
	newConnectorCfgs map[component.ID]component.Config,
	connectorFactories map[component.Type]connector.Factory,
) error {
	// Build exporters.
	if len(state.exporterNodesToRebuild) > 0 {
		exporterBuilder := builders.NewExporter(newExporterCfgs, exporterFactories)
		for _, en := range state.exporterNodesToRebuild {
			if err := en.buildComponent(ctx, set.Telemetry, set.BuildInfo, exporterBuilder); err != nil {
				return fmt.Errorf("failed to build exporter %q: %w", en.componentID, err)
			}
		}
	}

	// Build connectors.
	if len(state.connectorNodesToRebuild) > 0 {
		connectorBuilder := builders.NewConnector(newConnectorCfgs, connectorFactories)
		for nodeID, cn := range state.connectorNodesToRebuild {
			if err := cn.buildComponent(ctx, set.Telemetry, set.BuildInfo, connectorBuilder, g.nextConsumers(nodeID)); err != nil {
				return fmt.Errorf("failed to build connector %q: %w", cn.componentID, err)
			}
		}
	}

	// Rebuild fanOutNode for pipelines with changed exporters/connectors.
	for pipelineID := range state.pipelinesNeedingFanOutRebuild {
		pipe := g.pipelines[pipelineID]
		g.rebuildFanOutNode(pipe.fanOutNode)
	}

	// Build processors (reverse order).
	if len(state.affectedPipelines) > 0 {
		processorBuilder := builders.NewProcessor(newProcessorCfgs, processorFactories)
		for pipelineID := range state.affectedPipelines {
			pipe := g.pipelines[pipelineID]
			for i := len(pipe.processors) - 1; i >= 0; i-- {
				pn := pipe.processors[i].(*processorNode)
				next := g.nextConsumers(pn.ID())[0]
				if err := pn.buildComponent(ctx, set.Telemetry, set.BuildInfo, processorBuilder, next); err != nil {
					return fmt.Errorf("failed to build processor %q: %w", pn.componentID, err)
				}
			}
		}
	}

	// Rebuild capabilitiesNode for affected pipelines.
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		g.rebuildCapabilitiesNode(pipe)
	}

	// Build receivers.
	if len(state.receiversToAdd) > 0 || len(state.receiversToRebuild) > 0 {
		receiverBuilder := builders.NewReceiver(newReceiverCfgs, receiverFactories)
		for _, pipe := range g.pipelines {
			for _, node := range pipe.receivers {
				rn, ok := node.(*receiverNode)
				if !ok {
					continue
				}
				nodeID := rn.ID()
				if !state.receiversToAdd[nodeID] && state.receiversToRebuild[nodeID] == nil {
					continue
				}
				if state.builtReceivers[nodeID] {
					continue
				}
				state.builtReceivers[nodeID] = true
				if err := rn.buildComponent(ctx, set.Telemetry, set.BuildInfo, receiverBuilder, g.nextConsumers(nodeID)); err != nil {
					return fmt.Errorf("failed to build receiver %q: %w", rn.componentID, err)
				}
			}
		}
	}

	return nil
}

// rebuildFanOutNode rebuilds a fanOutNode with its current consumers.
func (g *Graph) rebuildFanOutNode(n *fanOutNode) {
	nexts := g.nextConsumers(n.ID())
	switch n.pipelineID.Signal() {
	case pipeline.SignalTraces:
		consumers := make([]consumer.Traces, 0, len(nexts))
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Traces))
		}
		n.baseConsumer = fanoutconsumer.NewTraces(consumers)
	case pipeline.SignalMetrics:
		consumers := make([]consumer.Metrics, 0, len(nexts))
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Metrics))
		}
		n.baseConsumer = fanoutconsumer.NewMetrics(consumers)
	case pipeline.SignalLogs:
		consumers := make([]consumer.Logs, 0, len(nexts))
		for _, next := range nexts {
			consumers = append(consumers, next.(consumer.Logs))
		}
		n.baseConsumer = fanoutconsumer.NewLogs(consumers)
	case xpipeline.SignalProfiles:
		consumers := make([]xconsumer.Profiles, 0, len(nexts))
		for _, next := range nexts {
			consumers = append(consumers, next.(xconsumer.Profiles))
		}
		n.baseConsumer = fanoutconsumer.NewProfiles(consumers)
	}
}

// rebuildCapabilitiesNode rebuilds a pipeline's capabilitiesNode.
func (g *Graph) rebuildCapabilitiesNode(pipe *pipelineNodes) {
	capability := consumer.Capabilities{
		MutatesData: pipe.fanOutNode.getConsumer().Capabilities().MutatesData,
	}
	for _, proc := range pipe.processors {
		capability.MutatesData = capability.MutatesData || proc.(*processorNode).getConsumer().Capabilities().MutatesData
	}
	next := g.nextConsumers(pipe.capabilitiesNode.ID())[0]
	switch pipe.capabilitiesNode.pipelineID.Signal() {
	case pipeline.SignalTraces:
		cc := capabilityconsumer.NewTraces(next.(consumer.Traces), capability)
		pipe.capabilitiesNode.baseConsumer = cc
		pipe.ConsumeTracesFunc = cc.ConsumeTraces
	case pipeline.SignalMetrics:
		cc := capabilityconsumer.NewMetrics(next.(consumer.Metrics), capability)
		pipe.capabilitiesNode.baseConsumer = cc
		pipe.ConsumeMetricsFunc = cc.ConsumeMetrics
	case pipeline.SignalLogs:
		cc := capabilityconsumer.NewLogs(next.(consumer.Logs), capability)
		pipe.capabilitiesNode.baseConsumer = cc
		pipe.ConsumeLogsFunc = cc.ConsumeLogs
	case xpipeline.SignalProfiles:
		cc := capabilityconsumer.NewProfiles(next.(xconsumer.Profiles), capability)
		pipe.capabilitiesNode.baseConsumer = cc
		pipe.ConsumeProfilesFunc = cc.ConsumeProfiles
	}
}

// reloadStart starts all components that were built or need restarting.
func (g *Graph) reloadStart(ctx context.Context, state *reloadState, host *Host) error {
	// Start exporters.
	for nodeID, en := range state.exporterNodesToRebuild {
		instanceID := g.instanceIDs[nodeID]
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStarting))
		if compErr := en.Start(ctx, &HostWrapper{Host: host, InstanceID: instanceID}); compErr != nil {
			host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(compErr))
			g.telemetry.Logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).
				Error("Failed to start exporter during partial reload",
					zap.Error(compErr), zap.String("id", instanceID.ComponentID().String()))
			return fmt.Errorf("failed to start exporter %q: %w", en.componentID, compErr)
		}
		host.Reporter.ReportOKIfStarting(instanceID)
	}

	// Start connectors.
	for nodeID, cn := range state.connectorNodesToRebuild {
		instanceID := g.instanceIDs[nodeID]
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStarting))
		if compErr := cn.Start(ctx, &HostWrapper{Host: host, InstanceID: instanceID}); compErr != nil {
			host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(compErr))
			g.telemetry.Logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).
				Error("Failed to start connector during partial reload",
					zap.Error(compErr), zap.String("id", instanceID.ComponentID().String()))
			return fmt.Errorf("failed to start connector %q: %w", cn.componentID, compErr)
		}
		host.Reporter.ReportOKIfStarting(instanceID)
	}

	// Start processors (reverse order).
	for pipelineID := range state.affectedPipelines {
		pipe := g.pipelines[pipelineID]
		for i := len(pipe.processors) - 1; i >= 0; i-- {
			pn := pipe.processors[i].(*processorNode)
			instanceID := g.instanceIDs[pn.ID()]
			host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStarting))
			if compErr := pn.Start(ctx, &HostWrapper{Host: host, InstanceID: instanceID}); compErr != nil {
				host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(compErr))
				g.telemetry.Logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).
					Error("Failed to start processor during partial reload",
						zap.Error(compErr), zap.String("id", instanceID.ComponentID().String()))
				return fmt.Errorf("failed to start processor %q: %w", pn.componentID, compErr)
			}
			host.Reporter.ReportOKIfStarting(instanceID)
		}
	}

	// Start new/rebuilt receivers.
	for nodeID := range state.builtReceivers {
		rn := g.componentGraph.Node(nodeID).(*receiverNode)
		instanceID := g.instanceIDs[nodeID]
		host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStarting))
		if compErr := rn.Start(ctx, &HostWrapper{Host: host, InstanceID: instanceID}); compErr != nil {
			host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(compErr))
			g.telemetry.Logger.WithOptions(zap.AddStacktrace(zap.DPanicLevel)).
				Error("Failed to start receiver during partial reload",
					zap.Error(compErr), zap.String("id", instanceID.ComponentID().String()))
			return fmt.Errorf("failed to start receiver %q: %w", rn.componentID, compErr)
		}
		host.Reporter.ReportOKIfStarting(instanceID)
	}

	return nil
}

func (g *Graph) GetExporters() map[pipeline.Signal]map[component.ID]component.Component {
	exportersMap := make(map[pipeline.Signal]map[component.ID]component.Component)
	exportersMap[pipeline.SignalTraces] = make(map[component.ID]component.Component)
	exportersMap[pipeline.SignalMetrics] = make(map[component.ID]component.Component)
	exportersMap[pipeline.SignalLogs] = make(map[component.ID]component.Component)
	exportersMap[xpipeline.SignalProfiles] = make(map[component.ID]component.Component)

	for _, pg := range g.pipelines {
		for _, expNode := range pg.exporters {
			// Skip connectors, otherwise individual components can introduce cycles
			if expNode, ok := g.componentGraph.Node(expNode.ID()).(*exporterNode); ok {
				exportersMap[expNode.pipelineType][expNode.componentID] = expNode.Component
			}
		}
	}
	return exportersMap
}

func cycleErr(err error, cycles [][]graph.Node) error {
	var topoErr topo.Unorderable
	if !errors.As(err, &topoErr) || len(cycles) == 0 || len(cycles[0]) == 0 {
		return err
	}

	// There may be multiple cycles, but report only the first one.
	cycle := cycles[0]

	// The last node is a duplicate of the first node.
	// Remove it because we may start from a different node.
	cycle = cycle[:len(cycle)-1]

	// A cycle always contains a connector. For the sake of consistent
	// error messages report the cycle starting from a connector.
	for i := 0; i < len(cycle); i++ {
		if _, ok := cycle[i].(*connectorNode); ok {
			cycle = append(cycle[i:], cycle[:i]...)
			break
		}
	}

	// Repeat the first node at the end to clarify the cycle
	cycle = append(cycle, cycle[0])

	// Build the error message
	componentDetails := make([]string, 0, len(cycle))
	for _, node := range cycle {
		switch n := node.(type) {
		case *processorNode:
			componentDetails = append(componentDetails, fmt.Sprintf("processor %q in pipeline %q", n.componentID, n.pipelineID.String()))
		case *connectorNode:
			componentDetails = append(componentDetails, fmt.Sprintf("connector %q (%s to %s)", n.componentID, n.exprPipelineType, n.rcvrPipelineType))
		default:
			continue // skip capabilities/fanout nodes
		}
	}
	return fmt.Errorf("cycle detected: %s", strings.Join(componentDetails, " -> "))
}

func connectorStability(f connector.Factory, expType, recType pipeline.Signal) component.StabilityLevel {
	switch expType {
	case pipeline.SignalTraces:
		switch recType {
		case pipeline.SignalTraces:
			return f.TracesToTracesStability()
		case pipeline.SignalMetrics:
			return f.TracesToMetricsStability()
		case pipeline.SignalLogs:
			return f.TracesToLogsStability()
		case xpipeline.SignalProfiles:
			fprof, ok := f.(xconnector.Factory)
			if !ok {
				return component.StabilityLevelUndefined
			}
			return fprof.TracesToProfilesStability()
		}
	case pipeline.SignalMetrics:
		switch recType {
		case pipeline.SignalTraces:
			return f.MetricsToTracesStability()
		case pipeline.SignalMetrics:
			return f.MetricsToMetricsStability()
		case pipeline.SignalLogs:
			return f.MetricsToLogsStability()
		case xpipeline.SignalProfiles:
			fprof, ok := f.(xconnector.Factory)
			if !ok {
				return component.StabilityLevelUndefined
			}
			return fprof.MetricsToProfilesStability()
		}
	case pipeline.SignalLogs:
		switch recType {
		case pipeline.SignalTraces:
			return f.LogsToTracesStability()
		case pipeline.SignalMetrics:
			return f.LogsToMetricsStability()
		case pipeline.SignalLogs:
			return f.LogsToLogsStability()
		case xpipeline.SignalProfiles:
			fprof, ok := f.(xconnector.Factory)
			if !ok {
				return component.StabilityLevelUndefined
			}
			return fprof.LogsToProfilesStability()
		}
	case xpipeline.SignalProfiles:
		fprof, ok := f.(xconnector.Factory)
		if !ok {
			return component.StabilityLevelUndefined
		}
		switch recType {
		case pipeline.SignalTraces:
			return fprof.ProfilesToTracesStability()
		case pipeline.SignalMetrics:
			return fprof.ProfilesToMetricsStability()
		case pipeline.SignalLogs:
			return fprof.ProfilesToLogsStability()
		case xpipeline.SignalProfiles:
			return fprof.ProfilesToProfilesStability()
		}
	}
	return component.StabilityLevelUndefined
}

var (
	_ component.Host                   = (*HostWrapper)(nil)
	_ componentstatus.Reporter         = (*HostWrapper)(nil)
	_ hostcapabilities.ExposeExporters = (*HostWrapper)(nil) //nolint:staticcheck // SA1019
)

type HostWrapper struct {
	*Host
	InstanceID *componentstatus.InstanceID
}

func (host *HostWrapper) Report(event *componentstatus.Event) {
	host.Reporter.ReportStatus(host.InstanceID, event)
}
