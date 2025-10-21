// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconnector

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

type mutatingProfilesSink struct {
	*consumertest.ProfilesSink
}

func (mts *mutatingProfilesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestProfilesRouterMultiplexing(t *testing.T) {
	num := 20
	for numIDs := 1; numIDs < num; numIDs++ {
		for numCons := 1; numCons < num; numCons++ {
			for numProfiles := 1; numProfiles < num; numProfiles++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numProfiles),
					fuzzProfiles(numIDs, numCons, numProfiles),
				)
			}
		}
	}
}

func fuzzProfiles(numIDs, numCons, numProfiles int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]pipeline.ID, 0, numCons)
		allCons := make([]xconsumer.Profiles, 0, numCons)
		allConsMap := make(map[pipeline.ID]xconsumer.Profiles)

		// If any consumer is mutating, the router must report mutating
		for i := range numCons {
			allIDs = append(allIDs, pipeline.NewIDWithName(xpipeline.SignalProfiles, "sink_"+strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numProfiles+i)%4 == 0 {
				allCons = append(allCons, &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)})
			} else {
				allCons = append(allCons, new(consumertest.ProfilesSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewProfilesRouter(allConsMap)
		td := testdata.GenerateProfiles(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteProfiles.
		expected := make(map[pipeline.ID]int, numCons)

		for i := range numProfiles {
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
			assert.NoError(t, fanout.ConsumeProfiles(context.Background(), td))

			// Validate expectations for all consumers
			for id := range expected {
				profiles := []pprofile.Profiles{}
				switch con := allConsMap[id].(type) {
				case *consumertest.ProfilesSink:
					profiles = con.AllProfiles()
				case *mutatingProfilesSink:
					profiles = con.AllProfiles()
				}
				assert.Len(t, profiles, expected[id])
				for n := 0; n < len(profiles); n++ {
					assert.Equal(t, td, profiles[n])
				}
			}
		}
	}
}

func TestProfilessRouterConsumer(t *testing.T) {
	ctx := context.Background()
	td := testdata.GenerateProfiles(1)

	fooID := pipeline.NewIDWithName(xpipeline.SignalProfiles, "foo")
	barID := pipeline.NewIDWithName(xpipeline.SignalProfiles, "bar")

	foo := new(consumertest.ProfilesSink)
	bar := new(consumertest.ProfilesSink)
	r := NewProfilesRouter(map[pipeline.ID]xconsumer.Profiles{fooID: foo, barID: bar})

	rcs := r.PipelineIDs()
	assert.Len(t, rcs, 2)
	assert.ElementsMatch(t, []pipeline.ID{fooID, barID}, rcs)

	assert.Empty(t, foo.AllProfiles())
	assert.Empty(t, bar.AllProfiles())

	both, err := r.Consumer(fooID, barID)
	assert.NotNil(t, both)
	assert.NoError(t, err)

	assert.NoError(t, both.ConsumeProfiles(ctx, td))
	assert.Len(t, foo.AllProfiles(), 1)
	assert.Len(t, bar.AllProfiles(), 1)

	fooOnly, err := r.Consumer(fooID)
	assert.NotNil(t, fooOnly)
	assert.NoError(t, err)

	assert.NoError(t, fooOnly.ConsumeProfiles(ctx, td))
	assert.Len(t, foo.AllProfiles(), 2)
	assert.Len(t, bar.AllProfiles(), 1)

	barOnly, err := r.Consumer(barID)
	assert.NotNil(t, barOnly)
	assert.NoError(t, err)

	assert.NoError(t, barOnly.ConsumeProfiles(ctx, td))
	assert.Len(t, foo.AllProfiles(), 2)
	assert.Len(t, bar.AllProfiles(), 2)

	none, err := r.Consumer()
	assert.Nil(t, none)
	require.Error(t, err)

	fake, err := r.Consumer(pipeline.NewIDWithName(xpipeline.SignalProfiles, "fake"))
	assert.Nil(t, fake)
	assert.Error(t, err)
}
