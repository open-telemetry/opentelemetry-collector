// Copyright 2018, OpenCensus Authors
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

package processortest

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/processor"
)

// TODO: Move this to processortest and use TraceData.

// ConcurrentTraceDataSink acts as a trace receiver for use in tests.
type ConcurrentTraceDataSink struct {
	mu     sync.Mutex
	traces []data.TraceData
}

var _ processor.TraceDataProcessor = (*ConcurrentTraceDataSink)(nil)

// ProcessTraceData stores traces for tests.
func (css *ConcurrentTraceDataSink) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	css.mu.Lock()
	defer css.mu.Unlock()

	css.traces = append(css.traces, td)

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (css *ConcurrentTraceDataSink) AllTraces() []data.TraceData {
	css.mu.Lock()
	defer css.mu.Unlock()

	return css.traces[:]
}

// ConcurrentMetricsDataSink acts as a metrics receiver for use in tests.
type ConcurrentMetricsDataSink struct {
	mu      sync.Mutex
	metrics []data.MetricsData
}

var _ processor.MetricsDataProcessor = (*ConcurrentMetricsDataSink)(nil)

// ProcessMetricsData stores traces for tests.
func (cms *ConcurrentMetricsDataSink) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	cms.metrics = append(cms.metrics, md)

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (cms *ConcurrentMetricsDataSink) AllMetrics() []data.MetricsData {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	return cms.metrics[:]
}

// ToJSON marshals a generic interface to JSON to enable easy comparisons.
func ToJSON(v interface{}) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
