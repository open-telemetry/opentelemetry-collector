package extensions

import (
	"fmt"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

type node struct {
	nodeID int64
	extID  component.ID
}

func (n node) ID() int64 {
	return n.nodeID
}

func computeOrder(exts *Extensions) ([]component.ID, error) {
	graph := simple.NewDirectedGraph()
	nodes := make(map[component.ID]*node)
	for extID := range exts.extMap {
		n := &node{
			nodeID: int64(len(nodes) + 1),
			extID:  extID,
		}
		graph.AddNode(n)
		nodes[extID] = n
	}
	for extID, ext := range exts.extMap {
		n := nodes[extID]
		if dep, ok := ext.(extension.DependentExtension); ok {
			for _, depID := range dep.Dependencies() {
				if d, ok := nodes[depID]; ok {
					graph.SetEdge(graph.NewEdge(n, d))
				} else {
					return nil, fmt.Errorf("unable to find extension %s on which extension %s depends", depID, extID)
				}
			}
		}
	}
	orderedNodes, err := topo.Sort(graph)
	if err != nil {
		return nil, fmt.Errorf("unable to sort the extenions dependency graph: %w", err)
	}

	order := make([]component.ID, len(orderedNodes))
	for i, n := range orderedNodes {
		order[i] = n.(*node).extID
	}
	return order, nil
}
