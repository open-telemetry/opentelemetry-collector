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
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func TestTracesNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	tfc := NewTraces([]consumer.Traces{nop})
	assert.Same(t, nop, tfc)
}

func TestTracesNotMultiplexingMutating(t *testing.T) {
	p := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	lfc := NewTraces([]consumer.Traces{p})
	assert.True(t, lfc.Capabilities().MutatesData)
}

func TestTracesMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.TracesSink)
	p2 := new(consumertest.TracesSink)
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for range 2 {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.Equal(t, td, p1.AllTraces()[0])
	assert.Equal(t, td, p1.AllTraces()[1])
	assert.Equal(t, td, p1.AllTraces()[0])
	assert.Equal(t, td, p1.AllTraces()[1])

	assert.Equal(t, td, p2.AllTraces()[0])
	assert.Equal(t, td, p2.AllTraces()[1])
	assert.Equal(t, td, p2.AllTraces()[0])
	assert.Equal(t, td, p2.AllTraces()[1])

	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])
	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])

	// The data should be marked as read only.
	assert.True(t, td.IsReadOnly())
}

func TestTracesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.True(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for range 2 {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &td, &p1.AllTraces()[0])
	assert.NotSame(t, &td, &p1.AllTraces()[1])
	assert.True(t, pref.EqualTraces(td, p1.AllTraces()[0]))
	assert.True(t, pref.EqualTraces(td, p1.AllTraces()[1]))

	assert.NotSame(t, &td, &p2.AllTraces()[0])
	assert.NotSame(t, &td, &p2.AllTraces()[1])
	assert.True(t, pref.EqualTraces(td, p2.AllTraces()[0]))
	assert.True(t, pref.EqualTraces(td, p2.AllTraces()[1]))

	// For this consumer, will receive the initial data.
	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])
	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])

	// The data should not be marked as read only.
	assert.False(t, td.IsReadOnly())
}

func TestReadOnlyTracesMultiplexingMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.True(t, tfc.Capabilities().MutatesData)

	tdOrig := testdata.GenerateTraces(1)
	td := testdata.GenerateTraces(1)
	td.MarkReadOnly()

	for range 2 {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	// All consumers should receive the cloned data.

	assert.NotEqual(t, td, p1.AllTraces()[0])
	assert.NotEqual(t, td, p1.AllTraces()[1])
	assert.Equal(t, tdOrig, p1.AllTraces()[0])
	assert.Equal(t, tdOrig, p1.AllTraces()[1])

	assert.NotEqual(t, td, p2.AllTraces()[0])
	assert.NotEqual(t, td, p2.AllTraces()[1])
	assert.Equal(t, tdOrig, p2.AllTraces()[0])
	assert.Equal(t, tdOrig, p2.AllTraces()[1])

	assert.NotEqual(t, td, p3.AllTraces()[0])
	assert.NotEqual(t, td, p3.AllTraces()[1])
	assert.Equal(t, tdOrig, p3.AllTraces()[0])
	assert.Equal(t, tdOrig, p3.AllTraces()[1])
}

func TestTracesMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := new(consumertest.TracesSink)
	p3 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for range 2 {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &td, &p1.AllTraces()[0])
	assert.NotSame(t, &td, &p1.AllTraces()[1])
	assert.True(t, pref.EqualTraces(td, p1.AllTraces()[0]))
	assert.True(t, pref.EqualTraces(td, p1.AllTraces()[1]))

	// For this consumer, will receive the initial data.
	assert.Equal(t, td, p2.AllTraces()[0])
	assert.Equal(t, td, p2.AllTraces()[1])
	assert.Equal(t, td, p2.AllTraces()[0])
	assert.Equal(t, td, p2.AllTraces()[1])

	// For this consumer, will clone the initial data.
	assert.NotSame(t, &td, &p3.AllTraces()[0])
	assert.NotSame(t, &td, &p3.AllTraces()[1])
	assert.True(t, pref.EqualTraces(td, p3.AllTraces()[0]))
	assert.True(t, pref.EqualTraces(td, p3.AllTraces()[1]))

	// The data should not be marked as read only.
	assert.False(t, td.IsReadOnly())
}

func TestTracesMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p2 := &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)}
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	assert.False(t, tfc.Capabilities().MutatesData)
	td := testdata.GenerateTraces(1)

	for range 2 {
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.NotSame(t, &td, &p1.AllTraces()[0])
	assert.NotSame(t, &td, &p1.AllTraces()[1])
	assert.True(t, pref.EqualTraces(td, p1.AllTraces()[0]))
	assert.True(t, pref.EqualTraces(td, p1.AllTraces()[1]))

	assert.NotSame(t, &td, &p2.AllTraces()[0])
	assert.NotSame(t, &td, &p2.AllTraces()[1])
	assert.True(t, pref.EqualTraces(td, p2.AllTraces()[0]))
	assert.True(t, pref.EqualTraces(td, p2.AllTraces()[1]))

	// For this consumer, will receive the initial data.
	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])
	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])

	// The data should not be marked as read only.
	assert.False(t, td.IsReadOnly())
}

func TestTracesMultiplexingRefCount(t *testing.T) {
	enableProtoPoolingForTest(t)

	// Each consumer acts like a pipeline: it takes ownership of the data if no one did yet and
	// releases it when it returns.
	var counts []int
	newConsumer := func() consumer.Traces {
		c, err := consumer.NewTraces(func(_ context.Context, td ptrace.Traces) error {
			if pref.MarkPipelineOwnedTraces(td) {
				defer pref.UnrefTraces(td)
			}
			counts = append(counts, td.SpanCount())
			return nil
		})
		require.NoError(t, err)
		return c
	}
	fc := NewTraces([]consumer.Traces{newConsumer(), newConsumer()})

	t.Run("not_owned", func(t *testing.T) {
		counts = nil
		td := testdata.GenerateTraces(2)
		require.NoError(t, fc.ConsumeTraces(context.Background(), td))
		assert.Equal(t, []int{2, 2}, counts)
		// The fanout consumer owned the data and released it after all consumers received it.
		assert.Equal(t, 0, td.SpanCount())
	})

	t.Run("owned_upstream", func(t *testing.T) {
		counts = nil
		td := testdata.GenerateTraces(2)
		require.True(t, pref.MarkPipelineOwnedTraces(td))
		require.NoError(t, fc.ConsumeTraces(context.Background(), td))
		assert.Equal(t, []int{2, 2}, counts)
		// Releasing the data is left to the upstream owner.
		assert.Equal(t, 2, td.SpanCount())
		pref.UnrefTraces(td)
	})
}

func TestTracesWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.TracesSink)

	tfc := NewTraces([]consumer.Traces{p1, p2, p3})
	td := testdata.GenerateTraces(1)

	for range 2 {
		require.Error(t, tfc.ConsumeTraces(context.Background(), td))
	}

	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])
	assert.Equal(t, td, p3.AllTraces()[0])
	assert.Equal(t, td, p3.AllTraces()[1])
}

type mutatingTracesSink struct {
	*consumertest.TracesSink
}

func (mts *mutatingTracesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
