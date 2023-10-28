package extensions

import (
	"hash/fnv"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"

	"go.opentelemetry.io/collector/component"
)

type dependencyGraph struct {
	graph *simple.DirectedGraph
	order []component.ID
}

type nodeID int64

func (n nodeID) ID() int64 {
	return int64(n)
}

func newNodeID(extID component.ID) nodeID {
	h := fnv.New64a()
	_, _ = h.Write([]byte(extID.String()))
	return nodeID(h.Sum64())
}

type node struct {
	nodeID
	extID component.ID
}

func newNode(extID component.ID) *node {
	return &node{
		nodeID: newNodeID(extID),
		extID:  extID,
	}
}

func newDependencyGraph() *dependencyGraph {
	return &dependencyGraph{
		graph: simple.NewDirectedGraph(),
	}
}

func (dg *dependencyGraph) addNode(extID component.ID) {
	dg.graph.AddNode(newNode(extID))
}

func (dg *dependencyGraph) addDependency(fromExtID, toExtID component.ID) {
	dg.graph.SetEdge(dg.graph.NewEdge(newNodeID(fromExtID), newNodeID(toExtID)))
}

func (dg *dependencyGraph) sort() ([]component.ID, error) {
	nodes, err := topo.Sort(dg.graph)
	if err != nil {
		return nil, err
	}
	order := make([]component.ID, len(nodes))
	for i, n := range nodes {
		order[i] = n.(*node).extID
	}
	return order, nil
}
