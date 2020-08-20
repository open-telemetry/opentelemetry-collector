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
package converter

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// NewInternalToOCTraceConverter creates new internalToOCTraceConverter that takes TraceConsumer and
// implements ConsumeTraces interface.
func NewInternalToOCTraceConverter(tc consumer.TraceConsumerOld) consumer.TraceConsumer {
	return &internalToOCTraceConverter{tc}
}

// internalToOCTraceConverter is a internal to oc translation shim that takes TraceConsumer and
// implements ConsumeTraces interface.
type internalToOCTraceConverter struct {
	traceConsumer consumer.TraceConsumerOld
}

// ConsumeTraces takes new-style data.Traces method, converts it to OC and uses old-style ConsumeTraceData method
// to process the trace data.
func (tc *internalToOCTraceConverter) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	ocTraces := internaldata.TraceDataToOC(td)
	for i := range ocTraces {
		err := tc.traceConsumer.ConsumeTraceData(ctx, ocTraces[i])
		if err != nil {
			return err
		}
	}
	return nil
}

var _ consumer.TraceConsumer = (*internalToOCTraceConverter)(nil)

// NewInternalToOCMetricsConverter creates new internalToOCMetricsConverter that takes MetricsConsumer and
// implements ConsumeTraces interface.
func NewInternalToOCMetricsConverter(mc consumer.MetricsConsumerOld) consumer.MetricsConsumer {
	return &internalToOCMetricsConverter{mc}
}

// internalToOCMetricsConverter is a internal to oc translation shim that takes MetricsConsumer and
// implements ConsumeMetrics interface.
type internalToOCMetricsConverter struct {
	metricsConsumer consumer.MetricsConsumerOld
}

// ConsumeMetrics takes new-style data.MetricData method, converts it to OC and uses old-style ConsumeMetricsData method
// to process the metrics data.
func (tc *internalToOCMetricsConverter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	ocMetrics := pdatautil.MetricsToMetricsData(md)
	for i := range ocMetrics {
		err := tc.metricsConsumer.ConsumeMetricsData(ctx, ocMetrics[i])
		if err != nil {
			return err
		}
	}
	return nil
}

var _ consumer.MetricsConsumer = (*internalToOCMetricsConverter)(nil)

// NewOCToInternalTraceConverter creates new ocToInternalTraceConverter that takes TraceConsumer and
// implements ConsumeTraces interface.
func NewOCToInternalTraceConverter(tc consumer.TraceConsumer) consumer.TraceConsumerOld {
	return &ocToInternalTraceConverter{tc}
}

// ocToInternalTraceConverter is a internal to oc translation shim that takes TraceConsumer and
// implements ConsumeTraces interface.
type ocToInternalTraceConverter struct {
	traceConsumer consumer.TraceConsumer
}

// ConsumeTraces takes new-style data.Traces method, converts it to OC and uses old-style ConsumeTraceData method
// to process the trace data.
func (tc *ocToInternalTraceConverter) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	traceData := internaldata.OCToTraceData(td)
	err := tc.traceConsumer.ConsumeTraces(ctx, traceData)
	if err != nil {
		return err
	}

	return nil
}

var _ consumer.TraceConsumerOld = (*ocToInternalTraceConverter)(nil)

// NewOCToInternalMetricsConverter creates new ocToInternalMetricsConverter that takes MetricsConsumer and
// implements ConsumeTraces interface.
func NewOCToInternalMetricsConverter(tc consumer.MetricsConsumer) consumer.MetricsConsumerOld {
	return &ocToInternalMetricsConverter{tc}
}

// ocToInternalMetricsConverter is a internal to oc translation shim that takes MetricsConsumer and
// implements ConsumeMetrics interface.
type ocToInternalMetricsConverter struct {
	metricsConsumer consumer.MetricsConsumer
}

// ConsumeMetrics takes new-style data.MetricData method, converts it to OC and uses old-style ConsumeMetricsData method
// to process the metrics data.
func (tc *ocToInternalMetricsConverter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	metricsData := pdatautil.MetricsFromMetricsData([]consumerdata.MetricsData{md})
	err := tc.metricsConsumer.ConsumeMetrics(ctx, metricsData)
	if err != nil {
		return err
	}
	return nil
}

var _ consumer.MetricsConsumerOld = (*ocToInternalMetricsConverter)(nil)
