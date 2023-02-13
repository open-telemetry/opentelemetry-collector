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
	"go.opentelemetry.io/collector/pdata/pmetric"
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

func TestMetricsRouterMultiplexing(t *testing.T) {
	var max = 20
	for numIDs := 1; numIDs < max; numIDs++ {
		for numCons := 1; numCons < max; numCons++ {
			for numMetrics := 1; numMetrics < max; numMetrics++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numMetrics),
					fuzzMetricsRouter(numIDs, numCons, numMetrics),
				)
			}
		}
	}
}

func fuzzMetricsRouter(numIDs, numCons, numMetrics int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]component.ID, 0, numCons)
		allCons := make([]consumer.Metrics, 0, numCons)
		allConsMap := make(map[component.ID]consumer.Metrics)

		// If any consumer is mutating, the router must report mutating
		for i := 0; i < numCons; i++ {
			allIDs = append(allIDs, component.NewIDWithName("sink", strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numMetrics+i)%4 == 0 {
				allCons = append(allCons, &mutatingMetricsSink{MetricsSink: new(consumertest.MetricsSink)})
			} else {
				allCons = append(allCons, new(consumertest.MetricsSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewMetricsRouter(allConsMap).(connector.MetricsRouter)
		md := testdata.GenerateMetrics(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteMetrics.
		expected := make(map[component.ID]int, numCons)

		for i := 0; i < numMetrics; i++ {
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
			assert.NoError(t, fanout.ConsumeMetrics(context.Background(), md))

			// Validate expectations for all consumers
			for id := range expected {
				metrics := []pmetric.Metrics{}
				switch con := allConsMap[id].(type) {
				case *consumertest.MetricsSink:
					metrics = con.AllMetrics()
				case *mutatingMetricsSink:
					metrics = con.AllMetrics()
				}
				assert.Len(t, metrics, expected[id])
				for n := 0; n < len(metrics); n++ {
					assert.EqualValues(t, md, metrics[n])
				}
			}
		}
	}
}

func TestMetricsRouterGetConsumers(t *testing.T) {
	ctx := context.Background()
	md := testdata.GenerateMetrics(1)

	fooID := component.NewID("foo")
	barID := component.NewID("bar")

	foo := new(consumertest.MetricsSink)
	bar := new(consumertest.MetricsSink)
	r := NewMetricsRouter(map[component.ID]consumer.Metrics{fooID: foo, barID: bar}).(connector.MetricsRouter)

	rcs := r.PipelineIDs()
	assert.Len(t, rcs, 2)
	assert.ElementsMatch(t, []component.ID{fooID, barID}, rcs)

	assert.Len(t, foo.AllMetrics(), 0)
	assert.Len(t, bar.AllMetrics(), 0)

	both, err := r.Consumer(fooID, barID)
	assert.NotNil(t, both)
	assert.NoError(t, err)

	assert.NoError(t, both.ConsumeMetrics(ctx, md))
	assert.Len(t, foo.AllMetrics(), 1)
	assert.Len(t, bar.AllMetrics(), 1)

	fooOnly, err := r.Consumer(fooID)
	assert.NotNil(t, fooOnly)
	assert.NoError(t, err)

	assert.NoError(t, fooOnly.ConsumeMetrics(ctx, md))
	assert.Len(t, foo.AllMetrics(), 2)
	assert.Len(t, bar.AllMetrics(), 1)

	barOnly, err := r.Consumer(barID)
	assert.NotNil(t, barOnly)
	assert.NoError(t, err)

	assert.NoError(t, barOnly.ConsumeMetrics(ctx, md))
	assert.Len(t, foo.AllMetrics(), 2)
	assert.Len(t, bar.AllMetrics(), 2)

	none, err := r.Consumer()
	assert.Nil(t, none)
	assert.Error(t, err)

	fake, err := r.Consumer(component.NewID("fake"))
	assert.Nil(t, fake)
	assert.Error(t, err)
}
