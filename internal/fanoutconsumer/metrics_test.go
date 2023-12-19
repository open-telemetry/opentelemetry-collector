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
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestMetricsNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	mfc := NewMetrics([]consumer.Metrics{nop})
	assert.Same(t, nop, mfc)
}

func TestMetricssNotMultiplexingMutating(t *testing.T) {
	p := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	lfc := NewMetrics([]consumer.Metrics{p})
	assert.True(t, lfc.Capabilities().MutatesData)
}

func TestMetricsMultiplexingNonMutating(t *testing.T) {
	p1 := new(consumertest.MetricsSink)
	p2 := new(consumertest.MetricsSink)
	p3 := new(consumertest.MetricsSink)

	mfc := NewMetrics([]consumer.Metrics{p1, p2, p3})
	assert.False(t, mfc.Capabilities().MutatesData)
	md := testdata.GenerateMetrics(1)

	for i := 0; i < 2; i++ {
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, md == p1.AllMetrics()[0])
	assert.True(t, md == p1.AllMetrics()[1])
	assert.EqualValues(t, md, p1.AllMetrics()[0])
	assert.EqualValues(t, md, p1.AllMetrics()[1])

	assert.True(t, md == p2.AllMetrics()[0])
	assert.True(t, md == p2.AllMetrics()[1])
	assert.EqualValues(t, md, p2.AllMetrics()[0])
	assert.EqualValues(t, md, p2.AllMetrics()[1])

	assert.True(t, md == p3.AllMetrics()[0])
	assert.True(t, md == p3.AllMetrics()[1])
	assert.EqualValues(t, md, p3.AllMetrics()[0])
	assert.EqualValues(t, md, p3.AllMetrics()[1])

	// The data should be marked as read only.
	assert.True(t, md.IsReadOnly())
}

func TestMetricsMultiplexingMutating(t *testing.T) {
	p1 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p2 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p3 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}

	mfc := NewMetrics([]consumer.Metrics{p1, p2, p3})
	assert.True(t, mfc.Capabilities().MutatesData)
	md := testdata.GenerateMetrics(1)

	for i := 0; i < 2; i++ {
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, md != p1.AllMetrics()[0])
	assert.True(t, md != p1.AllMetrics()[1])
	assert.EqualValues(t, md, p1.AllMetrics()[0])
	assert.EqualValues(t, md, p1.AllMetrics()[1])

	assert.True(t, md != p2.AllMetrics()[0])
	assert.True(t, md != p2.AllMetrics()[1])
	assert.EqualValues(t, md, p2.AllMetrics()[0])
	assert.EqualValues(t, md, p2.AllMetrics()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, md == p3.AllMetrics()[0])
	assert.True(t, md == p3.AllMetrics()[1])
	assert.EqualValues(t, md, p3.AllMetrics()[0])
	assert.EqualValues(t, md, p3.AllMetrics()[1])

	// The data should not be marked as read only.
	assert.False(t, md.IsReadOnly())
}

func TestReadOnlyMetricsMultiplexingMixFirstMutating(t *testing.T) {
	p1 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p2 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p3 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}

	mfc := NewMetrics([]consumer.Metrics{p1, p2, p3})
	assert.True(t, mfc.Capabilities().MutatesData)
	mdOrig := testdata.GenerateMetrics(1)
	md := testdata.GenerateMetrics(1)
	md.MarkReadOnly()

	for i := 0; i < 2; i++ {
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	// All consumers should receive the cloned data.

	assert.True(t, md != p1.AllMetrics()[0])
	assert.True(t, md != p1.AllMetrics()[1])
	assert.EqualValues(t, mdOrig, p1.AllMetrics()[0])
	assert.EqualValues(t, mdOrig, p1.AllMetrics()[1])

	assert.True(t, md != p2.AllMetrics()[0])
	assert.True(t, md != p2.AllMetrics()[1])
	assert.EqualValues(t, mdOrig, p2.AllMetrics()[0])
	assert.EqualValues(t, mdOrig, p2.AllMetrics()[1])

	assert.True(t, md != p3.AllMetrics()[0])
	assert.True(t, md != p3.AllMetrics()[1])
	assert.EqualValues(t, mdOrig, p3.AllMetrics()[0])
	assert.EqualValues(t, mdOrig, p3.AllMetrics()[1])
}

func TestMetricsMultiplexingMixLastMutating(t *testing.T) {
	p1 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p2 := new(consumertest.MetricsSink)
	p3 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}

	mfc := NewMetrics([]consumer.Metrics{p1, p2, p3})
	assert.False(t, mfc.Capabilities().MutatesData)
	md := testdata.GenerateMetrics(1)

	for i := 0; i < 2; i++ {
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, md != p1.AllMetrics()[0])
	assert.True(t, md != p1.AllMetrics()[1])
	assert.EqualValues(t, md, p1.AllMetrics()[0])
	assert.EqualValues(t, md, p1.AllMetrics()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, md == p2.AllMetrics()[0])
	assert.True(t, md == p2.AllMetrics()[1])
	assert.EqualValues(t, md, p2.AllMetrics()[0])
	assert.EqualValues(t, md, p2.AllMetrics()[1])

	// For this consumer, will clone the initial data.
	assert.True(t, md != p3.AllMetrics()[0])
	assert.True(t, md != p3.AllMetrics()[1])
	assert.EqualValues(t, md, p3.AllMetrics()[0])
	assert.EqualValues(t, md, p3.AllMetrics()[1])

	// The data should not be marked as read only.
	assert.False(t, md.IsReadOnly())
}

func TestMetricsMultiplexingMixLastNonMutating(t *testing.T) {
	p1 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p2 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p3 := new(consumertest.MetricsSink)

	mfc := NewMetrics([]consumer.Metrics{p1, p2, p3})
	assert.False(t, mfc.Capabilities().MutatesData)
	md := testdata.GenerateMetrics(1)

	for i := 0; i < 2; i++ {
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	assert.True(t, md != p1.AllMetrics()[0])
	assert.True(t, md != p1.AllMetrics()[1])
	assert.EqualValues(t, md, p1.AllMetrics()[0])
	assert.EqualValues(t, md, p1.AllMetrics()[1])

	assert.True(t, md != p2.AllMetrics()[0])
	assert.True(t, md != p2.AllMetrics()[1])
	assert.EqualValues(t, md, p2.AllMetrics()[0])
	assert.EqualValues(t, md, p2.AllMetrics()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, md == p3.AllMetrics()[0])
	assert.True(t, md == p3.AllMetrics()[1])
	assert.EqualValues(t, md, p3.AllMetrics()[0])
	assert.EqualValues(t, md, p3.AllMetrics()[1])

	// The data should not be marked as read only.
	assert.False(t, md.IsReadOnly())
}

func TestMetricsWhenErrors(t *testing.T) {
	p1 := mutatingErr{Consumer: consumertest.NewErr(errors.New("my error"))}
	p2 := consumertest.NewErr(errors.New("my error"))
	p3 := new(consumertest.MetricsSink)

	mfc := NewMetrics([]consumer.Metrics{p1, p2, p3})
	md := testdata.GenerateMetrics(1)

	for i := 0; i < 2; i++ {
		assert.Error(t, mfc.ConsumeMetrics(context.Background(), md))
	}

	assert.True(t, md == p3.AllMetrics()[0])
	assert.True(t, md == p3.AllMetrics()[1])
	assert.EqualValues(t, md, p3.AllMetrics()[0])
	assert.EqualValues(t, md, p3.AllMetrics()[1])
}

type mutatingMetricsSink struct {
	*consumertest.MetricsSink
}

func (mts *mutatingMetricsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
