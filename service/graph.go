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
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/internal/zpages"
)

var _ pipelines = (*pipelinesGraph)(nil)

type pipelinesGraph struct {
	// All component instances represented as nodes, with directed edges indicating data flow.
	componentGraph *simple.DirectedGraph

	// Keep track of how nodes relate to pipelines, so we can declare edges in the graph.
	pipelineGraphs map[component.ID]*pipelineGraph
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

func buildPipelinesGraph(ctx context.Context, set pipelinesSettings) (Pipelines, error) {
	pipelines := &pipelinesGraph{
		componentGraph: simple.NewDirectedGraph(),
		pipelineGraphs: make(map[component.ID]*pipelineGraph, len(set.PipelineConfigs)),
	}
	for pipelineID := range set.PipelineConfigs {
		pipelines.pipelineGraphs[pipelineID] = newPipelineGraph()
	}

	if err := pipelines.createNodes(set); err != nil {
		return nil, err
	}

	pipelines.createEdges()

	if err := pipelines.buildNodes(ctx, set.Telemetry, set.BuildInfo); err != nil {
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
			if connCfg := set.Connectors.Config(recvID); connCfg != nil {
				connectorsAsReceiver[recvID] = append(connectorsAsReceiver[recvID], pipelineID)
				continue
			}
			if err := g.addReceiver(pipelineID, recvID, set.Receivers); err != nil {
				return err
			}
		}

		for _, procID := range pipelineCfg.Processors {
			if err := g.addProcessor(pipelineID, procID, set.Processors); err != nil {
				return err
			}
		}
	}

	// All exporters added after all receivers to ensure deterministic error when a connector is not configured
	for pipelineID, pipelineCfg := range set.PipelineConfigs {
		for _, exprID := range pipelineCfg.Exporters {
			if connCfg := set.Connectors.Config(exprID); connCfg != nil {
				connectorsAsExporter[exprID] = append(connectorsAsExporter[exprID], pipelineID)
				continue
			}
			if err := g.addExporter(pipelineID, exprID, set.Exporters); err != nil {
				return err
			}
		}
	}

	return g.addConnectors(connectorsAsExporter, connectorsAsReceiver, set.Connectors)
}

func (g *pipelinesGraph) addReceiver(pipelineID, recvID component.ID, builder *receiver.Builder) error {
	receiverNodeID := newReceiverNodeID(pipelineID.Type(), recvID)

	if rcvrNode := g.componentGraph.Node(receiverNodeID.ID()); rcvrNode != nil {
		g.pipelineGraphs[pipelineID].addReceiver(rcvrNode)
		return nil
	}

	cfg := builder.Config(recvID)
	if cfg == nil {
		return fmt.Errorf("receiver %q is not configured", recvID)
	}

	factory := builder.Factory(recvID.Type())
	if factory == nil {
		return fmt.Errorf("receiver factory not available for: %q", recvID)
	}

	rcvrFactory, ok := factory.(receiver.Factory)
	if !ok {
		return fmt.Errorf("factory is not a receiver factory: %q", recvID.Type())
	}

	node := &receiverNode{
		nodeID:       receiverNodeID,
		componentID:  recvID,
		pipelineType: pipelineID.Type(),
		cfg:          cfg,
		factory:      rcvrFactory,
	}
	g.pipelineGraphs[pipelineID].addReceiver(node)
	g.componentGraph.AddNode(node)
	return nil
}

func (g *pipelinesGraph) addProcessor(pipelineID, procID component.ID, builder *processor.Builder) error {
	cfg := builder.Config(procID)
	if cfg == nil {
		return fmt.Errorf("processor %q is not configured", procID)
	}

	factory := builder.Factory(procID.Type())
	if factory == nil {
		return fmt.Errorf("processor factory not available for: %q", procID)
	}

	procFactory, ok := factory.(processor.Factory)
	if !ok {
		return fmt.Errorf("factory is not a processor factory: %q", procID.Type())
	}

	node := &processorNode{
		nodeID:      newProcessorNodeID(pipelineID, procID),
		componentID: procID,
		pipelineID:  pipelineID,
		cfg:         cfg,
		factory:     procFactory,
	}
	g.pipelineGraphs[pipelineID].addProcessor(node)
	g.componentGraph.AddNode(node)
	return nil
}

func (g *pipelinesGraph) addExporter(pipelineID, exprID component.ID, builder *exporter.Builder) error {
	exporterNodeID := newExporterNodeID(pipelineID.Type(), exprID)

	if expNode := g.componentGraph.Node(exporterNodeID.ID()); expNode != nil {
		g.pipelineGraphs[pipelineID].addExporter(expNode)
		return nil
	}

	cfg := builder.Config(exprID)
	if cfg == nil {
		return fmt.Errorf("exporter %q is not configured", exprID)
	}

	factory := builder.Factory(exprID.Type())
	if factory == nil {
		return fmt.Errorf("exporter factory not available for: %q", exprID)
	}

	exprFactory, ok := factory.(exporter.Factory)
	if !ok {
		return fmt.Errorf("factory is not an exporter factory: %q", exprID.Type())
	}

	node := &exporterNode{
		nodeID:       exporterNodeID,
		componentID:  exprID,
		pipelineType: pipelineID.Type(),
		cfg:          cfg,
		factory:      exprFactory,
	}
	g.pipelineGraphs[pipelineID].addExporter(node)
	g.componentGraph.AddNode(node)
	return nil
}

func (g *pipelinesGraph) addConnectors(asExporter, asReceiver map[component.ID][]component.ID, builder *connector.Builder) error {
	if len(asExporter) != len(asReceiver) {
		return fmt.Errorf("each connector must be used as both receiver and exporter")
	}

	// For each pair of pipelines that the connector connects, check if the
	// data type combination is supported. If so, create a connector.
	// A separate instance is created so that the fanoutprocessor will correctly
	// replicate signals to each connector as if it were a separate exporter.
	for connID, exprPipelineIDs := range asExporter {
		rcvrPipelineIDs, ok := asReceiver[connID]
		if !ok {
			return fmt.Errorf("connector %q must be used as receiver, only found as exporter", connID)
		}

		for _, eID := range exprPipelineIDs {
			for _, rID := range rcvrPipelineIDs {
				if err := g.addConnector(eID, rID, connID, builder); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (g *pipelinesGraph) addConnector(
	exprPipelineID, rcvrPipelineID, connID component.ID, builder *connector.Builder) error {
	connectorNodeID := newConnectorNodeID(exprPipelineID.Type(), rcvrPipelineID.Type(), connID)

	if connNode := g.componentGraph.Node(connectorNodeID.ID()); connNode != nil {
		g.pipelineGraphs[exprPipelineID].addExporter(connNode)
		g.pipelineGraphs[rcvrPipelineID].addReceiver(connNode)
		return nil
	}

	cfg := builder.Config(connID)
	if cfg == nil {
		return fmt.Errorf("connector %q is not configured", connID)
	}

	factory := builder.Factory(connID.Type())
	if factory == nil {
		return fmt.Errorf("connector factory not available for: %q", connID)
	}

	connFactory, ok := factory.(connector.Factory)
	if !ok {
		return fmt.Errorf("factory is not a connector factory: %q", connID.Type())
	}

	node := &connectorNode{
		nodeID:           connectorNodeID,
		componentID:      connID,
		exprPipelineType: exprPipelineID.Type(),
		rcvrPipelineType: rcvrPipelineID.Type(),
		cfg:              cfg,
		factory:          connFactory,
	}
	g.pipelineGraphs[exprPipelineID].addExporter(node)
	g.pipelineGraphs[rcvrPipelineID].addReceiver(node)
	g.componentGraph.AddNode(node)
	return nil
}

func (g *pipelinesGraph) createEdges() {
	for pipelineID, pg := range g.pipelineGraphs {
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

func (g *pipelinesGraph) buildNodes(ctx context.Context, tel component.TelemetrySettings, info component.BuildInfo) error {
	nodes, err := topo.Sort(g.componentGraph)
	if err != nil {
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
				nodeCycle = append(nodeCycle, fmt.Sprintf("receiver \"%s\"", n.ComponentID()))
			case *processorNode:
				nodeCycle = append(nodeCycle, fmt.Sprintf("processor \"%s\"", n.ComponentID()))
			case *exporterNode:
				nodeCycle = append(nodeCycle, fmt.Sprintf("exporter \"%s\"", n.ComponentID()))
			case *connectorNode:
				nodeCycle = append(nodeCycle, fmt.Sprintf("connector \"%s\"", n.ComponentID()))
			}
		}
		// Prepend the last node to clarify the cycle
		nodeCycle = append([]string{nodeCycle[len(nodeCycle)-1]}, nodeCycle...)
		return fmt.Errorf("cycle detected: %s", strings.Join(nodeCycle, ", "))
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		switch n := node.(type) {
		case *receiverNode:
			nexts := g.nextConsumers(n.ID())
			if len(nexts) == 0 {
				return fmt.Errorf("receiver %q has no next consumer: %w", n.componentID, err)
			}
			err = n.build(ctx, tel, info, nexts)
			if err != nil {
				return err
			}
		case *processorNode:
			nexts := g.nextConsumers(n.ID())
			if len(nexts) == 0 {
				return fmt.Errorf("processor %q has no next consumer: %w", n.componentID, err)
			}
			if len(nexts) > 1 {
				return fmt.Errorf("processor %q has multiple consumers", n.componentID)
			}
			err = n.build(ctx, tel, info, nexts[0])
			if err != nil {
				return err
			}
		case *connectorNode:
			nexts := g.nextConsumers(n.ID())
			if len(nexts) == 0 {
				return fmt.Errorf("connector %q has no next consumer: %w", n.componentID, err)
			}
			err = n.build(ctx, tel, info, nexts)
			if err != nil {
				return err
			}
		case *fanInNode:
			nexts := g.nextConsumers(n.ID())
			if len(nexts) != 1 {
				return fmt.Errorf("fan-in in pipeline %q must have one consumer: %w", n.pipelineID, err)
			}
			n.build(nexts[0], g.nextProcessors(n.ID()))
		case *fanOutNode:
			nexts := g.nextConsumers(n.ID())
			if len(nexts) == 0 {
				return fmt.Errorf("fan-out in pipeline %q has no next consumer: %w", n.pipelineID, err)
			}
			err = n.build(nexts)
			if err != nil {
				return err
			}
		case *exporterNode:
			err = n.build(ctx, tel, info)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *pipelinesGraph) nextConsumers(nodeID int64) []baseConsumer {
	nextNodes := g.componentGraph.From(nodeID)
	nextConsumers := make([]baseConsumer, 0, nextNodes.Len())
	for nextNodes.Next() {
		switch next := nextNodes.Node().(type) {
		case *processorNode:
			nextConsumers = append(nextConsumers, next.Component.(baseConsumer))
		case *exporterNode:
			nextConsumers = append(nextConsumers, next.Component.(baseConsumer))
		case *connectorNode:
			nextConsumers = append(nextConsumers, next.Component.(baseConsumer))
		case *fanInNode:
			nextConsumers = append(nextConsumers, next.baseConsumer)
		case *fanOutNode:
			nextConsumers = append(nextConsumers, next.baseConsumer)
		default:
			panic(fmt.Sprintf("type cannot be consumer: %T", next))
		}
	}
	return nextConsumers
}

func (g *pipelinesGraph) nextProcessors(nodeID int64) []*processorNode {
	nextProcessors := make([]*processorNode, 0)
	for {
		nextNodes := g.componentGraph.From(nodeID)
		if nextNodes.Len() != 1 {
			break
		}
		procNode, ok := nextNodes.Node().(*processorNode)
		if !ok {
			break
		}
		nextProcessors = append(nextProcessors, procNode)
	}
	return nextProcessors
}

// A node-based representation of a pipeline configuration.
type pipelineGraph struct {

	// Use maps for receivers and exporters to assist with deduplication of connector instances.
	receivers map[int64]graph.Node
	exporters map[int64]graph.Node

	// The order of processors is very important. Therefore use a slice for processors.
	processors []graph.Node
}

func newPipelineGraph() *pipelineGraph {
	return &pipelineGraph{
		receivers: make(map[int64]graph.Node),
		exporters: make(map[int64]graph.Node),
	}
}

func (p *pipelineGraph) addReceiver(node graph.Node) {
	p.receivers[node.ID()] = node
}
func (p *pipelineGraph) addProcessor(node graph.Node) {
	p.processors = append(p.processors, node)
}
func (p *pipelineGraph) addExporter(node graph.Node) {
	p.exporters[node.ID()] = node
}

func (g *pipelinesGraph) GetExporters() map[component.DataType]map[component.ID]component.Component {
	exportersMap := make(map[component.DataType]map[component.ID]component.Component)
	exportersMap[component.DataTypeTraces] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeMetrics] = make(map[component.ID]component.Component)
	exportersMap[component.DataTypeLogs] = make(map[component.ID]component.Component)

	for _, pg := range g.pipelineGraphs {
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
	sumData.Rows = make([]zpages.SummaryPipelinesTableRowData, 0, len(g.pipelineGraphs))

	for pipelineID, pipelineGraph := range g.pipelineGraphs {
		row := zpages.SummaryPipelinesTableRowData{
			FullName:   pipelineID.String(),
			InputType:  string(pipelineID.Type()),
			Receivers:  make([]string, len(pipelineGraph.receivers)),
			Processors: make([]string, len(pipelineGraph.processors)),
			Exporters:  make([]string, len(pipelineGraph.exporters)),
		}
		for _, recvNode := range pipelineGraph.receivers {
			switch node := recvNode.(type) {
			case *receiverNode:
				row.Receivers = append(row.Receivers, node.componentID.String())
			case *connectorNode:
				row.Receivers = append(row.Receivers, node.componentID.String()+" (connector)")
			}
		}
		for _, procNode := range pipelineGraph.processors {
			node := procNode.(*processorNode)
			row.Processors = append(row.Processors, node.componentID.String())
			row.MutatesData = row.MutatesData || node.Component.(baseConsumer).Capabilities().MutatesData
		}
		for _, expNode := range pipelineGraph.exporters {
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
