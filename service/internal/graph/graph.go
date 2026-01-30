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
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
	if err := rn.Shutdown(ctx); err != nil {
		host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
		return fmt.Errorf("failed to shutdown receiver %q: %w", rn.componentID, err)
	}
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	return nil
}

func (g *Graph) shutdownProcessorNode(ctx context.Context, pn *processorNode, host *Host) error {
	instanceID := g.instanceIDs[pn.ID()]
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
	if err := pn.Shutdown(ctx); err != nil {
		host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
		return fmt.Errorf("failed to shutdown processor %q: %w", pn.componentID, err)
	}
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	return nil
}

func (g *Graph) shutdownExporterNode(ctx context.Context, nodeID int64, en *exporterNode, host *Host) error {
	instanceID := g.instanceIDs[nodeID]
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
	if err := en.Shutdown(ctx); err != nil {
		host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
		return fmt.Errorf("failed to shutdown exporter %q: %w", en.componentID, err)
	}
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	return nil
}

func (g *Graph) shutdownConnectorNode(ctx context.Context, nodeID int64, cn *connectorNode, host *Host) error {
	instanceID := g.instanceIDs[nodeID]
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopping))
	if err := cn.Shutdown(ctx); err != nil {
		host.Reporter.ReportStatus(instanceID, componentstatus.NewPermanentErrorEvent(err))
		return fmt.Errorf("failed to shutdown connector %q: %w", cn.componentID, err)
	}
	host.Reporter.ReportStatus(instanceID, componentstatus.NewEvent(componentstatus.StatusStopped))
	return nil
}

// reloadState holds all the state needed during a partial reload operation.
type reloadState struct {
	// Pipelines
	pipelinesToAdd                map[pipeline.ID]bool // New pipelines in config.
	pipelinesToRemove             map[pipeline.ID]bool // Pipelines no longer in config.
	pipelinesAffected             map[pipeline.ID]bool // Pipelines affected by processor changes.
	pipelinesNeedingFanOutRebuild map[pipeline.ID]bool // Pipelines needing fanOutNode rebuild due to exporter/connector changes.

	// Receivers
	receiversToRemove  map[int64]*receiverNode // Receivers to shut down and remove.
	receiversToRebuild map[int64]*receiverNode // Receivers to shut down and rebuild.
	receiversToAdd     map[int64]bool          // Receivers to create (new pipelines or new receiver IDs).

	// Exporters
	exportersToAdd     map[int64]bool          // Exporters to create (for new pipelines).
	exportersToRebuild map[int64]*exporterNode // Exporters to shut down and rebuild.

	// Connectors
	connectorsToAdd     map[int64]bool           // Connectors to create (new connectors or new exporter/receiver combinations).
	connectorsToRebuild map[int64]*connectorNode // Connectors to shut down and rebuild.
	connectorsToRemove  map[int64]*connectorNode // Connectors to shut down and remove.

	// Built (for starting)
	builtReceiver  map[int64]bool // Receivers that were built.
	builtExporter  map[int64]bool // Exporters that were built.
	builtConnector map[int64]bool // Connectors that were built.
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
// The reload process has six phases:
//  1. Identify changes: Determine which components changed and which pipelines are affected
//  2. Shutdown: Stop all components that need to be recreated (receivers first, then downstream)
//  3. Update builders: Create new builders for components that will be rebuilt
//  4. Update graph: Remove old nodes, create new nodes, wire edges
//  5. Build: Create new component instances (downstream first, then upstream)
//  6. Start: Start all recreated components (downstream first, then receivers last)
//
// Returns true if changes were detected and a partial reload was performed,
// false if no changes were detected. An error is returned if the reload fails.
func (g *Graph) Reload(ctx context.Context, set *Settings,
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
	state := g.identifyChanges(*set,
		oldReceiverCfgs, newReceiverCfgs,
		oldProcessorCfgs, newProcessorCfgs,
		oldExporterCfgs, newExporterCfgs,
		oldConnectorCfgs, newConnectorCfgs)

	// Check if there's anything to do.
	if len(state.pipelinesToAdd) == 0 && len(state.pipelinesToRemove) == 0 &&
		len(state.pipelinesAffected) == 0 && len(state.receiversToRemove) == 0 &&
		len(state.receiversToRebuild) == 0 && len(state.receiversToAdd) == 0 &&
		len(state.exportersToRebuild) == 0 && len(state.connectorsToRebuild) == 0 &&
		len(state.exportersToAdd) == 0 && len(state.connectorsToAdd) == 0 &&
		len(state.connectorsToRemove) == 0 {
		g.telemetry.Logger.Info("Partial reload: no changes detected")
		return false, nil
	}

	// Phase 2: Shutdown components.
	if err := g.reloadShutdown(ctx, state, host); err != nil {
		return false, err
	}

	// Phase 3: Update builders for components that will be rebuilt.
	// Note: pipelinesToAdd are added to pipelinesAffected in reloadUpdateGraph, but we need
	// to ensure builders are created before that.
	if len(state.exportersToRebuild) > 0 || len(state.exportersToAdd) > 0 || len(state.pipelinesToAdd) > 0 {
		set.ExporterBuilder = builders.NewExporter(newExporterCfgs, exporterFactories)
	}
	if len(state.connectorsToRebuild) > 0 || len(state.connectorsToAdd) > 0 {
		set.ConnectorBuilder = builders.NewConnector(newConnectorCfgs, connectorFactories)
	}
	if len(state.pipelinesAffected) > 0 || len(state.pipelinesToAdd) > 0 {
		set.ProcessorBuilder = builders.NewProcessor(newProcessorCfgs, processorFactories)
	}
	if len(state.receiversToAdd) > 0 || len(state.receiversToRebuild) > 0 || len(state.pipelinesToAdd) > 0 {
		set.ReceiverBuilder = builders.NewReceiver(newReceiverCfgs, receiverFactories)
	}

	// Phase 4: Update graph (remove old nodes, create new nodes, wire edges).
	g.reloadUpdateGraph(*set, state)

	// Phase 5: Build components.
	if err := g.reloadBuild(ctx, *set, state); err != nil {
		return false, err
	}

	// Phase 6: Start components.
	if err := g.reloadStart(ctx, state, host); err != nil {
		return false, err
	}

	g.telemetry.Logger.Info("Partial reload completed successfully",
		zap.Int("pipelines_added", len(state.pipelinesToAdd)),
		zap.Int("pipelines_removed", len(state.pipelinesToRemove)),
		zap.Int("affected_pipelines", len(state.pipelinesAffected)),
		zap.Int("receivers_added", len(state.receiversToAdd)),
		zap.Int("receivers_rebuilt", len(state.receiversToRebuild)),
		zap.Int("receivers_removed", len(state.receiversToRemove)),
		zap.Int("exporters_added", len(state.exportersToAdd)),
		zap.Int("exporters_rebuilt", len(state.exportersToRebuild)),
		zap.Int("connectors_added", len(state.connectorsToAdd)),
		zap.Int("connectors_rebuilt", len(state.connectorsToRebuild)),
		zap.Int("connectors_removed", len(state.connectorsToRemove)))
	return true, nil
}

func newReloadState() *reloadState {
	return &reloadState{
		pipelinesToAdd:                make(map[pipeline.ID]bool),
		pipelinesToRemove:             make(map[pipeline.ID]bool),
		pipelinesAffected:             make(map[pipeline.ID]bool),
		pipelinesNeedingFanOutRebuild: make(map[pipeline.ID]bool),
		receiversToRemove:             make(map[int64]*receiverNode),
		receiversToRebuild:            make(map[int64]*receiverNode),
		receiversToAdd:                make(map[int64]bool),
		exportersToAdd:                make(map[int64]bool),
		exportersToRebuild:            make(map[int64]*exporterNode),
		connectorsToRebuild:           make(map[int64]*connectorNode),
		connectorsToAdd:               make(map[int64]bool),
		connectorsToRemove:            make(map[int64]*connectorNode),
		builtReceiver:                 make(map[int64]bool),
		builtExporter:                 make(map[int64]bool),
		builtConnector:                make(map[int64]bool),
	}
}

// identifyChanges analyzes config differences and returns the reload state.
func (g *Graph) identifyChanges(set Settings,
	oldReceiverCfgs, newReceiverCfgs map[component.ID]component.Config,
	oldProcessorCfgs, newProcessorCfgs map[component.ID]component.Config,
	oldExporterCfgs, newExporterCfgs map[component.ID]component.Config,
	oldConnectorCfgs, newConnectorCfgs map[component.ID]component.Config,
) *reloadState {
	state := newReloadState()

	g.identifyPipelineChanges(set, state)
	g.identifyProcessorChanges(set, state, oldProcessorCfgs, newProcessorCfgs)
	g.identifyReceiverChanges(set, state, oldReceiverCfgs, newReceiverCfgs)
	g.identifyExporterConnectorChanges(set, state, oldExporterCfgs, newExporterCfgs, oldConnectorCfgs, newConnectorCfgs)
	g.markReceiversInAffectedPipelines(state)

	return state
}

// identifyPipelineChanges identifies pipelines to add and remove.
func (g *Graph) identifyPipelineChanges(set Settings, state *reloadState) {
	for pipelineID := range set.PipelineConfigs {
		if _, exists := g.pipelines[pipelineID]; !exists {
			state.pipelinesToAdd[pipelineID] = true
		}
	}
	for pipelineID := range g.pipelines {
		if _, exists := set.PipelineConfigs[pipelineID]; !exists {
			state.pipelinesToRemove[pipelineID] = true
		}
	}
}

// identifyProcessorChanges identifies pipelines with processor changes.
// Note: exporter/connector changes also add to pipelinesAffected.
func (g *Graph) identifyProcessorChanges(set Settings, state *reloadState,
	oldProcessorCfgs, newProcessorCfgs map[component.ID]component.Config,
) {
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		if state.pipelinesToAdd[pipelineID] {
			continue
		}
		oldPipe := g.pipelines[pipelineID]
		oldProcIDs := make([]component.ID, 0, len(oldPipe.processors))
		for _, node := range oldPipe.processors {
			oldProcIDs = append(oldProcIDs, node.(*processorNode).componentID)
		}
		if !slices.Equal(oldProcIDs, pipelineCfg.Processors) {
			state.pipelinesAffected[pipelineID] = true
			continue
		}
		for _, procID := range pipelineCfg.Processors {
			if !reflect.DeepEqual(oldProcessorCfgs[procID], newProcessorCfgs[procID]) {
				state.pipelinesAffected[pipelineID] = true
				break
			}
		}
	}
}

// identifyReceiverChanges categorizes receiver changes into add/remove/rebuild.
func (g *Graph) identifyReceiverChanges(set Settings, state *reloadState,
	oldReceiverCfgs, newReceiverCfgs map[component.ID]component.Config,
) {
	// Collect current receivers from all pipelines (including those being removed).
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

	// Collect desired receivers from all pipelines (including new ones).
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

	// Categorize each receiver.
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
}

// identifyExporterConnectorChanges identifies exporter/connector config and list changes.
func (g *Graph) identifyExporterConnectorChanges(set Settings, state *reloadState,
	oldExporterCfgs, newExporterCfgs map[component.ID]component.Config,
	oldConnectorCfgs, newConnectorCfgs map[component.ID]component.Config,
) {
	// Helper to check if a component is a connector in the NEW config.
	isNewConnector := func(id component.ID) bool {
		_, ok := newConnectorCfgs[id]
		return ok
	}

	// Find exporters with config changes.
	exportersToRebuild := make(map[component.ID]bool)
	for expID := range newExporterCfgs {
		if !reflect.DeepEqual(oldExporterCfgs[expID], newExporterCfgs[expID]) {
			exportersToRebuild[expID] = true
		}
	}

	// Find connectors with config changes.
	connectorsToRebuild := make(map[component.ID]bool)
	for connID := range newConnectorCfgs {
		if !reflect.DeepEqual(oldConnectorCfgs[connID], newConnectorCfgs[connID]) {
			connectorsToRebuild[connID] = true
		}
	}

	// Mark exporter/connector nodes for rebuild based on config changes.
	for pipelineID, pipe := range g.pipelines {
		if state.pipelinesToRemove[pipelineID] {
			continue
		}
		for _, node := range pipe.exporters {
			if en, ok := node.(*exporterNode); ok && exportersToRebuild[en.componentID] {
				state.exportersToRebuild[en.ID()] = en
				state.pipelinesNeedingFanOutRebuild[pipelineID] = true
				state.pipelinesAffected[pipelineID] = true
			}
			if cn, ok := node.(*connectorNode); ok && connectorsToRebuild[cn.componentID] {
				state.connectorsToRebuild[cn.ID()] = cn
				state.pipelinesNeedingFanOutRebuild[pipelineID] = true
				state.pipelinesAffected[pipelineID] = true
			}
		}
	}

	// Build maps of current connector usage from existing graph.
	currentConnAsExporter := make(map[component.ID]map[pipeline.ID]bool)
	currentConnAsReceiver := make(map[component.ID]map[pipeline.ID]bool)
	for pipelineID, pipe := range g.pipelines {
		if state.pipelinesToRemove[pipelineID] {
			continue
		}
		for _, node := range pipe.exporters {
			if cn, ok := node.(*connectorNode); ok {
				if currentConnAsExporter[cn.componentID] == nil {
					currentConnAsExporter[cn.componentID] = make(map[pipeline.ID]bool)
				}
				currentConnAsExporter[cn.componentID][pipelineID] = true
			}
		}
		for _, node := range pipe.receivers {
			if cn, ok := node.(*connectorNode); ok {
				if currentConnAsReceiver[cn.componentID] == nil {
					currentConnAsReceiver[cn.componentID] = make(map[pipeline.ID]bool)
				}
				currentConnAsReceiver[cn.componentID][pipelineID] = true
			}
		}
	}

	// Build maps of desired connector usage from new config.
	desiredConnAsExporter := make(map[component.ID]map[pipeline.ID]bool)
	desiredConnAsReceiver := make(map[component.ID]map[pipeline.ID]bool)
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		if state.pipelinesToAdd[pipelineID] {
			continue
		}
		for _, exprID := range pipelineCfg.Exporters {
			if isNewConnector(exprID) {
				if desiredConnAsExporter[exprID] == nil {
					desiredConnAsExporter[exprID] = make(map[pipeline.ID]bool)
				}
				desiredConnAsExporter[exprID][pipelineID] = true
			}
		}
		for _, recvID := range pipelineCfg.Receivers {
			if isNewConnector(recvID) {
				if desiredConnAsReceiver[recvID] == nil {
					desiredConnAsReceiver[recvID] = make(map[pipeline.ID]bool)
				}
				desiredConnAsReceiver[recvID][pipelineID] = true
			}
		}
	}

	// Detect changes in exporter lists (regular exporters).
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		if state.pipelinesToAdd[pipelineID] || state.pipelinesToRemove[pipelineID] {
			continue
		}
		pipe := g.pipelines[pipelineID]

		currentExpIDs := make(map[component.ID]bool)
		for _, node := range pipe.exporters {
			if en, ok := node.(*exporterNode); ok {
				currentExpIDs[en.componentID] = true
			}
		}

		desiredExpIDs := make(map[component.ID]bool)
		for _, exprID := range pipelineCfg.Exporters {
			if !isNewConnector(exprID) {
				desiredExpIDs[exprID] = true
			}
		}

		if !maps.Equal(currentExpIDs, desiredExpIDs) {
			state.pipelinesNeedingFanOutRebuild[pipelineID] = true
			state.pipelinesAffected[pipelineID] = true
			for exprID := range desiredExpIDs {
				if !currentExpIDs[exprID] {
					expNode := newExporterNode(pipelineID.Signal(), exprID)
					state.exportersToAdd[expNode.ID()] = true
				}
			}
		}
	}

	// Detect connector node additions and removals.
	allConnectors := make(map[component.ID]bool)
	for connID := range newConnectorCfgs {
		allConnectors[connID] = true
	}
	for connID := range currentConnAsExporter {
		allConnectors[connID] = true
	}
	for connID := range currentConnAsReceiver {
		allConnectors[connID] = true
	}

	for connID := range allConnectors {
		currentExpPipes := currentConnAsExporter[connID]
		currentRecvPipes := currentConnAsReceiver[connID]
		desiredExpPipes := desiredConnAsExporter[connID]
		desiredRecvPipes := desiredConnAsReceiver[connID]

		// Check each current connector node for removal.
		for expPipeID := range currentExpPipes {
			for recvPipeID := range currentRecvPipes {
				connNode := newConnectorNode(expPipeID.Signal(), recvPipeID.Signal(), connID)
				nodeID := connNode.ID()

				expStillUsed := desiredExpPipes[expPipeID]
				recvStillUsed := desiredRecvPipes[recvPipeID]

				if !expStillUsed || !recvStillUsed {
					if existingNode := g.componentGraph.Node(nodeID); existingNode != nil {
						state.connectorsToRemove[nodeID] = existingNode.(*connectorNode)
						state.pipelinesNeedingFanOutRebuild[expPipeID] = true
						state.pipelinesAffected[expPipeID] = true
					}
				}
			}
		}

		// Check for new connector nodes to add.
		for expPipeID := range desiredExpPipes {
			if state.pipelinesToAdd[expPipeID] {
				continue
			}
			for recvPipeID := range desiredRecvPipes {
				if state.pipelinesToAdd[recvPipeID] {
					continue
				}

				connNode := newConnectorNode(expPipeID.Signal(), recvPipeID.Signal(), connID)
				nodeID := connNode.ID()

				wasExpUsed := currentExpPipes[expPipeID]
				wasRecvUsed := currentRecvPipes[recvPipeID]

				if !wasExpUsed || !wasRecvUsed {
					state.connectorsToAdd[nodeID] = true
					state.pipelinesNeedingFanOutRebuild[expPipeID] = true
					state.pipelinesAffected[expPipeID] = true
				}
			}
		}
	}
}

// markReceiversInAffectedPipelines marks receivers for rebuild in affected pipelines.
// Receivers store consumer references, so they must be rebuilt when downstream changes.
func (g *Graph) markReceiversInAffectedPipelines(state *reloadState) {
	for pipelineID := range state.pipelinesAffected {
		if state.pipelinesToRemove[pipelineID] {
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
}

// reloadShutdown stops all components that need to be stopped.
func (g *Graph) reloadShutdown(ctx context.Context, state *reloadState, host *Host) error {
	// Shutdown all receivers first (ensures no data in flight).
	// This includes receivers from pipelines being removed, which are captured
	// in receiversToRemove (if only used by removed pipelines) or receiversToRebuild
	// (if shared with remaining pipelines).
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
	for pipelineID := range state.pipelinesAffected {
		pipe := g.pipelines[pipelineID]
		for _, node := range pipe.processors {
			if err := g.shutdownProcessorNode(ctx, node.(*processorNode), host); err != nil {
				return err
			}
		}
	}

	// Shutdown processors in pipelines being removed.
	for pipelineID := range state.pipelinesToRemove {
		pipe := g.pipelines[pipelineID]
		for _, node := range pipe.processors {
			if err := g.shutdownProcessorNode(ctx, node.(*processorNode), host); err != nil {
				return err
			}
		}
	}

	// Shutdown exporters being rebuilt.
	shutdownExporters := make(map[int64]bool)
	for nodeID, en := range state.exportersToRebuild {
		shutdownExporters[nodeID] = true
		if err := g.shutdownExporterNode(ctx, nodeID, en, host); err != nil {
			return err
		}
	}

	// Shutdown exporters in pipelines being removed.
	for pipelineID := range state.pipelinesToRemove {
		pipe := g.pipelines[pipelineID]
		for nodeID, node := range pipe.exporters {
			en, ok := node.(*exporterNode)
			if !ok {
				continue
			}
			if shutdownExporters[nodeID] {
				continue
			}
			shutdownExporters[nodeID] = true
			if err := g.shutdownExporterNode(ctx, nodeID, en, host); err != nil {
				return err
			}
		}
	}

	// Shutdown connectors being rebuilt.
	shutdownConnectors := make(map[int64]bool)
	for nodeID, cn := range state.connectorsToRebuild {
		shutdownConnectors[nodeID] = true
		if err := g.shutdownConnectorNode(ctx, nodeID, cn, host); err != nil {
			return err
		}
	}

	// Shutdown connectors being removed.
	for nodeID, cn := range state.connectorsToRemove {
		if shutdownConnectors[nodeID] {
			continue
		}
		shutdownConnectors[nodeID] = true
		if err := g.shutdownConnectorNode(ctx, nodeID, cn, host); err != nil {
			return err
		}
	}

	return nil
}

// reloadUpdateGraph removes old nodes, creates new nodes, and wires edges.
func (g *Graph) reloadUpdateGraph(set Settings, state *reloadState) {
	// Remove all nodes from pipelines being removed.
	for pipelineID := range state.pipelinesToRemove {
		pipe := g.pipelines[pipelineID]

		// Remove receiver nodes (only receiverNodes, not connectorNodes).
		for nodeID, node := range pipe.receivers {
			if rn, ok := node.(*receiverNode); ok {
				// Only remove the node if it's not used by other pipelines.
				// Check if any other pipeline uses this receiver.
				usedElsewhere := false
				for otherPipelineID, otherPipe := range g.pipelines {
					if otherPipelineID == pipelineID || state.pipelinesToRemove[otherPipelineID] {
						continue
					}
					if _, exists := otherPipe.receivers[nodeID]; exists {
						usedElsewhere = true
						break
					}
				}
				if !usedElsewhere {
					g.componentGraph.RemoveNode(nodeID)
					delete(g.instanceIDs, nodeID)
				} else {
					// Update the instanceID to remove this pipeline from the list.
					instanceID := g.instanceIDs[nodeID]
					newPipelines := make([]pipeline.ID, 0)
					instanceID.AllPipelineIDs(func(pid pipeline.ID) bool {
						if pid != pipelineID {
							newPipelines = append(newPipelines, pid)
						}
						return true
					})
					g.instanceIDs[nodeID] = componentstatus.NewInstanceID(rn.componentID, component.KindReceiver, newPipelines...)
				}
			}
		}

		// Remove processor nodes.
		for _, node := range pipe.processors {
			pn := node.(*processorNode)
			g.componentGraph.RemoveNode(pn.ID())
			delete(g.instanceIDs, pn.ID())
		}

		// Remove exporter nodes (only exporterNodes, not connectorNodes).
		for nodeID, node := range pipe.exporters {
			if en, ok := node.(*exporterNode); ok {
				// Only remove the node if it's not used by other pipelines.
				usedElsewhere := false
				for otherPipelineID, otherPipe := range g.pipelines {
					if otherPipelineID == pipelineID || state.pipelinesToRemove[otherPipelineID] {
						continue
					}
					if _, exists := otherPipe.exporters[nodeID]; exists {
						usedElsewhere = true
						break
					}
				}
				if !usedElsewhere {
					g.componentGraph.RemoveNode(nodeID)
					delete(g.instanceIDs, nodeID)
				} else {
					// Update the instanceID to remove this pipeline from the list.
					instanceID := g.instanceIDs[nodeID]
					newPipelines := make([]pipeline.ID, 0)
					instanceID.AllPipelineIDs(func(pid pipeline.ID) bool {
						if pid != pipelineID {
							newPipelines = append(newPipelines, pid)
						}
						return true
					})
					g.instanceIDs[nodeID] = componentstatus.NewInstanceID(en.componentID, component.KindExporter, newPipelines...)
				}
			}
		}

		// Remove capabilities and fanout nodes.
		g.componentGraph.RemoveNode(pipe.capabilitiesNode.ID())
		g.componentGraph.RemoveNode(pipe.fanOutNode.ID())

		// Remove the pipeline entry.
		delete(g.pipelines, pipelineID)
	}

	// Remove receivers being removed or rebuilt.
	for nodeID := range state.receiversToRemove {
		g.componentGraph.RemoveNode(nodeID)
		delete(g.instanceIDs, nodeID)
	}
	for nodeID := range state.receiversToRebuild {
		g.componentGraph.RemoveNode(nodeID)
		delete(g.instanceIDs, nodeID)
	}
	for pipelineID, pipe := range g.pipelines {
		if state.pipelinesToRemove[pipelineID] {
			continue // Already removed.
		}
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
	for pipelineID := range state.pipelinesAffected {
		pipe := g.pipelines[pipelineID]
		for _, node := range pipe.processors {
			pn := node.(*processorNode)
			g.componentGraph.RemoveNode(pn.ID())
			delete(g.instanceIDs, pn.ID())
		}
		pipe.processors = nil
	}

	// Remove connector nodes being removed.
	for nodeID, cn := range state.connectorsToRemove {
		g.componentGraph.RemoveNode(nodeID)
		delete(g.instanceIDs, nodeID)
		// Remove from pipeline exporter/receiver maps.
		for _, pipe := range g.pipelines {
			delete(pipe.exporters, nodeID)
			delete(pipe.receivers, nodeID)
		}
		_ = cn // used for shutdown, node removal done above
	}

	// Create new exporter nodes for existing pipelines with changed exporter lists.
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		if state.pipelinesToAdd[pipelineID] || state.pipelinesToRemove[pipelineID] {
			continue
		}
		if !state.pipelinesNeedingFanOutRebuild[pipelineID] {
			continue
		}
		pipe := g.pipelines[pipelineID]
		for _, exprID := range pipelineCfg.Exporters {
			if set.ConnectorBuilder.IsConfigured(exprID) {
				continue // Connectors handled separately.
			}
			expNode := newExporterNode(pipelineID.Signal(), exprID)
			nodeID := expNode.ID()
			if state.exportersToAdd[nodeID] {
				// Create the new exporter node.
				created := g.createExporter(pipelineID, exprID)
				pipe.exporters[created.ID()] = created
			}
		}
	}

	// Create new connector nodes (for existing pipelines).
	for nodeID := range state.connectorsToAdd {
		// Find which connector this node represents by checking configs.
		for pipelineID, pipelineCfg := range set.PipelineConfigs {
			if state.pipelinesToAdd[pipelineID] || state.pipelinesToRemove[pipelineID] {
				continue
			}
			for _, exprID := range pipelineCfg.Exporters {
				if !set.ConnectorBuilder.IsConfigured(exprID) {
					continue
				}
				// Check all receiver pipelines for this connector.
				for recvPipelineID, recvPipelineCfg := range set.PipelineConfigs {
					if state.pipelinesToAdd[recvPipelineID] || state.pipelinesToRemove[recvPipelineID] {
						continue
					}
					for _, recvID := range recvPipelineCfg.Receivers {
						if recvID != exprID {
							continue
						}
						connNode := newConnectorNode(pipelineID.Signal(), recvPipelineID.Signal(), exprID)
						if connNode.ID() == nodeID {
							// Create the connector node.
							created := g.createConnector(pipelineID, recvPipelineID, exprID)
							g.pipelines[pipelineID].exporters[created.ID()] = created
							g.pipelines[recvPipelineID].receivers[created.ID()] = created
						}
					}
				}
			}
		}
	}

	// Create new pipelineNodes entries for added pipelines.
	for pipelineID := range state.pipelinesToAdd {
		g.pipelines[pipelineID] = &pipelineNodes{
			receivers: make(map[int64]graph.Node),
			exporters: make(map[int64]graph.Node),
		}
	}

	// Create nodes for added pipelines (receivers are handled separately below).
	for pipelineID := range state.pipelinesToAdd {
		pipe := g.pipelines[pipelineID]
		pipelineCfg := set.PipelineConfigs[pipelineID]

		// Create capabilities node.
		pipe.capabilitiesNode = newCapabilitiesNode(pipelineID)
		g.componentGraph.AddNode(pipe.capabilitiesNode)

		// Create processor nodes.
		for _, procID := range pipelineCfg.Processors {
			procNode := g.createProcessor(pipelineID, procID)
			pipe.processors = append(pipe.processors, procNode)
		}

		// Create fanout node.
		pipe.fanOutNode = newFanOutNode(pipelineID)
		g.componentGraph.AddNode(pipe.fanOutNode)

		// Create exporter nodes (skip connectors, they're handled separately).
		for _, exprID := range pipelineCfg.Exporters {
			if set.ConnectorBuilder.IsConfigured(exprID) {
				continue // Connectors are handled in the connector creation section.
			}
			expNode := g.createExporter(pipelineID, exprID)
			pipe.exporters[expNode.ID()] = expNode
			state.exportersToAdd[expNode.ID()] = true
		}

		// Mark this pipeline as affected so it gets wired and built.
		state.pipelinesAffected[pipelineID] = true
	}

	// Create connector nodes for added pipelines.
	// This handles connectors that connect new pipelines with existing or other new pipelines.
	for pipelineID := range state.pipelinesToAdd {
		pipe := g.pipelines[pipelineID]
		pipelineCfg := set.PipelineConfigs[pipelineID]

		// Handle connectors as exporters (this pipeline sends to connector).
		for _, exprID := range pipelineCfg.Exporters {
			if !set.ConnectorBuilder.IsConfigured(exprID) {
				continue
			}
			// Find receiver pipelines for this connector.
			for recvPipelineID, recvPipelineCfg := range set.PipelineConfigs {
				if state.pipelinesToRemove[recvPipelineID] {
					continue
				}
				for _, recvID := range recvPipelineCfg.Receivers {
					if recvID != exprID {
						continue
					}
					// Create connector node.
					connNode := g.createConnector(pipelineID, recvPipelineID, exprID)
					pipe.exporters[connNode.ID()] = connNode
					g.pipelines[recvPipelineID].receivers[connNode.ID()] = connNode
					state.connectorsToAdd[connNode.ID()] = true
				}
			}
		}

		// Handle connectors as receivers (this pipeline receives from connector).
		for _, recvID := range pipelineCfg.Receivers {
			if !set.ConnectorBuilder.IsConfigured(recvID) {
				continue
			}
			// Find exporter pipelines for this connector.
			for expPipelineID, expPipelineCfg := range set.PipelineConfigs {
				if state.pipelinesToRemove[expPipelineID] {
					continue
				}
				if state.pipelinesToAdd[expPipelineID] {
					continue // Already handled in the exporter loop above for this new pipeline.
				}
				for _, exprID := range expPipelineCfg.Exporters {
					if exprID != recvID {
						continue
					}
					// Create connector node.
					connNode := g.createConnector(expPipelineID, pipelineID, recvID)
					g.pipelines[expPipelineID].exporters[connNode.ID()] = connNode
					pipe.receivers[connNode.ID()] = connNode
					state.connectorsToAdd[connNode.ID()] = true
					// Mark exporter-side pipeline as needing fanOut rebuild.
					state.pipelinesNeedingFanOutRebuild[expPipelineID] = true
					state.pipelinesAffected[expPipelineID] = true
				}
			}
		}
	}

	// Create new processor nodes for existing affected pipelines.
	for pipelineID := range state.pipelinesAffected {
		if state.pipelinesToAdd[pipelineID] {
			continue // Already created above.
		}
		pipe := g.pipelines[pipelineID]
		pipelineCfg := set.PipelineConfigs[pipelineID]
		for _, procID := range pipelineCfg.Processors {
			procNode := g.createProcessor(pipelineID, procID)
			pipe.processors = append(pipe.processors, procNode)
		}
	}

	// Create new receiver nodes for all pipelines (including new pipelines).
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		if state.pipelinesToRemove[pipelineID] {
			continue
		}
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

	// Wire edges for affected pipelines (including new pipelines).
	for pipelineID := range state.pipelinesAffected {
		pipe := g.pipelines[pipelineID]
		var from graph.Node = pipe.capabilitiesNode
		for _, proc := range pipe.processors {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, proc))
			from = proc
		}
		g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, pipe.fanOutNode))

		// For new pipelines, wire edges to all exporters.
		// For existing pipelines with fanOut rebuild, wire edges to new exporters/connectors.
		if state.pipelinesToAdd[pipelineID] {
			for _, exporter := range pipe.exporters {
				g.componentGraph.SetEdge(g.componentGraph.NewEdge(pipe.fanOutNode, exporter))
			}
		} else if state.pipelinesNeedingFanOutRebuild[pipelineID] {
			for nodeID, exporter := range pipe.exporters {
				if state.exportersToAdd[nodeID] || state.connectorsToAdd[nodeID] {
					g.componentGraph.SetEdge(g.componentGraph.NewEdge(pipe.fanOutNode, exporter))
				}
			}
		}
	}

	// Wire edges for new connectors to their receiver-side capabilitiesNodes.
	for nodeID := range state.connectorsToAdd {
		node := g.componentGraph.Node(nodeID)
		if node == nil {
			continue
		}
		// Find all receiver pipelines that have this connector.
		for pipelineID, pipe := range g.pipelines {
			if state.pipelinesToRemove[pipelineID] {
				continue
			}
			if _, exists := pipe.receivers[nodeID]; exists {
				g.componentGraph.SetEdge(g.componentGraph.NewEdge(node, pipe.capabilitiesNode))
			}
		}
	}

	// Wire edges for new/rebuilt receivers.
	for pipelineID, pipe := range g.pipelines {
		if state.pipelinesToAdd[pipelineID] {
			// For new pipelines, wire all receivers.
			for _, node := range pipe.receivers {
				g.componentGraph.SetEdge(g.componentGraph.NewEdge(node, pipe.capabilitiesNode))
			}
		} else {
			// For existing pipelines, only wire new/rebuilt receivers.
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
}

// reloadBuild builds all components that need to be built.
// It uses the builders from set which were updated before this phase.
func (g *Graph) reloadBuild(ctx context.Context, set Settings, state *reloadState) error {
	// Build exporters being rebuilt.
	for nodeID, en := range state.exportersToRebuild {
		state.builtExporter[nodeID] = true
		if err := en.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ExporterBuilder); err != nil {
			return fmt.Errorf("failed to build exporter %q: %w", en.componentID, err)
		}
	}

	// Build new exporters (added to existing pipelines or new pipelines).
	for nodeID := range state.exportersToAdd {
		if state.builtExporter[nodeID] {
			continue
		}
		state.builtExporter[nodeID] = true
		node := g.componentGraph.Node(nodeID)
		if node == nil {
			continue
		}
		en := node.(*exporterNode)
		if err := en.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ExporterBuilder); err != nil {
			return fmt.Errorf("failed to build exporter %q: %w", en.componentID, err)
		}
	}

	// Build exporters for new pipelines (that weren't already built above).
	for pipelineID := range state.pipelinesToAdd {
		pipe := g.pipelines[pipelineID]
		for nodeID, node := range pipe.exporters {
			en, ok := node.(*exporterNode)
			if !ok {
				continue
			}
			if state.builtExporter[nodeID] {
				continue
			}
			state.builtExporter[nodeID] = true
			if err := en.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ExporterBuilder); err != nil {
				return fmt.Errorf("failed to build exporter %q: %w", en.componentID, err)
			}
		}
	}

	// Build connectors being rebuilt.
	for nodeID, cn := range state.connectorsToRebuild {
		state.builtConnector[nodeID] = true
		if err := cn.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ConnectorBuilder, g.nextConsumers(nodeID)); err != nil {
			return fmt.Errorf("failed to build connector %q: %w", cn.componentID, err)
		}
	}

	// Build new connectors.
	for nodeID := range state.connectorsToAdd {
		if state.builtConnector[nodeID] {
			continue
		}
		state.builtConnector[nodeID] = true
		node := g.componentGraph.Node(nodeID)
		if node == nil {
			continue
		}
		cn := node.(*connectorNode)
		if err := cn.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ConnectorBuilder, g.nextConsumers(nodeID)); err != nil {
			return fmt.Errorf("failed to build connector %q: %w", cn.componentID, err)
		}
	}

	// Rebuild fanOutNode for pipelines with changed exporters/connectors.
	for pipelineID := range state.pipelinesNeedingFanOutRebuild {
		pipe := g.pipelines[pipelineID]
		g.rebuildFanOutNode(pipe.fanOutNode)
	}

	// Build fanOutNode for new pipelines.
	for pipelineID := range state.pipelinesToAdd {
		pipe := g.pipelines[pipelineID]
		g.rebuildFanOutNode(pipe.fanOutNode)
	}

	// Build processors (reverse order).
	for pipelineID := range state.pipelinesAffected {
		pipe := g.pipelines[pipelineID]
		for i := len(pipe.processors) - 1; i >= 0; i-- {
			pn := pipe.processors[i].(*processorNode)
			next := g.nextConsumers(pn.ID())[0]
			if err := pn.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ProcessorBuilder, next); err != nil {
				return fmt.Errorf("failed to build processor %q: %w", pn.componentID, err)
			}
		}
	}

	// Rebuild capabilitiesNode for affected pipelines (including new pipelines).
	for pipelineID := range state.pipelinesAffected {
		pipe := g.pipelines[pipelineID]
		g.rebuildCapabilitiesNode(pipe)
	}

	// Build receivers.
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
			if state.builtReceiver[nodeID] {
				continue
			}
			state.builtReceiver[nodeID] = true
			if err := rn.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ReceiverBuilder, g.nextConsumers(nodeID)); err != nil {
				return fmt.Errorf("failed to build receiver %q: %w", rn.componentID, err)
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
	// Track which exporters we've already started.
	startedExporters := make(map[int64]bool)

	// Start exporters being rebuilt.
	for nodeID, en := range state.exportersToRebuild {
		startedExporters[nodeID] = true
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

	// Start exporters for new pipelines.
	for nodeID := range state.builtExporter {
		if startedExporters[nodeID] {
			continue
		}
		startedExporters[nodeID] = true
		en := g.componentGraph.Node(nodeID).(*exporterNode)
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

	// Start connectors (both rebuilt and newly added).
	for nodeID := range state.builtConnector {
		node := g.componentGraph.Node(nodeID)
		if node == nil {
			continue
		}
		cn := node.(*connectorNode)
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
	for pipelineID := range state.pipelinesAffected {
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
	for nodeID := range state.builtReceiver {
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
