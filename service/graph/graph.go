// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph // import "go.opentelemetry.io/collector/service/graph"

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
	"go.opentelemetry.io/collector/service/pipelines"
)

// Settings holds configuration for building a Graph.
type Settings struct {
	// Telemetry specifies the telemetry settings.
	Telemetry component.TelemetrySettings
	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// ReceiverBuilder is a helper struct that given a set of Configs and Factories helps with creating receivers.
	ReceiverBuilder *receiver.Builder
	// ProcessorBuilder is a helper struct that given a set of Configs and Factories helps with creating processors.
	ProcessorBuilder *processor.Builder
	// ExporterBuilder is a helper struct that given a set of Configs and Factories helps with creating exporters.
	ExporterBuilder *exporter.Builder
	// ConnectorBuilder is a helper struct that given a set of Configs and Factories helps with creating connectors.
	ConnectorBuilder *connector.Builder
}

// Pipelines of collector components.
type Pipelines struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelines map[component.ID]*pipelineNodes
}

// New builds a collector pipeline set.
func New(ctx context.Context, set Settings, cfg pipelines.Config) (*Pipelines, error) {
	pipelines := &Pipelines{
		componentGraph: simple.NewDirectedGraph(),
		pipelines:      make(map[component.ID]*pipelineNodes, len(cfg)),
	}
	for pipelineID := range cfg {
		pipelines.pipelines[pipelineID] = &pipelineNodes{
			receivers: make(map[int64]graph.Node),
			exporters: make(map[int64]graph.Node),
		}
	}
	if err := pipelines.createNodes(set, cfg); err != nil {
		return nil, err
	}
	pipelines.createEdges()
	return pipelines, pipelines.buildComponents(ctx, set)
}

// Creates a node for each instance of a component and adds it to the graph
func (p *Pipelines) createNodes(set Settings, cfg pipelines.Config) error {
	// Build a list of all connectors for easy reference
	connectors := make(map[component.ID]struct{})

	// Keep track of connectors and where they are used. (map[connectorID][]pipelineID)
	connectorsAsExporter := make(map[component.ID][]component.ID)
	connectorsAsReceiver := make(map[component.ID][]component.ID)

	for pipelineID, pipelineCfg := range cfg {
		pipe := p.pipelines[pipelineID]
		for _, recvID := range pipelineCfg.Receivers {
			if set.ConnectorBuilder.IsConfigured(recvID) {
				connectors[recvID] = struct{}{}
				connectorsAsReceiver[recvID] = append(connectorsAsReceiver[recvID], pipelineID)
				continue
			}
			rcvrNode := p.createReceiver(pipelineID.Type(), recvID)
			pipe.receivers[rcvrNode.ID()] = rcvrNode
		}

		pipe.capabilitiesNode = newCapabilitiesNode(pipelineID)

		for _, procID := range pipelineCfg.Processors {
			pipe.processors = append(pipe.processors, p.createProcessor(pipelineID, procID))
		}

		pipe.fanOutNode = newFanOutNode(pipelineID)

		for _, exprID := range pipelineCfg.Exporters {
			if set.ConnectorBuilder.IsConfigured(exprID) {
				connectors[exprID] = struct{}{}
				connectorsAsExporter[exprID] = append(connectorsAsExporter[exprID], pipelineID)
				continue
			}
			expNode := p.createExporter(pipelineID.Type(), exprID)
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
				connNode := p.createConnector(eID, rID, connID)
				p.pipelines[eID].exporters[connNode.ID()] = connNode
				p.pipelines[rID].receivers[connNode.ID()] = connNode
			}
		}
	}
	return nil
}

func (p *Pipelines) createReceiver(pipelineType component.DataType, recvID component.ID) *receiverNode {
	rcvrNode := newReceiverNode(pipelineType, recvID)
	if node := p.componentGraph.Node(rcvrNode.ID()); node != nil {
		return node.(*receiverNode)
	}
	p.componentGraph.AddNode(rcvrNode)
	return rcvrNode
}

func (p *Pipelines) createProcessor(pipelineID, procID component.ID) *processorNode {
	procNode := newProcessorNode(pipelineID, procID)
	p.componentGraph.AddNode(procNode)
	return procNode
}

func (p *Pipelines) createExporter(pipelineType component.DataType, exprID component.ID) *exporterNode {
	expNode := newExporterNode(pipelineType, exprID)
	if node := p.componentGraph.Node(expNode.ID()); node != nil {
		return node.(*exporterNode)
	}
	p.componentGraph.AddNode(expNode)
	return expNode
}

func (p *Pipelines) createConnector(exprPipelineID, rcvrPipelineID, connID component.ID) *connectorNode {
	connNode := newConnectorNode(exprPipelineID.Type(), rcvrPipelineID.Type(), connID)
	if node := p.componentGraph.Node(connNode.ID()); node != nil {
		return node.(*connectorNode)
	}
	p.componentGraph.AddNode(connNode)
	return connNode
}

func (p *Pipelines) createEdges() {
	for _, pg := range p.pipelines {
		for _, receiver := range pg.receivers {
			p.componentGraph.SetEdge(p.componentGraph.NewEdge(receiver, pg.capabilitiesNode))
		}

		var from, to graph.Node
		from = pg.capabilitiesNode
		for _, processor := range pg.processors {
			to = processor
			p.componentGraph.SetEdge(p.componentGraph.NewEdge(from, to))
			from = processor
		}
		to = pg.fanOutNode
		p.componentGraph.SetEdge(p.componentGraph.NewEdge(from, to))

		for _, exporter := range pg.exporters {
			p.componentGraph.SetEdge(p.componentGraph.NewEdge(pg.fanOutNode, exporter))
		}
	}
}

func (p *Pipelines) buildComponents(ctx context.Context, set Settings) error {
	nodes, err := topo.Sort(p.componentGraph)
	if err != nil {
		return cycleErr(err, topo.DirectedCyclesIn(p.componentGraph))
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		switch n := node.(type) {
		case *receiverNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ReceiverBuilder, p.nextConsumers(n.ID()))
		case *processorNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ProcessorBuilder, p.nextConsumers(n.ID())[0])
		case *exporterNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ExporterBuilder)
		case *connectorNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ConnectorBuilder, p.nextConsumers(n.ID()))
		case *capabilitiesNode:
			capability := consumer.Capabilities{MutatesData: false}
			for _, proc := range p.pipelines[n.pipelineID].processors {
				capability.MutatesData = capability.MutatesData || proc.getConsumer().Capabilities().MutatesData
			}
			next := p.nextConsumers(n.ID())[0]
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
			nexts := p.nextConsumers(n.ID())
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
func (p *Pipelines) nextConsumers(nodeID int64) []baseConsumer {
	nextNodes := p.componentGraph.From(nodeID)
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

// StartAll components in the graph in topological order.
func (p *Pipelines) StartAll(ctx context.Context, host component.Host) error {
	nodes, err := topo.Sort(p.componentGraph)
	if err != nil {
		return err
	}

	// Start in reverse topological order so that downstream components
	// are started before upstream components. This ensures that each
	// component's consumer is ready to consume.
	for i := len(nodes) - 1; i >= 0; i-- {
		comp, ok := nodes[i].(component.Component)
		if !ok {
			// Skip capabilities/fanout nodes
			continue
		}
		if compErr := comp.Start(ctx, host); compErr != nil {
			return compErr
		}
	}
	return nil
}

// ShutdownAll components in the graph in topological order.
func (p *Pipelines) ShutdownAll(ctx context.Context) error {
	nodes, err := topo.Sort(p.componentGraph)
	if err != nil {
		return err
	}

	// Stop in topological order so that upstream components
	// are stopped before downstream components.  This ensures
	// that each component has a chance to drain to its consumer
	// before the consumer is stopped.
	var errs error
	for i := 0; i < len(nodes); i++ {
		comp, ok := nodes[i].(component.Component)
		if !ok {
			// Skip capabilities/fanout nodes
			continue
		}
		errs = multierr.Append(errs, comp.Shutdown(ctx))
	}
	return errs
}

// Deprecated: [0.79.0] This function will be removed in the future.
// Several components in the contrib repository use this function so it cannot be removed
// before those cases are removed. In most cases, use of this function can be replaced by a
// connector. See https://github.com/open-telemetry/opentelemetry-collector/issues/7370 and
// https://github.com/open-telemetry/opentelemetry-collector/pull/7390#issuecomment-1483710184
// for additional information.
func (p *Pipelines) GetExporters() map[component.DataType]map[component.ID]component.Component {
	exportersMap := make(map[component.DataType]map[component.ID]component.Component)
	exportersMap[component.DataTypeTraces] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeMetrics] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeLogs] = make(map[component.ID]component.Component)

	for _, pg := range p.pipelines {
		for _, expNode := range pg.exporters {
			// Skip connectors, otherwise individual components can introduce cycles
			if expNode, ok := p.componentGraph.Node(expNode.ID()).(*exporterNode); ok {
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
