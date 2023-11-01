// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"fmt"
	"strings"

	"gonum.org/v1/gonum/graph"
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
		if dep, ok := ext.(extension.Dependent); ok {
			for _, depID := range dep.Dependencies() {
				if d, ok := nodes[depID]; ok {
					graph.SetEdge(graph.NewEdge(d, n))
				} else {
					return nil, fmt.Errorf("unable to find extension %s on which extension %s depends", depID, extID)
				}
			}
		}
	}
	orderedNodes, err := topo.Sort(graph)
	if err != nil {
		return nil, cycleErr(err, topo.DirectedCyclesIn(graph))
	}

	order := make([]component.ID, len(orderedNodes))
	for i, n := range orderedNodes {
		order[i] = n.(*node).extID
	}
	return order, nil
}

func cycleErr(err error, cycles [][]graph.Node) error {
	var cycleStr = ""
	for _, cycle := range cycles {
		var names []string
		for _, n := range cycle {
			node := n.(*node)
			names = append(names, node.extID.String())
		}
		cycleStr = "[" + strings.Join(names, " -> ") + "]"
		//lint:ignore SA4004 only report the first cycle
		break
	}
	return fmt.Errorf("unable to sort the extenions dependency graph (cycle %s): %w", cycleStr, err)
}
