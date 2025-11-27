// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fanoutconsumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfilesNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	tfc := NewProfiles([]xconsumer.Profiles{nop})
	assert.Same(t, nop, tfc)
}

func TestProfilesNotMultiplexingMutating(t *testing.T) {
	p := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	lfc := NewProfiles([]xconsumer.Profiles{p})
	assert.True(t, lfc.Capabilities().MutatesData)
}

func TestProfilesMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.ProfilesSink)
	p2 := new(consumertest.ProfilesSink)
	p3 := new(consumertest.ProfilesSink)

	tfc := NewProfiles([]xconsumer.Profiles{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateProfiles(1)

	for range 2 {
		err := tfc.ConsumeProfiles(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.Equal(t, td, p1.AllProfiles()[0])
	assert.Equal(t, td, p1.AllProfiles()[1])
	assert.Equal(t, td, p1.AllProfiles()[0])
	assert.Equal(t, td, p1.AllProfiles()[1])

	assert.Equal(t, td, p2.AllProfiles()[0])
	assert.Equal(t, td, p2.AllProfiles()[1])
	assert.Equal(t, td, p2.AllProfiles()[0])
	assert.Equal(t, td, p2.AllProfiles()[1])

	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])

	// The data should be marked as read only.
	assert.True(t, td.IsReadOnly())
}

func TestProfilesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	tfc := NewProfiles([]xconsumer.Profiles{p1, p2, p3})
	assert.True(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateProfiles(1)

	for range 2 {
		err := tfc.ConsumeProfiles(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &td, &p1.AllProfiles()[0])
	assert.NotSame(t, &td, &p1.AllProfiles()[1])
	assert.Equal(t, td, p1.AllProfiles()[0])
	assert.Equal(t, td, p1.AllProfiles()[1])

	assert.NotSame(t, &td, &p2.AllProfiles()[0])
	assert.NotSame(t, &td, &p2.AllProfiles()[1])
	assert.Equal(t, td, p2.AllProfiles()[0])
	assert.Equal(t, td, p2.AllProfiles()[1])

	// For this consumer, will receive the initial data.
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])

	// The data should not be marked as read only.
	assert.False(t, td.IsReadOnly())
}

func TestReadOnlyProfilesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	tfc := NewProfiles([]xconsumer.Profiles{p1, p2, p3})
	assert.True(t, tfc.Capabilities().MutatesData)

	tdOrig := testdata.GenerateProfiles(1)
	td := testdata.GenerateProfiles(1)
	td.MarkReadOnly()

	for range 2 {
		err := tfc.ConsumeProfiles(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	// All consumers should receive the cloned data.

	assert.NotEqual(t, td, p1.AllProfiles()[0])
	assert.NotEqual(t, td, p1.AllProfiles()[1])
	assert.Equal(t, tdOrig, p1.AllProfiles()[0])
	assert.Equal(t, tdOrig, p1.AllProfiles()[1])

	assert.NotEqual(t, td, p2.AllProfiles()[0])
	assert.NotEqual(t, td, p2.AllProfiles()[1])
	assert.Equal(t, tdOrig, p2.AllProfiles()[0])
	assert.Equal(t, tdOrig, p2.AllProfiles()[1])

	assert.NotEqual(t, td, p3.AllProfiles()[0])
	assert.NotEqual(t, td, p3.AllProfiles()[1])
	assert.Equal(t, tdOrig, p3.AllProfiles()[0])
	assert.Equal(t, tdOrig, p3.AllProfiles()[1])
}

func TestProfilesMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := new(consumertest.ProfilesSink)
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	tfc := NewProfiles([]xconsumer.Profiles{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateProfiles(1)

	for range 2 {
		err := tfc.ConsumeProfiles(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &td, &p1.AllProfiles()[0])
	assert.NotSame(t, &td, &p1.AllProfiles()[1])
	assert.Equal(t, td, p1.AllProfiles()[0])
	assert.Equal(t, td, p1.AllProfiles()[1])

	// For this consumer, will receive the initial data.
	assert.Equal(t, td, p2.AllProfiles()[0])
	assert.Equal(t, td, p2.AllProfiles()[1])
	assert.Equal(t, td, p2.AllProfiles()[0])
	assert.Equal(t, td, p2.AllProfiles()[1])

	// For this consumer, will clone the initial data.
	assert.NotSame(t, &td, &p3.AllProfiles()[0])
	assert.NotSame(t, &td, &p3.AllProfiles()[1])
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])

	// The data should not be marked as read only.
	assert.False(t, td.IsReadOnly())
}

func TestProfilesMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := new(consumertest.ProfilesSink)

	tfc := NewProfiles([]xconsumer.Profiles{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateProfiles(1)

	for range 2 {
		err := tfc.ConsumeProfiles(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &td, &p1.AllProfiles()[0])
	assert.NotSame(t, &td, &p1.AllProfiles()[1])
	assert.Equal(t, td, p1.AllProfiles()[0])
	assert.Equal(t, td, p1.AllProfiles()[1])

	assert.NotSame(t, &td, &p2.AllProfiles()[0])
	assert.NotSame(t, &td, &p2.AllProfiles()[1])
	assert.Equal(t, td, p2.AllProfiles()[0])
	assert.Equal(t, td, p2.AllProfiles()[1])

	// For this consumer, will receive the initial data.
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])

	// The data should not be marked as read only.
	assert.False(t, td.IsReadOnly())
}

func TestProfilesWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.ProfilesSink)

	tfc := NewProfiles([]xconsumer.Profiles{p1, p2, p3})
	td := testdata.GenerateProfiles(1)

	for range 2 {
		require.Error(t, tfc.ConsumeProfiles(context.Background(), td))
	}

	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])
	assert.Equal(t, td, p3.AllProfiles()[0])
	assert.Equal(t, td, p3.AllProfiles()[1])
}

type mutatingProfilesSink struct {
	*consumertest.ProfilesSink
}

func (mts *mutatingProfilesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
