// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connector

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
)

type mutatingTracesSink struct {
	*consumertest.TracesSink
}

func (mts *mutatingTracesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestTracesRouterMultiplexing(t *testing.T) {
	num := 20
	for numIDs := 1; numIDs < num; numIDs++ {
		for numCons := 1; numCons < num; numCons++ {
			for numTraces := 1; numTraces < num; numTraces++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numTraces),
					fuzzTraces(numIDs, numCons, numTraces),
				)
			}
		}
	}
}

func fuzzTraces(numIDs, numCons, numTraces int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]pipeline.ID, 0, numCons)
		allCons := make([]consumer.Traces, 0, numCons)
		allConsMap := make(map[pipeline.ID]consumer.Traces)

		// If any consumer is mutating, the router must report mutating
		for i := range numCons {
			allIDs = append(allIDs, pipeline.NewIDWithName(pipeline.SignalTraces, "sink_"+strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numTraces+i)%4 == 0 {
				allCons = append(allCons, &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)})
			} else {
				allCons = append(allCons, new(consumertest.TracesSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewTracesRouter(allConsMap)
		td := testdata.GenerateTraces(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteTraces.
		expected := make(map[pipeline.ID]int, numCons)

		for i := range numTraces {
			// Build a random set of ids (no duplicates)
			randCons := make(map[pipeline.ID]bool, numIDs)
			for j := range numIDs {
				// This number should be pretty random and less than numCons
				conNum := (numCons + numIDs + i + j) % numCons
				randCons[allIDs[conNum]] = true
			}

			// Convert to slice, update expectations
			conIDs := make([]pipeline.ID, 0, len(randCons))
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
					assert.Equal(t, td, traces[n])
				}
			}
		}
	}
}

func TestTracesRouterConsumer(t *testing.T) {
	ctx := context.Background()
	td := testdata.GenerateTraces(1)

	fooID := pipeline.NewIDWithName(pipeline.SignalTraces, "foo")
	barID := pipeline.NewIDWithName(pipeline.SignalTraces, "bar")

	foo := new(consumertest.TracesSink)
	bar := new(consumertest.TracesSink)
	r := NewTracesRouter(map[pipeline.ID]consumer.Traces{fooID: foo, barID: bar})

	rcs := r.PipelineIDs()
	assert.Len(t, rcs, 2)
	assert.ElementsMatch(t, []pipeline.ID{fooID, barID}, rcs)

	assert.Empty(t, foo.AllTraces())
	assert.Empty(t, bar.AllTraces())

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
	require.Error(t, err)

	fake, err := r.Consumer(pipeline.NewIDWithName(pipeline.SignalTraces, "fake"))
	assert.Nil(t, fake)
	assert.Error(t, err)
}
