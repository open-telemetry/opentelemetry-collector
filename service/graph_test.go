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

package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph/simple"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

var _ component.Component = &testNode{}

type testNode struct {
	id          component.ID
	startErr    error
	shutdownErr error
}

// ID satisfies the graph.Node interface, allowing
// testNode to be used in a simple.DirectedGraph
func (n *testNode) ID() int64 {
	return int64(newNodeID(n.id.String()))
}

func (n *testNode) Start(ctx context.Context, _ component.Host) error {
	if n.startErr != nil {
		return n.startErr
	}
	if cwo, ok := ctx.(*contextWithOrder); ok {
		cwo.record(n.id)
	}
	return nil
}

func (n *testNode) Shutdown(ctx context.Context) error {
	if n.shutdownErr != nil {
		return n.shutdownErr
	}
	if cwo, ok := ctx.(*contextWithOrder); ok {
		cwo.record(n.id)
	}
	return nil
}

type contextWithOrder struct {
	context.Context
	sync.Mutex
	next  int
	order map[component.ID]int
}

func (c *contextWithOrder) record(id component.ID) {
	c.Lock()
	c.order[id] = c.next
	c.next++
	c.Unlock()
}

func TestGraphStartStop(t *testing.T) {
	testCases := []struct {
		name  string
		edges [][2]component.ID
	}{
		{
			name: "single",
			edges: [][2]component.ID{
				{component.NewIDWithName("r", "1"), component.NewIDWithName("p", "1")},
				{component.NewIDWithName("r", "2"), component.NewIDWithName("p", "1")},
				{component.NewIDWithName("p", "1"), component.NewIDWithName("p", "2")},
				{component.NewIDWithName("p", "2"), component.NewIDWithName("e", "1")},
				{component.NewIDWithName("p", "1"), component.NewIDWithName("e", "2")},
			},
		},
		{
			name: "multi",
			edges: [][2]component.ID{
				// Pipeline 1
				{component.NewIDWithName("r", "1"), component.NewIDWithName("p", "1")},
				{component.NewIDWithName("r", "2"), component.NewIDWithName("p", "1")},
				{component.NewIDWithName("p", "1"), component.NewIDWithName("p", "2")},
				{component.NewIDWithName("p", "2"), component.NewIDWithName("e", "1")},
				{component.NewIDWithName("p", "1"), component.NewIDWithName("e", "2")},

				// Pipeline 2, shares r1 and e2
				{component.NewIDWithName("r", "1"), component.NewIDWithName("p", "3")},
				{component.NewIDWithName("p", "3"), component.NewIDWithName("e", "2")},
			},
		},
		{
			name: "connected",
			edges: [][2]component.ID{
				// Pipeline 1
				{component.NewIDWithName("r", "1"), component.NewIDWithName("p", "1")},
				{component.NewIDWithName("r", "2"), component.NewIDWithName("p", "1")},
				{component.NewIDWithName("p", "1"), component.NewIDWithName("p", "2")},
				{component.NewIDWithName("p", "2"), component.NewIDWithName("e", "1")},
				{component.NewIDWithName("p", "1"), component.NewIDWithName("c", "1")},

				// Pipeline 2, shares r1 and c1
				{component.NewIDWithName("r", "1"), component.NewIDWithName("p", "3")},
				{component.NewIDWithName("p", "3"), component.NewIDWithName("c", "1")},

				// Pipeline 3, emits to e2 and c2
				{component.NewIDWithName("c", "1"), component.NewIDWithName("e", "2")},
				{component.NewIDWithName("c", "1"), component.NewIDWithName("c", "2")},

				// Pipeline 4, also emits to e2
				{component.NewIDWithName("c", "2"), component.NewIDWithName("e", "2")},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &contextWithOrder{
				Context: context.Background(),
				order:   map[component.ID]int{},
			}

			pg := &pipelinesGraph{componentGraph: simple.NewDirectedGraph()}
			for _, edge := range tt.edges {
				f, t := &testNode{id: edge[0]}, &testNode{id: edge[1]}
				pg.componentGraph.SetEdge(simple.Edge{F: f, T: t})
			}

			require.NoError(t, pg.StartAll(ctx, componenttest.NewNopHost()))
			for _, edge := range tt.edges {
				assert.Greater(t, ctx.order[edge[0]], ctx.order[edge[1]])
			}

			ctx.order = map[component.ID]int{}
			require.NoError(t, pg.ShutdownAll(ctx))
			for _, edge := range tt.edges {
				assert.Less(t, ctx.order[edge[0]], ctx.order[edge[1]])
			}
		})
	}
}

func TestGraphStartStopCycle(t *testing.T) {
	pg := &pipelinesGraph{componentGraph: simple.NewDirectedGraph()}

	r1 := &testNode{id: component.NewIDWithName("r", "1")}
	p1 := &testNode{id: component.NewIDWithName("p", "1")}
	c1 := &testNode{id: component.NewIDWithName("c", "1")}
	e1 := &testNode{id: component.NewIDWithName("e", "1")}

	pg.componentGraph.SetEdge(simple.Edge{F: r1, T: p1})
	pg.componentGraph.SetEdge(simple.Edge{F: p1, T: c1})
	pg.componentGraph.SetEdge(simple.Edge{F: c1, T: e1})
	pg.componentGraph.SetEdge(simple.Edge{F: c1, T: p1}) // loop back

	err := pg.StartAll(context.Background(), componenttest.NewNopHost())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `topo: no topological ordering: cyclic components`)

	err = pg.ShutdownAll(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `topo: no topological ordering: cyclic components`)
}

func TestGraphStartStopComponentError(t *testing.T) {
	pg := &pipelinesGraph{componentGraph: simple.NewDirectedGraph()}
	pg.componentGraph.SetEdge(simple.Edge{
		F: &testNode{
			id:       component.NewIDWithName("r", "1"),
			startErr: errors.New("foo"),
		},
		T: &testNode{
			id:          component.NewIDWithName("e", "1"),
			shutdownErr: errors.New("bar"),
		},
	})
	assert.EqualError(t, pg.StartAll(context.Background(), componenttest.NewNopHost()), "foo")
	assert.EqualError(t, pg.ShutdownAll(context.Background()), "bar")
}
