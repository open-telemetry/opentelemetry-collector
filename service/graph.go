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
	"net/http"

	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
)

var _ pipelines = (*pipelinesGraph)(nil)

type pipelinesGraph struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelines map[component.ID]*pipelineNodes
}

func buildPipelinesGraph(ctx context.Context, set pipelinesSettings) (pipelines, error) {
	pipelines := &pipelinesGraph{
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
	return pipelines, pipelines.buildNodes(ctx, set)
}

// Creates a node for each instance of a component and adds it to the graph
func (g *pipelinesGraph) createNodes(set pipelinesSettings) {
	// Keep track of connectors and where they are used. (map[connectorID][]pipelineID)
	connectorsAsExporter := make(map[component.ID][]component.ID)
	connectorsAsReceiver := make(map[component.ID][]component.ID)

	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		for _, recvID := range pipelineCfg.Receivers {
			if set.Connectors.IsConfigured(recvID) {
				connectorsAsReceiver[recvID] = append(connectorsAsReceiver[recvID], pipelineID)
				continue
			}
			g.addReceiver(pipelineID, recvID)
		}
		for _, procID := range pipelineCfg.Processors {
			g.addProcessor(pipelineID, procID)
		}
		for _, exprID := range pipelineCfg.Exporters {
			if set.Connectors.IsConfigured(exprID) {
				connectorsAsExporter[exprID] = append(connectorsAsExporter[exprID], pipelineID)
				continue
			}
			g.addExporter(pipelineID, exprID)
		}
	}

	for connID, exprPipelineIDs := range connectorsAsExporter {
		for _, eID := range exprPipelineIDs {
			for _, rID := range connectorsAsReceiver[connID] {
				g.addConnector(eID, rID, connID)
			}
		}
	}
}

func (g *pipelinesGraph) addReceiver(pipelineID, recvID component.ID) {
	node := newReceiverNode(pipelineID, recvID)
	if rcvrNode := g.componentGraph.Node(node.ID()); rcvrNode != nil {
		g.pipelines[pipelineID].addReceiver(rcvrNode)
		return
	}
	g.pipelines[pipelineID].addReceiver(node)
	g.componentGraph.AddNode(node)
}

func (g *pipelinesGraph) addProcessor(pipelineID, procID component.ID) {
	node := newProcessorNode(pipelineID, procID)
	g.pipelines[pipelineID].addProcessor(node)
	g.componentGraph.AddNode(node)
}

func (g *pipelinesGraph) addExporter(pipelineID, exprID component.ID) {
	node := newExporterNode(pipelineID, exprID)
	if expNode := g.componentGraph.Node(node.ID()); expNode != nil {
		g.pipelines[pipelineID].addExporter(expNode)
		return
	}
	g.pipelines[pipelineID].addExporter(node)
	g.componentGraph.AddNode(node)
}

func (g *pipelinesGraph) addConnector(exprPipelineID, rcvrPipelineID, connID component.ID) {
	node := newConnectorNode(exprPipelineID.Type(), rcvrPipelineID.Type(), connID)
	if connNode := g.componentGraph.Node(node.ID()); connNode != nil {
		g.pipelines[exprPipelineID].addExporter(connNode)
		g.pipelines[rcvrPipelineID].addReceiver(connNode)
		return
	}
	g.pipelines[exprPipelineID].addExporter(node)
	g.pipelines[rcvrPipelineID].addReceiver(node)
	g.componentGraph.AddNode(node)
}

func (g *pipelinesGraph) createEdges() {
	for pipelineID, pg := range g.pipelines {
		fanOutToExporters := newFanOutNode(pipelineID)
		for _, exporter := range pg.exporters {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(fanOutToExporters, exporter))
		}

		if len(pg.processors) == 0 {
			for _, receiver := range pg.receivers {
				g.componentGraph.SetEdge(g.componentGraph.NewEdge(receiver, fanOutToExporters))
			}
			continue
		}

		fanInToProcessors := newFanInNode(pipelineID)
		for _, receiver := range pg.receivers {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(receiver, fanInToProcessors))
		}

		g.componentGraph.SetEdge(g.componentGraph.NewEdge(fanInToProcessors, pg.processors[0]))
		for i := 0; i+1 < len(pg.processors); i++ {
			g.componentGraph.SetEdge(g.componentGraph.NewEdge(pg.processors[i], pg.processors[i+1]))
		}
		g.componentGraph.SetEdge(g.componentGraph.NewEdge(pg.processors[len(pg.processors)-1], fanOutToExporters))
	}
}

func (g *pipelinesGraph) buildNodes(ctx context.Context, set pipelinesSettings) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return err // TODO clean up error message
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		switch n := node.(type) {
		case *receiverNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Receivers, g.nextConsumers(n.ID()))
		case *processorNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Processors, g.nextConsumers(n.ID())[0])
		case *connectorNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Connectors, g.nextConsumers(n.ID()))
		case *exporterNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Exporters)
		case *fanInNode:
			n.build(g.nextConsumers(n.ID())[0], g.nextProcessors(n.ID()))
		case *fanOutNode:
			n.build(g.nextConsumers(n.ID()))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Find all nodes
func (g *pipelinesGraph) nextConsumers(nodeID int64) []baseConsumer {
	nextNodes := g.componentGraph.From(nodeID)
	nexts := make([]baseConsumer, 0, nextNodes.Len())
	for nextNodes.Next() {
		switch next := nextNodes.Node().(type) {
		case *processorNode:
			nexts = append(nexts, next.Component.(baseConsumer))
		case *exporterNode:
			nexts = append(nexts, next.Component.(baseConsumer))
		case *connectorNode:
			nexts = append(nexts, next.Component.(baseConsumer))
		case *fanInNode:
			nexts = append(nexts, next.baseConsumer)
		case *fanOutNode:
			nexts = append(nexts, next.baseConsumer)
		}
	}
	return nexts
}

// Get all processors in this pipeline
func (g *pipelinesGraph) nextProcessors(nodeID int64) []*processorNode {
	nextNodes := g.componentGraph.From(nodeID)
	if procNode, ok := nextNodes.Node().(*processorNode); ok {
		return append([]*processorNode{procNode}, g.nextProcessors(procNode.ID())...)
	}
	return []*processorNode{}
}

// A node-based representation of a pipeline configuration.
type pipelineNodes struct {
	// Use maps for receivers and exporters to assist with deduplication of connector instances.
	receivers map[int64]graph.Node
	exporters map[int64]graph.Node

	// The order of processors is very important. Therefore use a slice for processors.
	processors []graph.Node
}

func (p *pipelineNodes) addReceiver(node graph.Node) {
	p.receivers[node.ID()] = node
}
func (p *pipelineNodes) addProcessor(node graph.Node) {
	p.processors = append(p.processors, node)
}
func (p *pipelineNodes) addExporter(node graph.Node) {
	p.exporters[node.ID()] = node
}

func (g *pipelinesGraph) StartAll(ctx context.Context, host component.Host) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return err
	}

	// Start exporters first, and work towards receivers
	for i := len(nodes) - 1; i >= 0; i-- {
		comp, ok := nodes[i].(component.Component)
		if !ok {
			// Skip fanin/out nodes
			continue
		}
		if compErr := comp.Start(ctx, host); compErr != nil {
			return compErr
		}
	}
	return nil
}

func (g *pipelinesGraph) ShutdownAll(ctx context.Context) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
		return err
	}

	// Stop receivers first, and work towards exporters
	var errs error
	for i := 0; i < len(nodes); i++ {
		comp, ok := nodes[i].(component.Component)
		if !ok {
			// Skip fanin/out nodes
			continue
		}
		errs = multierr.Append(errs, comp.Shutdown(ctx))
	}
	return errs
}

func (g *pipelinesGraph) GetExporters() map[component.DataType]map[component.ID]component.Component {
	// TODO actual implementation
	return nil
}

func (g *pipelinesGraph) HandleZPages(w http.ResponseWriter, r *http.Request) {
	// TODO actual implementation
}
