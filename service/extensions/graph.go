package extensions

import (
	"fmt"
	"hash/fnv"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

type dependencyGraph struct {
	exts  *Extensions
	graph *simple.DirectedGraph
}

type node struct {
	nodeID uint64
	extID  component.ID
}

func (n node) ID() int64 {
	return int64(n.nodeID)
}

func newNode(extID component.ID) *node {
	h := fnv.New64a()
	_, _ = h.Write([]byte(extID.String()))
	return &node{
		nodeID: h.Sum64(),
		extID:  extID,
	}
}

func newDependencyGraph(exts *Extensions) *dependencyGraph {
	return &dependencyGraph{
		exts:  exts,
		graph: simple.NewDirectedGraph(),
	}
}

func (dg *dependencyGraph) computeOrder() ([]component.ID, error) {
	nodes := make(map[component.ID]*node)
	for extID := range dg.exts.extMap {
		n := newNode(extID)
		dg.graph.AddNode(n)
		nodes[extID] = n
	}
	fmt.Printf("%+v\n", nodes)
	for extID, ext := range dg.exts.extMap {
		n := nodes[extID]
		if dep, ok := ext.(extension.DependentExtension); ok {
			for _, depID := range dep.Dependencies() {
				if d, ok := nodes[depID]; ok {
					dg.graph.SetEdge(dg.graph.NewEdge(n, d))
				} else {
					return nil, fmt.Errorf("unable to find extension %s on which extension %s depends", depID, extID)
				}
			}
		}
	}
	fmt.Printf("%+v\n", dg.graph)
	orderedNodes, err := topo.Sort(dg.graph)
	if err != nil {
		return nil, fmt.Errorf("unable to sort the dependency graph: %w", err)
	}

	order := make([]component.ID, len(orderedNodes))
	for i, n := range orderedNodes {
		order[i] = n.(*node).extID
	}
	return order, nil
}
