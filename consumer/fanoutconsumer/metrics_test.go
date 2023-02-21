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
}

func TestMetricsMultiplexingMutating(t *testing.T) {
	p1 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
	p2 := &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)}
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

	assert.True(t, md != p2.AllMetrics()[0])
	assert.True(t, md != p2.AllMetrics()[1])
	assert.EqualValues(t, md, p2.AllMetrics()[0])
	assert.EqualValues(t, md, p2.AllMetrics()[1])

	// For this consumer, will receive the initial data.
	assert.True(t, md == p3.AllMetrics()[0])
	assert.True(t, md == p3.AllMetrics()[1])
	assert.EqualValues(t, md, p3.AllMetrics()[0])
	assert.EqualValues(t, md, p3.AllMetrics()[1])
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
