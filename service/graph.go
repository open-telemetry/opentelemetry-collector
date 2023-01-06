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
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

var _ pipelines = (*pipelinesGraph)(nil)

type pipelinesGraph struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelines map[component.ID]*pipelineNodes
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
			continue
		}
		errs = multierr.Append(errs, comp.Shutdown(ctx))
	}
	return errs
}

func buildPipelinesGraph(ctx context.Context, set pipelinesSettings) (pipelines, error) {
	pipelines := &pipelinesGraph{
		componentGraph: simple.NewDirectedGraph(),
		pipelines:      make(map[component.ID]*pipelineNodes, len(set.PipelineConfigs)),
	}
	for pipelineID := range set.PipelineConfigs {
		pipelines.pipelines[pipelineID] = newPipelineNodes()
	}

	if err := pipelines.createNodes(set); err != nil {
		return nil, err
	}
	pipelines.createEdges()
	if err := pipelines.buildNodes(ctx, set); err != nil {
		return nil, err
	}
	return pipelines, nil
}

// Creates a node for each instance of a component and adds it to the graph
func (g *pipelinesGraph) createNodes(set pipelinesSettings) error {

	// map[connectorID]pipelineIDs
	// Keep track of connectors and where they are used.
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

	if len(connectorsAsExporter) != len(connectorsAsReceiver) {
		return fmt.Errorf("each connector must be used as both receiver and exporter")
	}
	for connID, exprPipelineIDs := range connectorsAsExporter {
		rcvrPipelineIDs, ok := connectorsAsReceiver[connID]
		if !ok {
			return fmt.Errorf("connector %q must be used as receiver, only found as exporter", connID)
		}
		for _, eID := range exprPipelineIDs {
			for _, rID := range rcvrPipelineIDs {
				g.addConnector(eID, rID, connID)
			}
		}
	}
	return nil
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
		return cycleErr(err)
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		switch n := node.(type) {
		case *receiverNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Receivers, g.nextConsumers(n.ID()))
		case *processorNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Processors, g.nextConsumers(n.ID()))
		case *connectorNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Connectors, g.nextConsumers(n.ID()))
		case *exporterNode:
			err = n.build(ctx, set.Telemetry, set.BuildInfo, set.Exporters)
		case *fanInNode:
			err = n.build(g.nextConsumers(n.ID()), g.nextProcessors(n.ID()))
		case *fanOutNode:
			err = n.build(g.nextConsumers(n.ID()))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

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
		default:
			panic(fmt.Sprintf("type cannot be consumer: %T", next))
		}
	}
	return nexts
}

func (g *pipelinesGraph) nextProcessors(nodeID int64) []*processorNode {
	nexts := make([]*processorNode, 0)
	for {
		nextNodes := g.componentGraph.From(nodeID)
		if nextNodes.Len() != 1 {
			break
		}
		procNode, ok := nextNodes.Node().(*processorNode)
		if !ok {
			break
		}
		nexts = append(nexts, procNode)
	}
	return nexts
}

// A node-based representation of a pipeline configuration.
type pipelineNodes struct {

	// Use maps for receivers and exporters to assist with deduplication of connector instances.
	receivers map[int64]graph.Node
	exporters map[int64]graph.Node

	// The order of processors is very important. Therefore use a slice for processors.
	processors []graph.Node
}

func newPipelineNodes() *pipelineNodes {
	return &pipelineNodes{
		receivers: make(map[int64]graph.Node),
		exporters: make(map[int64]graph.Node),
	}
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

func (g *pipelinesGraph) GetExporters() map[component.DataType]map[component.ID]component.Component {
	exportersMap := make(map[component.DataType]map[component.ID]component.Component)
	exportersMap[component.DataTypeTraces] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeMetrics] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeLogs] = make(map[component.ID]component.Component)

	for _, pg := range g.pipelines {
		for _, expNode := range pg.exporters {
			expOrConnNode := g.componentGraph.Node(expNode.ID())
			expNode, ok := expOrConnNode.(*exporterNode)
			if !ok {
				continue
			}
			exportersMap[expNode.pipelineType][expNode.componentID] = expNode.Component
		}
	}
	return exportersMap
}

func (g *pipelinesGraph) HandleZPages(w http.ResponseWriter, r *http.Request) {
	qValues := r.URL.Query()
	pipelineName := qValues.Get(zPipelineName)
	componentName := qValues.Get(zComponentName)
	componentKind := qValues.Get(zComponentKind)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	zpages.WriteHTMLPageHeader(w, zpages.HeaderData{Title: "Pipelines"})
	zpages.WriteHTMLPipelinesSummaryTable(w, g.getPipelinesSummaryTableData())
	if pipelineName != "" && componentName != "" && componentKind != "" {
		fullName := componentName
		if componentKind == "processor" {
			fullName = pipelineName + "/" + componentName
		}
		zpages.WriteHTMLComponentHeader(w, zpages.ComponentHeaderData{
			Name: componentKind + ": " + fullName,
		})
		// TODO: Add config + status info.
	}
	zpages.WriteHTMLPageFooter(w)
}

func (g *pipelinesGraph) getPipelinesSummaryTableData() zpages.SummaryPipelinesTableData {
	sumData := zpages.SummaryPipelinesTableData{}
	sumData.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(g.pipelines))

	for pipelineID, pipelineNodes := range g.pipelines {
		row := zpages.SummaryPipelinesTableRowData{
			FullName:   pipelineID.String(),
			InputType:  string(pipelineID.Type()),
			Receivers:  make([]string, len(pipelineNodes.receivers)),
			Processors: make([]string, len(pipelineNodes.processors)),
			Exporters:  make([]string, len(pipelineNodes.exporters)),
		}
		for _, recvNode := range pipelineNodes.receivers {
			switch node := recvNode.(type) {
			case *receiverNode:
				row.Receivers = append(row.Receivers, node.componentID.String())
			case *connectorNode:
				row.Receivers = append(row.Receivers, node.componentID.String()+" (connector)")
			}
		}
		for _, procNode := range pipelineNodes.processors {
			node := procNode.(*processorNode)
			row.Processors = append(row.Processors, node.componentID.String())
			row.MutatesData = row.MutatesData || node.Component.(baseConsumer).Capabilities().MutatesData
		}
		for _, expNode := range pipelineNodes.exporters {
			switch node := expNode.(type) {
			case *exporterNode:
				row.Exporters = append(row.Exporters, node.componentID.String())
			case *connectorNode:
				row.Exporters = append(row.Exporters, node.componentID.String()+" (connector)")
			}
		}
		sumData.Rows = append(sumData.Rows, row)
	}

	sort.Slice(sumData.Rows, func(i, j int) bool {
		return sumData.Rows[i].FullName < sumData.Rows[j].FullName
	})
	return sumData
}

func cycleErr(err error) error {
	var topoErr topo.Unorderable
	if !errors.As(err, &topoErr) {
		return topoErr
	}

	// It is possible to have multiple cycles, but it is enough to report the first cycle
	cycle := topoErr[0]
	nodeCycle := make([]string, 0, len(cycle)+1)
	for _, node := range cycle {
		switch n := node.(type) {
		case *receiverNode:
			nodeCycle = append(nodeCycle, fmt.Sprintf("receiver \"%s\"", n.componentID))
		case *processorNode:
			nodeCycle = append(nodeCycle, fmt.Sprintf("processor \"%s\"", n.componentID))
		case *exporterNode:
			nodeCycle = append(nodeCycle, fmt.Sprintf("exporter \"%s\"", n.componentID))
		case *connectorNode:
			nodeCycle = append(nodeCycle, fmt.Sprintf("connector \"%s\"", n.componentID))
		}
	}
	// Prepend the last node to clarify the cycle
	nodeCycle = append([]string{nodeCycle[len(nodeCycle)-1]}, nodeCycle...)
	return fmt.Errorf("cycle detected: %s", strings.Join(nodeCycle, ", "))
}
