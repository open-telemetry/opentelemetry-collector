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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
)

type mutatingLogsSink struct {
	*consumertest.LogsSink
}

func (mts *mutatingLogsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestLogsRouterMultiplexing(t *testing.T) {
	num := 20
	for numIDs := 1; numIDs < num; numIDs++ {
		for numCons := 1; numCons < num; numCons++ {
			for numLogs := 1; numLogs < num; numLogs++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numLogs),
					fuzzLogs(numIDs, numCons, numLogs),
				)
			}
		}
	}
}

func fuzzLogs(numIDs, numCons, numLogs int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]pipeline.ID, 0, numCons)
		allCons := make([]consumer.Logs, 0, numCons)
		allConsMap := make(map[pipeline.ID]consumer.Logs)

		// If any consumer is mutating, the router must report mutating
		for i := range numCons {
			allIDs = append(allIDs, pipeline.NewIDWithName(pipeline.SignalLogs, "sink_"+strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numLogs+i)%4 == 0 {
				allCons = append(allCons, &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)})
			} else {
				allCons = append(allCons, new(consumertest.LogsSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewLogsRouter(allConsMap)
		ld := testdata.GenerateLogs(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteLogs.
		expected := make(map[pipeline.ID]int, numCons)

		for i := range numLogs {
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
			assert.NoError(t, fanout.ConsumeLogs(context.Background(), ld))

			// Validate expectations for all consumers
			for id := range expected {
				logs := []plog.Logs{}
				switch con := allConsMap[id].(type) {
				case *consumertest.LogsSink:
					logs = con.AllLogs()
				case *mutatingLogsSink:
					logs = con.AllLogs()
				}
				assert.Len(t, logs, expected[id])
				for n := 0; n < len(logs); n++ {
					assert.Equal(t, ld, logs[n])
				}
			}
		}
	}
}

func TestLogsRouterConsumers(t *testing.T) {
	ctx := context.Background()
	ld := testdata.GenerateLogs(1)

	fooID := pipeline.NewIDWithName(pipeline.SignalLogs, "foo")
	barID := pipeline.NewIDWithName(pipeline.SignalLogs, "bar")

	foo := new(consumertest.LogsSink)
	bar := new(consumertest.LogsSink)
	r := NewLogsRouter(map[pipeline.ID]consumer.Logs{fooID: foo, barID: bar})

	rcs := r.PipelineIDs()
	assert.Len(t, rcs, 2)
	assert.ElementsMatch(t, []pipeline.ID{fooID, barID}, rcs)

	assert.Empty(t, foo.AllLogs())
	assert.Empty(t, bar.AllLogs())

	both, err := r.Consumer(fooID, barID)
	assert.NotNil(t, both)
	assert.NoError(t, err)

	assert.NoError(t, both.ConsumeLogs(ctx, ld))
	assert.Len(t, foo.AllLogs(), 1)
	assert.Len(t, bar.AllLogs(), 1)

	fooOnly, err := r.Consumer(fooID)
	assert.NotNil(t, fooOnly)
	assert.NoError(t, err)

	assert.NoError(t, fooOnly.ConsumeLogs(ctx, ld))
	assert.Len(t, foo.AllLogs(), 2)
	assert.Len(t, bar.AllLogs(), 1)

	barOnly, err := r.Consumer(barID)
	assert.NotNil(t, barOnly)
	assert.NoError(t, err)

	assert.NoError(t, barOnly.ConsumeLogs(ctx, ld))
	assert.Len(t, foo.AllLogs(), 2)
	assert.Len(t, bar.AllLogs(), 2)

	none, err := r.Consumer()
	assert.Nil(t, none)
	require.Error(t, err)

	fake, err := r.Consumer(pipeline.NewIDWithName(pipeline.SignalLogs, "fake"))
	assert.Nil(t, fake)
	assert.Error(t, err)
}
