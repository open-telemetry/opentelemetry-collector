// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestProfilesNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	lfc := NewProfiles([]consumer.Profiles{nop})
	assert.Same(t, nop, lfc)
}

func TestProfilesMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.ProfilesSink)
	p2 := new(consumertest.ProfilesSink)
	p3 := new(consumertest.ProfilesSink)

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateProfiles(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeProfiles(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld == p1.AllProfiles()[0])
	assert.True(t, ld == p1.AllProfiles()[1])
	assert.EqualValues(t, ld, p1.AllProfiles()[0])
	assert.EqualValues(t, ld, p1.AllProfiles()[1])

	assert.True(t, ld == p2.AllProfiles()[0])
	assert.True(t, ld == p2.AllProfiles()[1])
	assert.EqualValues(t, ld, p2.AllProfiles()[0])
	assert.EqualValues(t, ld, p2.AllProfiles()[1])

	assert.True(t, ld == p3.AllProfiles()[0])
	assert.True(t, ld == p3.AllProfiles()[1])
	assert.EqualValues(t, ld, p3.AllProfiles()[0])
	assert.EqualValues(t, ld, p3.AllProfiles()[1])
}

func TestProfilesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateProfiles(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeProfiles(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld != p1.AllProfiles()[0])
	assert.True(t, ld != p1.AllProfiles()[1])
	assert.EqualValues(t, ld, p1.AllProfiles()[0])
	assert.EqualValues(t, ld, p1.AllProfiles()[1])

	assert.True(t, ld != p2.AllProfiles()[0])
	assert.True(t, ld != p2.AllProfiles()[1])
	assert.EqualValues(t, ld, p2.AllProfiles()[0])
	assert.EqualValues(t, ld, p2.AllProfiles()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld == p3.AllProfiles()[0])
	assert.True(t, ld == p3.AllProfiles()[1])
	assert.EqualValues(t, ld, p3.AllProfiles()[0])
	assert.EqualValues(t, ld, p3.AllProfiles()[1])
}

func TestProfilesMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := new(consumertest.ProfilesSink)
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateProfiles(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeProfiles(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld != p1.AllProfiles()[0])
	assert.True(t, ld != p1.AllProfiles()[1])
	assert.EqualValues(t, ld, p1.AllProfiles()[0])
	assert.EqualValues(t, ld, p1.AllProfiles()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld == p2.AllProfiles()[0])
	assert.True(t, ld == p2.AllProfiles()[1])
	assert.EqualValues(t, ld, p2.AllProfiles()[0])
	assert.EqualValues(t, ld, p2.AllProfiles()[1])

	// For this consumer, will clone the initial data.
	assert.True(t, ld != p3.AllProfiles()[0])
	assert.True(t, ld != p3.AllProfiles()[1])
	assert.EqualValues(t, ld, p3.AllProfiles()[0])
	assert.EqualValues(t, ld, p3.AllProfiles()[1])
}

func TestProfilesMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := new(consumertest.ProfilesSink)

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	assert.False(t, lfc.Capabilities().MutatesData)
	ld := testdata.GenerateProfiles(1)

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeProfiles(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, ld != p1.AllProfiles()[0])
	assert.True(t, ld != p1.AllProfiles()[1])
	assert.EqualValues(t, ld, p1.AllProfiles()[0])
	assert.EqualValues(t, ld, p1.AllProfiles()[1])

	assert.True(t, ld != p2.AllProfiles()[0])
	assert.True(t, ld != p2.AllProfiles()[1])
	assert.EqualValues(t, ld, p2.AllProfiles()[0])
	assert.EqualValues(t, ld, p2.AllProfiles()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, ld == p3.AllProfiles()[0])
	assert.True(t, ld == p3.AllProfiles()[1])
	assert.EqualValues(t, ld, p3.AllProfiles()[0])
	assert.EqualValues(t, ld, p3.AllProfiles()[1])
}

func TestProfilesWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.ProfilesSink)

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	ld := testdata.GenerateProfiles(1)

	for i := 0; i < 2; i++ {
		assert.Error(t, lfc.ConsumeProfiles(context.Background(), ld))
	}

	assert.True(t, ld == p3.AllProfiles()[0])
	assert.True(t, ld == p3.AllProfiles()[1])
	assert.EqualValues(t, ld, p3.AllProfiles()[0])
	assert.EqualValues(t, ld, p3.AllProfiles()[1])
}

type mutatingProfilesSink struct {
	*consumertest.ProfilesSink
}

func (mts *mutatingProfilesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

type mutatingErr struct {
	consumertest.Consumer
}

func (mts mutatingErr) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestProfilesRouterMultiplexing(t *testing.T) {
	var max = 20
	for numIDs := 1; numIDs < max; numIDs++ {
		for numCons := 1; numCons < max; numCons++ {
			for numProfiles := 1; numProfiles < max; numProfiles++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-profiles", numIDs, numCons, numProfiles),
					fuzzProfilesRouter(numIDs, numCons, numProfiles),
				)
			}
		}
	}
}

func fuzzProfilesRouter(numIDs, numCons, numProfiles int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]component.ID, 0, numCons)
		allCons := make([]consumer.Profiles, 0, numCons)
		allConsMap := make(map[component.ID]consumer.Profiles)

		// If any consumer is mutating, the router must report mutating
		for i := 0; i < numCons; i++ {
			allIDs = append(allIDs, component.NewIDWithName("sink", strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numProfiles+i)%4 == 0 {
				allCons = append(allCons, &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)})
			} else {
				allCons = append(allCons, new(consumertest.ProfilesSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewProfilesRouter(allConsMap).(connector.ProfilesRouter)
		ld := testdata.GenerateProfiles(1)

		// Keep track of how many profiles each consumer should receive.
		// This will be validated after every call to RouteProfiles.
		expected := make(map[component.ID]int, numCons)

		for i := 0; i < numProfiles; i++ {
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
			assert.NoError(t, fanout.ConsumeProfiles(context.Background(), ld))

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
					assert.EqualValues(t, ld, profiles[n])
				}
			}
		}
	}
}

func TestProfilesRouterGetConsumers(t *testing.T) {
	ctx := context.Background()
	ld := testdata.GenerateProfiles(1)

	fooID := component.NewID("foo")
	barID := component.NewID("bar")

	foo := new(consumertest.ProfilesSink)
	bar := new(consumertest.ProfilesSink)
	r := NewProfilesRouter(map[component.ID]consumer.Profiles{fooID: foo, barID: bar}).(connector.ProfilesRouter)

	rcs := r.PipelineIDs()
	assert.Len(t, rcs, 2)
	assert.ElementsMatch(t, []component.ID{fooID, barID}, rcs)

	assert.Len(t, foo.AllProfiles(), 0)
	assert.Len(t, bar.AllProfiles(), 0)

	both, err := r.Consumer(fooID, barID)
	assert.NotNil(t, both)
	assert.NoError(t, err)

	assert.NoError(t, both.ConsumeProfiles(ctx, ld))
	assert.Len(t, foo.AllProfiles(), 1)
	assert.Len(t, bar.AllProfiles(), 1)

	fooOnly, err := r.Consumer(fooID)
	assert.NotNil(t, fooOnly)
	assert.NoError(t, err)

	assert.NoError(t, fooOnly.ConsumeProfiles(ctx, ld))
	assert.Len(t, foo.AllProfiles(), 2)
	assert.Len(t, bar.AllProfiles(), 1)

	barOnly, err := r.Consumer(barID)
	assert.NotNil(t, barOnly)
	assert.NoError(t, err)

	assert.NoError(t, barOnly.ConsumeProfiles(ctx, ld))
	assert.Len(t, foo.AllProfiles(), 2)
	assert.Len(t, bar.AllProfiles(), 2)

	none, err := r.Consumer()
	assert.Nil(t, none)
	assert.Error(t, err)

	fake, err := r.Consumer(component.NewID("fake"))
	assert.Nil(t, fake)
	assert.Error(t, err)
}
