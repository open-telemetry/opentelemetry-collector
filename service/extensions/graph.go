// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"errors"
	"fmt"
	"strings"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
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
		if dep, ok := ext.(extensioncapabilities.Dependent); ok {
			for _, depID := range dep.Dependencies() {
				d, ok := nodes[depID]
				if !ok {
					return nil, fmt.Errorf("unable to find extension %s on which extension %s depends", depID, extID)
				}
				graph.SetEdge(graph.NewEdge(d, n))
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
	var topoErr topo.Unorderable
	if !errors.As(err, &topoErr) || len(cycles) == 0 || len(cycles[0]) == 0 {
		return err
	}

	cycle := cycles[0]
	var names []string
	for _, n := range cycle {
		node := n.(*node)
		names = append(names, node.extID.String())
	}
	cycleStr := "[" + strings.Join(names, " -> ") + "]"
	return fmt.Errorf("unable to order extensions by dependencies, cycle found %s: %w", cycleStr, err)
}
