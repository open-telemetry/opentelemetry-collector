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

package testhelper

import (
	"context"
	"encoding/json"
	"sync"

	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/processor"
)

// TODO: Move this to processortest and use TraceData.

// ConcurrentSpanSink acts as a trace receiver for use in tests.
type ConcurrentSpanSink struct {
	mu     sync.Mutex
	traces []*agenttracepb.ExportTraceServiceRequest
}

var _ processor.TraceDataProcessor = (*ConcurrentSpanSink)(nil)

// ProcessTraceData stores traces for tests.
func (css *ConcurrentSpanSink) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	css.mu.Lock()
	defer css.mu.Unlock()

	css.traces = append(css.traces, &agenttracepb.ExportTraceServiceRequest{
		Node:  td.Node,
		Spans: td.Spans,
	})

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (css *ConcurrentSpanSink) AllTraces() []*agenttracepb.ExportTraceServiceRequest {
	css.mu.Lock()
	defer css.mu.Unlock()

	return css.traces[:]
}

// ConcurrentMetricsSink acts as a metrics receiver for use in tests.
type ConcurrentMetricsSink struct {
	mu      sync.Mutex
	metrics []*agentmetricspb.ExportMetricsServiceRequest
}

var _ processor.MetricsDataProcessor = (*ConcurrentMetricsSink)(nil)

// ProcessMetricsData stores traces for tests.
func (cms *ConcurrentMetricsSink) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	cms.metrics = append(cms.metrics, &agentmetricspb.ExportMetricsServiceRequest{
		Node:     md.Node,
		Resource: md.Resource,
		Metrics:  md.Metrics,
	})

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (cms *ConcurrentMetricsSink) AllMetrics() []*agentmetricspb.ExportMetricsServiceRequest {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	return cms.metrics[:]
}

// ToJSON marshals a generic interface to JSON to enable easy comparisons.
func ToJSON(v interface{}) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
