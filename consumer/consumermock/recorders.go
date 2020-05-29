// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumermock

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type allConsumers interface {
	consumer.TraceConsumer
	consumer.TraceConsumerOld
	consumer.MetricsConsumer
	consumer.MetricsConsumerOld
}

// Recorder is an interface that satisfies all consumer interfaces and can
// be queried about data sent to it.
type Recorder interface {
	allConsumers
	GetStats() string
	ClearReceivedItems()
	SpansReceived() uint64
	MetricsReceived() uint64
	DataItemsReceived() uint64
	Traces() []pdata.Traces
	TracesOld() []consumerdata.TraceData
	Metrics() []pdata.Metrics
	MetricsOld() []consumerdata.MetricsData
}

// NilRecorder does not record any spans or metrics sent to it.
var NilRecorder Recorder = &nilRecorder{allConsumers: Nil}

var _ Recorder = (*nilRecorder)(nil)

type nilRecorder struct {
	allConsumers
}

func (n nilRecorder) Traces() []pdata.Traces {
	return nil
}

func (n nilRecorder) TracesOld() []consumerdata.TraceData {
	return nil
}

func (n nilRecorder) Metrics() []pdata.Metrics {
	return nil
}

func (n nilRecorder) MetricsOld() []consumerdata.MetricsData {
	return nil
}

func (n nilRecorder) ClearReceivedItems() {
}

func (n nilRecorder) SpansReceived() uint64 {
	return 0
}

func (n nilRecorder) MetricsReceived() uint64 {
	return 0
}

func (n nilRecorder) DataItemsReceived() uint64 {
	return 0
}

func (n nilRecorder) GetStats() string {
	return "(nil)"
}

var _ Recorder = (*InMemoryRecorder)(nil)

// InMemoryRecorder saves all spans and metrics sent to it in memory.
type InMemoryRecorder struct {
	Trace
	Metric
}

func (rec *InMemoryRecorder) GetStats() string {
	return fmt.Sprintf("Received:%5d items", rec.DataItemsReceived())
}

// DataItemsReceived returns total number of received spans and metrics.
func (rec *InMemoryRecorder) DataItemsReceived() uint64 {
	return rec.Trace.SpansReceived() + rec.Metric.MetricsReceived()
}

// ClearReceivedItems clears the list of received traces and metrics. Note: counters
// return by DataItemsReceived() are not cleared, they are cumulative.
func (rec *InMemoryRecorder) ClearReceivedItems() {
	rec.Trace.ClearReceivedItems()
	rec.Metric.ClearReceivedItems()
}
