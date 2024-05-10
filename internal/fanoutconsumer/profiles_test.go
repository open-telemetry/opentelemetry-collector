// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fanoutconsumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestProfilesNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	lfc := NewProfiles([]consumer.Profiles{nop})
	assert.Same(t, nop, lfc)
}

func TestProfilesNotMultiplexingMutating(t *testing.T) {
	p := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	lfc := NewProfiles([]consumer.Profiles{p})
	assert.True(t, lfc.Capabilities().MutatesData)
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

	// The data should be marked as read only.
	assert.True(t, ld.IsReadOnly())
}

func TestProfilesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	assert.True(t, lfc.Capabilities().MutatesData)
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

	// The data should not be marked as read only.
	assert.False(t, ld.IsReadOnly())
}

func TestReadOnlyProfilesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p2 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}
	p3 := &mutatingProfilesSink{ProfilesSink: new(consumertest.ProfilesSink)}

	lfc := NewProfiles([]consumer.Profiles{p1, p2, p3})
	assert.True(t, lfc.Capabilities().MutatesData)
	ldOrig := testdata.GenerateProfiles(1)
	ld := testdata.GenerateProfiles(1)
	ld.MarkReadOnly()

	for i := 0; i < 2; i++ {
		err := lfc.ConsumeProfiles(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	// All consumers should receive the cloned data.

	assert.True(t, ld != p1.AllProfiles()[0])
	assert.True(t, ld != p1.AllProfiles()[1])
	assert.EqualValues(t, ldOrig, p1.AllProfiles()[0])
	assert.EqualValues(t, ldOrig, p1.AllProfiles()[1])

	assert.True(t, ld != p2.AllProfiles()[0])
	assert.True(t, ld != p2.AllProfiles()[1])
	assert.EqualValues(t, ldOrig, p2.AllProfiles()[0])
	assert.EqualValues(t, ldOrig, p2.AllProfiles()[1])

	assert.True(t, ld != p3.AllProfiles()[0])
	assert.True(t, ld != p3.AllProfiles()[1])
	assert.EqualValues(t, ldOrig, p3.AllProfiles()[0])
	assert.EqualValues(t, ldOrig, p3.AllProfiles()[1])
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

	// The data should not be marked as read only.
	assert.False(t, ld.IsReadOnly())
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

	// The data should not be marked as read only.
	assert.False(t, ld.IsReadOnly())
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
