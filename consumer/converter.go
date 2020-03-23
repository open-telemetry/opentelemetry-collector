// Copyright 2019, OpenTelemetry Authors
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

// Package consumer contains interfaces that receive and process consumerdata.
package consumer

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/internaldata"
)

// NewInternalToOCTraceConverter creates new internalToOCTraceConverter that takes TraceConsumer and
// implements ConsumeTrace interface.
func NewInternalToOCTraceConverter(tc TraceConsumer) TraceConsumerV2 {
	return &internalToOCTraceConverter{tc}
}

// internalToOCTraceConverter is a internal to oc translation shim that takes TraceConsumer and
// implements ConsumeTrace interface.
type internalToOCTraceConverter struct {
	traceConsumer TraceConsumer
}

// ConsumeTrace takes new-style data.TraceData method, converts it to OC and uses old-style ConsumeTraceData method
// to process the trace data.
func (tc *internalToOCTraceConverter) ConsumeTrace(ctx context.Context, td data.TraceData) error {
	ocTraces := internaldata.TraceDataToOC(td)
	for i := range ocTraces {
		err := tc.traceConsumer.ConsumeTraceData(ctx, ocTraces[i])
		if err != nil {
			return err
		}
	}
	return nil
}

var _ TraceConsumerV2 = (*internalToOCTraceConverter)(nil)

// NewInternalToOCMetricsConverter creates new internalToOCMetricsConverter that takes MetricsConsumer and
// implements ConsumeTrace interface.
func NewInternalToOCMetricsConverter(tc MetricsConsumer) MetricsConsumerV2 {
	return &internalToOCMetricsConverter{tc}
}

// internalToOCMetricsConverter is a internal to oc translation shim that takes MetricsConsumer and
// implements ConsumeMetrics interface.
type internalToOCMetricsConverter struct {
	metricsConsumer MetricsConsumer
}

// ConsumeMetrics takes new-style data.MetricData method, converts it to OC and uses old-style ConsumeMetricsData method
// to process the metrics data.
func (tc *internalToOCMetricsConverter) ConsumeMetrics(ctx context.Context, td data.MetricData) error {
	ocMetrics := internaldata.MetricDataToOC(td)
	for i := range ocMetrics {
		err := tc.metricsConsumer.ConsumeMetricsData(ctx, ocMetrics[i])
		if err != nil {
			return err
		}
	}
	return nil
}

var _ MetricsConsumerV2 = (*internalToOCMetricsConverter)(nil)
