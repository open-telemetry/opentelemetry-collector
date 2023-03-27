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
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/capabilityconsumer"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

// Settings holds configuration for building builtPipelines.
type Settings struct {
	Telemetry component.TelemetrySettings
	BuildInfo component.BuildInfo

	ReceiverBuilder  *receiver.Builder
	ProcessorBuilder *processor.Builder
	ExporterBuilder  *exporter.Builder
	ConnectorBuilder *connector.Builder

	// PipelineConfigs is a map of component.ID to PipelineConfig.
	PipelineConfigs map[component.ID]*PipelineConfig
}

type PipelineConfig struct {
	Receivers  []component.ID
	Processors []component.ID
	Exporters  []component.ID
}

type Graph struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelines map[component.ID]*pipelineNodes
}

func Build(ctx context.Context, set Settings) (*Graph, error) {
	pipelines := &Graph{
		componentGraph: simple.NewDirectedGraph(),
		pipelines:      make(map[component.ID]*pipelineNodes, len(set.PipelineConfigs)),
	}
	for pipelineID := range set.PipelineConfigs {
		pipelines.pipelines[pipelineID] = &pipelineNodes{
			receivers: make(map[int64]graph.Node),
			exporters: make(map[int64]graph.Node),
		}
	}
	pipelines.createNodes(set)
	pipelines.createEdges()
	return pipelines, pipelines.buildComponents(ctx, set)
}

// Creates a node for each instance of a component and adds it to the graph
func (g *Graph) createNodes(set Settings) {
	// Keep track of connectors and where they are used. (map[connectorID][]pipelineID)
	connectorsAsExporter := make(map[component.ID][]component.ID)
	connectorsAsReceiver := make(map[component.ID][]component.ID)

	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		pipe := g.pipelines[pipelineID]
		for _, recvID := range pipelineCfg.Receivers {
			if set.ConnectorBuilder.IsConfigured(recvID) {
				connectorsAsReceiver[recvID] = append(connectorsAsReceiver[recvID], pipelineID)
				continue
			}
			rcvrNode := g.createReceiver(pipelineID.Type(), recvID)
			pipe.receivers[rcvrNode.ID()] = rcvrNode
		}

		pipe.capabilitiesNode = newCapabilitiesNode(pipelineID)

		for _, procID := range pipelineCfg.Processors {
			pipe.processors = append(pipe.processors, g.createProcessor(pipelineID, procID))
		}

		pipe.fanOutNode = newFanOutNode(pipelineID)

		for _, exprID := range pipelineCfg.Exporters {
			if set.ConnectorBuilder.IsConfigured(exprID) {
				connectorsAsExporter[exprID] = append(connectorsAsExporter[exprID], pipelineID)
				continue
			}
			expNode := g.createExporter(pipelineID.Type(), exprID)
			pipe.exporters[expNode.ID()] = expNode
		}
	}

	for connID, exprPipelineIDs := range connectorsAsExporter {
		for _, eID := range exprPipelineIDs {
			for _, rID := range connectorsAsReceiver[connID] {
				connNode := g.createConnector(eID, rID, connID)
				g.pipelines[eID].exporters[connNode.ID()] = connNode
				g.pipelines[rID].receivers[connNode.ID()] = connNode
			}
		}
	}
}

func (g *Graph) createReceiver(pipelineType component.DataType, recvID component.ID) *receiverNode {
	rcvrNode := newReceiverNode(pipelineType, recvID)
	if node := g.componentGraph.Node(rcvrNode.ID()); node != nil {
		return node.(*receiverNode)
	}
	g.componentGraph.AddNode(rcvrNode)
	return rcvrNode
}

func (g *Graph) createProcessor(pipelineID, procID component.ID) *processorNode {
	procNode := newProcessorNode(pipelineID, procID)
	g.componentGraph.AddNode(procNode)
	return procNode
}

func (g *Graph) createExporter(pipelineType component.DataType, exprID component.ID) *exporterNode {
	expNode := newExporterNode(pipelineType, exprID)
	if node := g.componentGraph.Node(expNode.ID()); node != nil {
		return node.(*exporterNode)
	}
	g.componentGraph.AddNode(expNode)
	return expNode
}

func (g *Graph) createConnector(exprPipelineID, rcvrPipelineID, connID component.ID) *connectorNode {
	connNode := newConnectorNode(exprPipelineID.Type(), rcvrPipelineID.Type(), connID)
	if node := g.componentGraph.Node(connNode.ID()); node != nil {
		return node.(*connectorNode)
	}
	g.componentGraph.AddNode(connNode)
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
		switch n := node.(type) {
		case *receiverNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ReceiverBuilder, g.nextConsumers(n.ID()))
		case *processorNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ProcessorBuilder, g.nextConsumers(n.ID())[0])
		case *exporterNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ExporterBuilder)
		case *connectorNode:
			err = n.buildComponent(ctx, set.Telemetry, set.BuildInfo, set.ConnectorBuilder, g.nextConsumers(n.ID()))
		case *capabilitiesNode:
			capability := consumer.Capabilities{MutatesData: false}
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
		comp, ok := nodes[i].(component.Component)
		if !ok {
			// Skip capabilities/fanout nodes
			continue
		}
		errs = multierr.Append(errs, comp.Shutdown(ctx))
	}
	return errs
}

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
