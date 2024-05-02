// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/internal/graph"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/servicetelemetry"
	"go.opentelemetry.io/collector/service/pipelines"
)

// Settings holds configuration for building builtPipelines.
type Settings struct {
	Telemetry servicetelemetry.TelemetrySettings
	BuildInfo component.BuildInfo

	ReceiverBuilder  *receiver.Builder
	ProcessorBuilder *processor.Builder
	ExporterBuilder  *exporter.Builder
	ConnectorBuilder *connector.Builder

	// PipelineConfigs is a map of component.ID to PipelineConfig.
	PipelineConfigs pipelines.Config
}

type Graph struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelines map[component.ID]*pipelineNodes

	// Keep track of status source per node
	instanceIDs map[int64]*component.InstanceID

	telemetry servicetelemetry.TelemetrySettings
}

func Build(ctx context.Context, set Settings) (*Graph, error) {
	pipelines := &Graph{
		componentGraph: simple.NewDirectedGraph(),
		pipelines:      make(map[component.ID]*pipelineNodes, len(set.PipelineConfigs)),
		instanceIDs:    make(map[int64]*component.InstanceID),
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
	return pipelines, pipelines.buildComponents(ctx, set)
}

// Creates a node for each instance of a component and adds it to the graph
func (g *Graph) createNodes(set Settings) error {
	// Build a list of all connectors for easy reference
	connectors := make(map[component.ID]struct{})

	// Keep track of connectors and where they are used. (map[connectorID][]pipelineID)
	connectorsAsExporter := make(map[component.ID][]component.ID)
	connectorsAsReceiver := make(map[component.ID][]component.ID)

	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		pipe := g.pipelines[pipelineID]
		for _, recvID := range pipelineCfg.Receivers {
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

		expTypes := make(map[component.DataType]bool)
		for _, pipelineID := range connectorsAsExporter[connID] {
			// The presence of each key indicates how the connector is used as an exporter.
			// The value is initially set to false. Later we will set the value to true *if* we
			// confirm that there is a supported corresponding use as a receiver.
			expTypes[pipelineID.Type()] = false
		}
		recTypes := make(map[component.DataType]bool)
		for _, pipelineID := range connectorsAsReceiver[connID] {
			// The presence of each key indicates how the connector is used as a receiver.
			// The value is initially set to false. Later we will set the value to true *if* we
			// confirm that there is a supported corresponding use as an exporter.
			recTypes[pipelineID.Type()] = false
		}

		for expType := range expTypes {
			for recType := range recTypes {
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
			return fmt.Errorf("connector %q used as exporter in %s pipeline but not used in any supported receiver pipeline", connID, expType)
		}
		for recType, supportedUse := range recTypes {
			if supportedUse {
				continue
			}
			return fmt.Errorf("connector %q used as receiver in %s pipeline but not used in any supported exporter pipeline", connID, recType)
		}

		for _, eID := range connectorsAsExporter[connID] {
			for _, rID := range connectorsAsReceiver[connID] {
				if connectorStability(connFactory, eID.Type(), rID.Type()) == component.StabilityLevelUndefined {
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

func (g *Graph) createReceiver(pipelineID, recvID component.ID) *receiverNode {
	rcvrNode := newReceiverNode(pipelineID.Type(), recvID)
	if node := g.componentGraph.Node(rcvrNode.ID()); node != nil {
		g.instanceIDs[node.ID()].PipelineIDs[pipelineID] = struct{}{}
		return node.(*receiverNode)
	}
	g.componentGraph.AddNode(rcvrNode)
	g.instanceIDs[rcvrNode.ID()] = &component.InstanceID{
		ID:   recvID,
		Kind: component.KindReceiver,
		PipelineIDs: map[component.ID]struct{}{
			pipelineID: {},
		},
	}
	return rcvrNode
}

func (g *Graph) createProcessor(pipelineID, procID component.ID) *processorNode {
	procNode := newProcessorNode(pipelineID, procID)
	g.componentGraph.AddNode(procNode)
	g.instanceIDs[procNode.ID()] = &component.InstanceID{
		ID:   procID,
		Kind: component.KindProcessor,
		PipelineIDs: map[component.ID]struct{}{
			pipelineID: {},
		},
	}
	return procNode
}

func (g *Graph) createExporter(pipelineID, exprID component.ID) *exporterNode {
	expNode := newExporterNode(pipelineID.Type(), exprID)
	if node := g.componentGraph.Node(expNode.ID()); node != nil {
		g.instanceIDs[expNode.ID()].PipelineIDs[pipelineID] = struct{}{}
		return node.(*exporterNode)
	}
	g.componentGraph.AddNode(expNode)
	g.instanceIDs[expNode.ID()] = &component.InstanceID{
		ID:   expNode.componentID,
		Kind: component.KindExporter,
		PipelineIDs: map[component.ID]struct{}{
			pipelineID: {},
		},
	}
	return expNode
}

func (g *Graph) createConnector(exprPipelineID, rcvrPipelineID, connID component.ID) *connectorNode {
	connNode := newConnectorNode(exprPipelineID.Type(), rcvrPipelineID.Type(), connID)
	if node := g.componentGraph.Node(connNode.ID()); node != nil {
		instanceID := g.instanceIDs[connNode.ID()]
		instanceID.PipelineIDs[exprPipelineID] = struct{}{}
		instanceID.PipelineIDs[rcvrPipelineID] = struct{}{}
		return node.(*connectorNode)
	}
	g.componentGraph.AddNode(connNode)
	g.instanceIDs[connNode.ID()] = &component.InstanceID{
		ID:   connNode.componentID,
		Kind: component.KindConnector,
		PipelineIDs: map[component.ID]struct{}{
			exprPipelineID: {},
			rcvrPipelineID: {},
		},
	}
	return connNode
}

func (g *Graph) createEdges() {
	for _, pg := range g.pipelines {
		for _, receiver := range pg.receivers {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(receiver, pg.capabilitiesNode))
		}

		var from, to graph.Node
		from = pg.capabilitiesNode
		for _, processor := range pg.processors {
			to = processor
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, to))
			from = processor
		}
		to = pg.fanOutNode
		g.componentGraph.SetEdge(g.componentGraph.NewEdge(from, to))

		for _, exporter := range pg.exporters {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(pg.fanOutNode, exporter))
		}
	}
}

func (g *Graph) buildComponents(ctx context.Context, set Settings) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return cycleErr(err, topo.DirectedCyclesIn(g.componentGraph))
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]

		// skipped for capabilitiesNodes and fanoutNodes as they are not assigned componentIDs.
		var telemetrySettings component.TelemetrySettings
		if instanceID, ok := g.instanceIDs[node.ID()]; ok {
			telemetrySettings = set.Telemetry.ToComponentTelemetrySettings(instanceID)
		}

		switch n := node.(type) {
		case *receiverNode:
			err = n.buildComponent(ctx, telemetrySettings, set.BuildInfo, set.ReceiverBuilder, g.nextConsumers(n.ID()))
		case *processorNode:
			err = n.buildComponent(ctx, telemetrySettings, set.BuildInfo, set.ProcessorBuilder, g.nextConsumers(n.ID())[0])
		case *exporterNode:
			err = n.buildComponent(ctx, telemetrySettings, set.BuildInfo, set.ExporterBuilder)
		case *connectorNode:
			err = n.buildComponent(ctx, telemetrySettings, set.BuildInfo, set.ConnectorBuilder, g.nextConsumers(n.ID()))
		case *capabilitiesNode:
			capability := consumer.Capabilities{
				// The fanOutNode represents the aggregate capabilities of the exporters in the pipeline.
				MutatesData: g.pipelines[n.pipelineID].fanOutNode.getConsumer().Capabilities().MutatesData,
			}
			for _, proc := range g.pipelines[n.pipelineID].processors {
				capability.MutatesData = capability.MutatesData || proc.getConsumer().Capabilities().MutatesData
			}
			next := g.nextConsumers(n.ID())[0]
			switch n.pipelineID.Type() {
			case component.DataTypeTraces:
				cc := capabilityconsumer.NewTraces(next.(consumer.Traces), capability)
				n.baseConsumer = cc
				n.ConsumeTracesFunc = cc.ConsumeTraces
			case component.DataTypeMetrics:
				cc := capabilityconsumer.NewMetrics(next.(consumer.Metrics), capability)
				n.baseConsumer = cc
				n.ConsumeMetricsFunc = cc.ConsumeMetrics
			case component.DataTypeLogs:
				cc := capabilityconsumer.NewLogs(next.(consumer.Logs), capability)
				n.baseConsumer = cc
				n.ConsumeLogsFunc = cc.ConsumeLogs
			}
		case *fanOutNode:
			nexts := g.nextConsumers(n.ID())
			switch n.pipelineID.Type() {
			case component.DataTypeTraces:
				consumers := make([]consumer.Traces, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(consumer.Traces))
				}
				n.baseConsumer = fanoutconsumer.NewTraces(consumers)
			case component.DataTypeMetrics:
				consumers := make([]consumer.Metrics, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(consumer.Metrics))
				}
				n.baseConsumer = fanoutconsumer.NewMetrics(consumers)
			case component.DataTypeLogs:
				consumers := make([]consumer.Logs, 0, len(nexts))
				for _, next := range nexts {
					consumers = append(consumers, next.(consumer.Logs))
				}
				n.baseConsumer = fanoutconsumer.NewLogs(consumers)
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
	processors []*processorNode

	// Emits to exporters.
	*fanOutNode

	// Use map to assist with deduplication of connector instances.
	exporters map[int64]graph.Node
}

func (g *Graph) StartAll(ctx context.Context, host component.Host) error {
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
		g.telemetry.Status.ReportStatus(
			instanceID,
			component.NewStatusEvent(component.StatusStarting),
		)

		if compErr := comp.Start(ctx, host); compErr != nil {
			g.telemetry.Status.ReportStatus(
				instanceID,
				component.NewPermanentErrorEvent(compErr),
			)
			return compErr
		}

		g.telemetry.Status.ReportOKIfStarting(instanceID)
	}
	return nil
}

func (g *Graph) ShutdownAll(ctx context.Context) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return err
	}

	// Stop in topological order so that upstream components
	// are stopped before downstream components.  This ensures
	// that each component has a chance to drain to its consumer
	// before the consumer is stopped.
	var errs error
	for i := 0; i < len(nodes); i++ {
		node := nodes[i]
		comp, ok := node.(component.Component)

		if !ok {
			// Skip capabilities/fanout nodes
			continue
		}

		instanceID := g.instanceIDs[node.ID()]
		g.telemetry.Status.ReportStatus(
			instanceID,
			component.NewStatusEvent(component.StatusStopping),
		)

		if compErr := comp.Shutdown(ctx); compErr != nil {
			errs = multierr.Append(errs, compErr)
			g.telemetry.Status.ReportStatus(
				instanceID,
				component.NewPermanentErrorEvent(compErr),
			)
			continue
		}

		g.telemetry.Status.ReportStatus(
			instanceID,
			component.NewStatusEvent(component.StatusStopped),
		)
	}
	return errs
}

// Deprecated: [0.79.0] This function will be removed in the future.
// Several components in the contrib repository use this function so it cannot be removed
// before those cases are removed. In most cases, use of this function can be replaced by a
// connector. See https://github.com/open-telemetry/opentelemetry-collector/issues/7370 and
// https://github.com/open-telemetry/opentelemetry-collector/pull/7390#issuecomment-1483710184
// for additional information.
func (g *Graph) GetExporters() map[component.DataType]map[component.ID]component.Component {
	exportersMap := make(map[component.DataType]map[component.ID]component.Component)
	exportersMap[component.DataTypeTraces] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeMetrics] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeLogs] = make(map[component.ID]component.Component)

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
			componentDetails = append(componentDetails, fmt.Sprintf("processor %q in pipeline %q", n.componentID, n.pipelineID))
		case *connectorNode:
			componentDetails = append(componentDetails, fmt.Sprintf("connector %q (%s to %s)", n.componentID, n.exprPipelineType, n.rcvrPipelineType))
		default:
			continue // skip capabilities/fanout nodes
		}
	}
	return fmt.Errorf("cycle detected: %s", strings.Join(componentDetails, " -> "))
}

func connectorStability(f connector.Factory, expType, recType component.Type) component.StabilityLevel {
	switch expType {
	case component.DataTypeTraces:
		switch recType {
		case component.DataTypeTraces:
			return f.TracesToTracesStability()
		case component.DataTypeMetrics:
			return f.TracesToMetricsStability()
		case component.DataTypeLogs:
			return f.TracesToLogsStability()
		}
	case component.DataTypeMetrics:
		switch recType {
		case component.DataTypeTraces:
			return f.MetricsToTracesStability()
		case component.DataTypeMetrics:
			return f.MetricsToMetricsStability()
		case component.DataTypeLogs:
			return f.MetricsToLogsStability()
		}
	case component.DataTypeLogs:
		switch recType {
		case component.DataTypeTraces:
			return f.LogsToTracesStability()
		case component.DataTypeMetrics:
			return f.LogsToMetricsStability()
		case component.DataTypeLogs:
			return f.LogsToLogsStability()
		}
	}
	return component.StabilityLevelUndefined
}
