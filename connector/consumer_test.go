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

package connector // import "go.opentelemetry.io/collector/connector"

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/fanoutconsumer"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestTracesRouter(t *testing.T) {
	var max = 20
	for numIDs := 1; numIDs < max; numIDs++ {
		for numCons := 1; numCons < max; numCons++ {
			for numTraces := 1; numTraces < max; numTraces++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numTraces),
					fuzzTracesRouter(numIDs, numCons, numTraces),
				)
			}
		}
	}
}

func fuzzTracesRouter(numIDs, numCons, numTraces int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]component.ID, 0, numCons)
		allCons := make([]consumer.Traces, 0, numCons)
		allConsMap := make(map[component.ID]consumer.Traces)

		// If any consumer is mutating, the router must report mutating
		for i := 0; i < numCons; i++ {
			allIDs = append(allIDs, component.NewIDWithName("sink", strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numTraces+i)%4 == 0 {
				allCons = append(allCons, &mutatingTracesSink{TracesSink: new(consumertest.TracesSink)})
			} else {
				allCons = append(allCons, new(consumertest.TracesSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewTracesConsumer(allConsMap)
		assert.False(t, r.Capabilities().MutatesData)

		consumers := r.Consumers()

		td := testdata.GenerateTraces(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteTraces.
		expected := make(map[component.ID]int, numCons)

		for i := 0; i < numTraces; i++ {
			// Build a random set of ids (no duplicates)
			randCons := make(map[component.ID]bool, numIDs)
			for j := 0; j < numIDs; j++ {
				// This number should be pretty random and less than numCons
				conNum := (numCons + numIDs + i + j) % numCons
				randCons[allIDs[conNum]] = true
			}

			tcs := make([]consumer.Traces, 0, len(randCons))
			for id := range randCons {
				tcs = append(tcs, consumers[id])
				expected[id]++
			}

			// Route to list of consumers
			assert.NoError(t, fanoutconsumer.NewTraces(tcs).ConsumeTraces(context.Background(), td))

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
					assert.EqualValues(t, td, traces[n])
				}
			}
		}
	}
}

func TestTracesRouterGetConsumer(t *testing.T) {
	ctx := context.Background()
	ld := testdata.GenerateTraces(1)

	fooID := component.NewID("foo")
	barID := component.NewID("bar")

	foo := new(consumertest.TracesSink)
	bar := new(consumertest.TracesSink)
	r := NewTracesConsumer(map[component.ID]consumer.Traces{fooID: foo, barID: bar})
	assert.Len(t, foo.AllTraces(), 0)
	assert.Len(t, bar.AllTraces(), 0)

	cons := r.Consumers()

	both := fanoutconsumer.NewTraces([]consumer.Traces{cons[fooID], cons[barID]})
	assert.NoError(t, both.ConsumeTraces(ctx, ld))
	assert.Len(t, foo.AllTraces(), 1)
	assert.Len(t, bar.AllTraces(), 1)

	fooOnly := fanoutconsumer.NewTraces([]consumer.Traces{cons[fooID]})
	assert.NoError(t, fooOnly.ConsumeTraces(ctx, ld))
	assert.Len(t, foo.AllTraces(), 2)
	assert.Len(t, bar.AllTraces(), 1)

	barOnly := fanoutconsumer.NewTraces([]consumer.Traces{cons[barID]})
	assert.NoError(t, barOnly.ConsumeTraces(ctx, ld))
	assert.Len(t, foo.AllTraces(), 2)
	assert.Len(t, bar.AllTraces(), 2)
}

type mutatingTracesSink struct {
	*consumertest.TracesSink
}

func (mts *mutatingTracesSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestMetricsRouter(t *testing.T) {
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

		r := NewMetricsConsumer(allConsMap)
		assert.False(t, r.Capabilities().MutatesData)

		consumers := r.Consumers()

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

			mcs := make([]consumer.Metrics, 0, len(randCons))
			for id := range randCons {
				mcs = append(mcs, consumers[id])
				expected[id]++
			}

			// Route to list of consumers
			assert.NoError(t, fanoutconsumer.NewMetrics(mcs).ConsumeMetrics(context.Background(), md))

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

func TestMetricsRouterGetConsumer(t *testing.T) {
	ctx := context.Background()
	ld := testdata.GenerateMetrics(1)

	fooID := component.NewID("foo")
	barID := component.NewID("bar")

	foo := new(consumertest.MetricsSink)
	bar := new(consumertest.MetricsSink)
	r := NewMetricsConsumer(map[component.ID]consumer.Metrics{fooID: foo, barID: bar})
	assert.Len(t, foo.AllMetrics(), 0)
	assert.Len(t, bar.AllMetrics(), 0)

	cons := r.Consumers()

	both := fanoutconsumer.NewMetrics([]consumer.Metrics{cons[fooID], cons[barID]})
	assert.NoError(t, both.ConsumeMetrics(ctx, ld))
	assert.Len(t, foo.AllMetrics(), 1)
	assert.Len(t, bar.AllMetrics(), 1)

	fooOnly := fanoutconsumer.NewMetrics([]consumer.Metrics{cons[fooID]})
	assert.NoError(t, fooOnly.ConsumeMetrics(ctx, ld))
	assert.Len(t, foo.AllMetrics(), 2)
	assert.Len(t, bar.AllMetrics(), 1)

	barOnly := fanoutconsumer.NewMetrics([]consumer.Metrics{cons[barID]})
	assert.NoError(t, barOnly.ConsumeMetrics(ctx, ld))
	assert.Len(t, foo.AllMetrics(), 2)
	assert.Len(t, bar.AllMetrics(), 2)
}

type mutatingMetricsSink struct {
	*consumertest.MetricsSink
}

func (mts *mutatingMetricsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func TestLogsRouter(t *testing.T) {
	var max = 20
	for numIDs := 1; numIDs < max; numIDs++ {
		for numCons := 1; numCons < max; numCons++ {
			for numLogs := 1; numLogs < max; numLogs++ {
				t.Run(
					fmt.Sprintf("%d-ids/%d-cons/%d-logs", numIDs, numCons, numLogs),
					fuzzLogsRouter(numIDs, numCons, numLogs),
				)
			}
		}
	}
}

func fuzzLogsRouter(numIDs, numCons, numLogs int) func(*testing.T) {
	return func(t *testing.T) {
		allIDs := make([]component.ID, 0, numCons)
		allCons := make([]consumer.Logs, 0, numCons)
		allConsMap := make(map[component.ID]consumer.Logs)

		// If any consumer is mutating, the router must report mutating
		for i := 0; i < numCons; i++ {
			allIDs = append(allIDs, component.NewIDWithName("sink", strconv.Itoa(numCons)))
			// Random chance for each consumer to be mutating
			if (numCons+numLogs+i)%4 == 0 {
				allCons = append(allCons, &mutatingLogsSink{LogsSink: new(consumertest.LogsSink)})
			} else {
				allCons = append(allCons, new(consumertest.LogsSink))
			}
			allConsMap[allIDs[i]] = allCons[i]
		}

		r := NewLogsConsumer(allConsMap)
		assert.False(t, r.Capabilities().MutatesData)

		consumers := r.Consumers()

		ld := testdata.GenerateLogs(1)

		// Keep track of how many logs each consumer should receive.
		// This will be validated after every call to RouteLogs.
		expected := make(map[component.ID]int, numCons)

		for i := 0; i < numLogs; i++ {
			// Build a random set of ids (no duplicates)
			randCons := make(map[component.ID]bool, numIDs)
			for j := 0; j < numIDs; j++ {
				// This number should be pretty random and less than numCons
				conNum := (numCons + numIDs + i + j) % numCons
				randCons[allIDs[conNum]] = true
			}

			lcs := make([]consumer.Logs, 0, len(randCons))
			for id := range randCons {
				lcs = append(lcs, consumers[id])
				expected[id]++
			}

			// Route to list of consumers
			assert.NoError(t, fanoutconsumer.NewLogs(lcs).ConsumeLogs(context.Background(), ld))

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
					assert.EqualValues(t, ld, logs[n])
				}
			}
		}
	}
}

func TestLogsRouterGetConsumer(t *testing.T) {
	ctx := context.Background()
	ld := testdata.GenerateLogs(1)

	fooID := component.NewID("foo")
	barID := component.NewID("bar")

	foo := new(consumertest.LogsSink)
	bar := new(consumertest.LogsSink)
	r := NewLogsConsumer(map[component.ID]consumer.Logs{fooID: foo, barID: bar})
	assert.Len(t, foo.AllLogs(), 0)
	assert.Len(t, bar.AllLogs(), 0)

	cons := r.Consumers()

	both := fanoutconsumer.NewLogs([]consumer.Logs{cons[fooID], cons[barID]})
	assert.NoError(t, both.ConsumeLogs(ctx, ld))
	assert.Len(t, foo.AllLogs(), 1)
	assert.Len(t, bar.AllLogs(), 1)

	fooOnly := fanoutconsumer.NewLogs([]consumer.Logs{cons[fooID]})
	assert.NoError(t, fooOnly.ConsumeLogs(ctx, ld))
	assert.Len(t, foo.AllLogs(), 2)
	assert.Len(t, bar.AllLogs(), 1)

	barOnly := fanoutconsumer.NewLogs([]consumer.Logs{cons[barID]})
	assert.NoError(t, barOnly.ConsumeLogs(ctx, ld))
	assert.Len(t, foo.AllLogs(), 2)
	assert.Len(t, bar.AllLogs(), 2)
}

type mutatingLogsSink struct {
	*consumertest.LogsSink
}

func (mts *mutatingLogsSink) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}
