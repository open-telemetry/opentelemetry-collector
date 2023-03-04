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

package fanoutconsumer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestTracesNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	tfc := NewTraces([]consumer.Traces{nop})
	assert.Same(t, nop, tfc)
}

func TestTracesMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.TracesSink)
	p2 := new(consumertest.TracesSink)
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td == p1.AllTraces()[0])
	assert.True(t, td == p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	assert.True(t, td == p2.AllTraces()[0])
	assert.True(t, td == p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td != p1.AllTraces()[0])
	assert.True(t, td != p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	assert.True(t, td != p2.AllTraces()[0])
	assert.True(t, td != p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := new(consumertest.TracesSink)
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td != p1.AllTraces()[0])
	assert.True(t, td != p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, td == p2.AllTraces()[0])
	assert.True(t, td == p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	// For this consumer, will clone the initial data.
	assert.True(t, td != p3.AllTraces()[0])
	assert.True(t, td != p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, td != p1.AllTraces()[0])
	assert.True(t, td != p1.AllTraces()[1])
	assert.EqualValues(t, td, p1.AllTraces()[0])
	assert.EqualValues(t, td, p1.AllTraces()[1])

	assert.True(t, td != p2.AllTraces()[0])
	assert.True(t, td != p2.AllTraces()[1])
	assert.EqualValues(t, td, p2.AllTraces()[0])
	assert.EqualValues(t, td, p2.AllTraces()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

func TestTracesWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	td := testdata.GenerateTraces(1)

	for i := 0; i < 2; i++ {
		assert.Error(t, tfc.ConsumeTraces(context.Background(), td))
	}

	assert.True(t, td == p3.AllTraces()[0])
	assert.True(t, td == p3.AllTraces()[1])
	assert.EqualValues(t, td, p3.AllTraces()[0])
	assert.EqualValues(t, td, p3.AllTraces()[1])
}

type mutatingTracesSink struct {
	*consumertest.TracesSink
}

func (mts *mutatingTracesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestTracesRouterMultiplexing(t *testing.T) {
	var max = 20
	for numIDs := 1; numIDs < max; numIDs++ {
		for numCons := 1; numCons < max; numCons++ {
			for numTraces := 1; numTraces < max; numTraces++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numTraces),
					fuzzTracesRouter(numIDs, numCons, numTraces),
				)
			}
		}
	}
}

func fuzzTracesRouter(numIDs, numCons, numTraces int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]component.ID, 0, numCons)
		allCons := make([]consumer.Traces, 0, numCons)
		allConsMap := make(map[component.ID]consumer.Traces)

		// If any consumer is mutating, the router must report mutating
		for i := 0; i < numCons; i++ {
			allIDs = append(allIDs, component.NewIDWithName("sink", strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numTraces+i)%4 == 0 {
				allCons = append(allCons, &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)})
			} else {
				allCons = append(allCons, new(consumertest.TracesSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewTracesRouter(allConsMap).(connector.TracesRouter)
		td := testdata.GenerateTraces(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteTraces.
		expected := make(map[component.ID]int, numCons)

		for i := 0; i < numTraces; i++ {
			// Build a random set of ids (no duplicates)
			randCons := make(map[component.ID]bool, numIDs)
			for j := 0; j < numIDs; j++ {
				// This number should be pretty random and less than numCons
				conNum := (numCons + numIDs + i + j) % numCons
				randCons[allIDs[conNum]] = true
			}

			// Convert to slice, update expectations
			conIDs := make([]component.ID, 0, len(randCons))
			for id := range randCons {
				conIDs = append(conIDs, id)
				expected[id]++
			}

			// Route to list of consumers
			fanout, err := r.Consumer(conIDs...)
			assert.NoError(t, err)
			assert.NoError(t, fanout.ConsumeTraces(context.Background(), td))

			// Validate expectations for all consumers
			for id := range expected {
				traces := []ptrace.Traces{}
				switch con := allConsMap[id].(type) {
				case *consumertest.TracesSink:
					traces = con.AllTraces()
				case *mutatingTracesSink:
					traces = con.AllTraces()
				}
				assert.Len(t, traces, expected[id])
				for n := 0; n < len(traces); n++ {
					assert.EqualValues(t, td, traces[n])
				}
			}
		}
	}
}

func TestTracesRouterGetConsumer(t *testing.T) {
	ctx := context.Background()
	td := testdata.GenerateTraces(1)

	fooID := component.NewID("foo")
	barID := component.NewID("bar")

	foo := new(consumertest.TracesSink)
	bar := new(consumertest.TracesSink)
	r := NewTracesRouter(map[component.ID]consumer.Traces{fooID: foo, barID: bar}).(connector.TracesRouter)

	rcs := r.PipelineIDs()
	assert.Len(t, rcs, 2)
	assert.ElementsMatch(t, []component.ID{fooID, barID}, rcs)

	assert.Len(t, foo.AllTraces(), 0)
	assert.Len(t, bar.AllTraces(), 0)

	both, err := r.Consumer(fooID, barID)
	assert.NotNil(t, both)
	assert.NoError(t, err)

	assert.NoError(t, both.ConsumeTraces(ctx, td))
	assert.Len(t, foo.AllTraces(), 1)
	assert.Len(t, bar.AllTraces(), 1)

	fooOnly, err := r.Consumer(fooID)
	assert.NotNil(t, fooOnly)
	assert.NoError(t, err)

	assert.NoError(t, fooOnly.ConsumeTraces(ctx, td))
	assert.Len(t, foo.AllTraces(), 2)
	assert.Len(t, bar.AllTraces(), 1)

	barOnly, err := r.Consumer(barID)
	assert.NotNil(t, barOnly)
	assert.NoError(t, err)

	assert.NoError(t, barOnly.ConsumeTraces(ctx, td))
	assert.Len(t, foo.AllTraces(), 2)
	assert.Len(t, bar.AllTraces(), 2)

	none, err := r.Consumer()
	assert.Nil(t, none)
	assert.Error(t, err)

	fake, err := r.Consumer(component.NewID("fake"))
	assert.Nil(t, fake)
	assert.Error(t, err)
}
