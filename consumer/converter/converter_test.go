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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func TestNewInternalToOCTraceConverter(t *testing.T) {
	td := testdata.GenerateTraceDataTwoSpansSameResourceOneDifferent()
	traceExporterOld := new(exportertest.SinkTraceExporterOld)
	converter := NewInternalToOCTraceConverter(traceExporterOld)

	err := converter.ConsumeTraces(context.Background(), td)
	assert.NoError(t, err)

	ocTraces := traceExporterOld.AllTraces()
	assert.Equal(t, len(ocTraces), 2)
	assert.EqualValues(t, ocTraces, internaldata.TraceDataToOC(td))

	traceExporterOld.SetConsumeTraceError(fmt.Errorf("consumer error"))
	err = converter.ConsumeTraces(context.Background(), td)
	assert.NotNil(t, err)
}

func TestNewInternalToOCMetricsConverter(t *testing.T) {
	md := testdata.GenerateMetricDataOneMetric()
	metricsExporterOld := new(exportertest.SinkMetricsExporterOld)
	converter := NewInternalToOCMetricsConverter(metricsExporterOld)

	err := converter.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(md))
	assert.NoError(t, err)

	ocMetrics := metricsExporterOld.AllMetrics()
	assert.Equal(t, len(ocMetrics), 1)
	assert.EqualValues(t, ocMetrics, internaldata.MetricDataToOC(md))

	metricsExporterOld.SetConsumeMetricsError(fmt.Errorf("consumer error"))
	err = converter.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(md))
	assert.NotNil(t, err)
}

func TestNewOCTraceToInternalTraceConverter(t *testing.T) {
	td := testdata.GenerateTraceDataOneSpan()
	ocTraceData := internaldata.TraceDataToOC(td)[0]
	traceExporter := new(exportertest.SinkTraceExporter)
	converter := NewOCToInternalTraceConverter(traceExporter)

	err := converter.ConsumeTraceData(context.Background(), ocTraceData)
	assert.NoError(t, err)
	err = converter.ConsumeTraceData(context.Background(), ocTraceData)
	assert.NoError(t, err)

	ocTraces := traceExporter.AllTraces()
	assert.Equal(t, len(ocTraces), 2)
	assert.EqualValues(t, ocTraces[0], td)

	traceExporter.SetConsumeTraceError(fmt.Errorf("consumer error"))
	err = converter.ConsumeTraceData(context.Background(), ocTraceData)
	assert.NotNil(t, err)
}

func TestNewOCToInternalMetricsConverter(t *testing.T) {
	md := testdata.GenerateMetricDataOneMetric()
	ocMetricData := internaldata.MetricDataToOC(md)[0]
	metricsExporter := new(exportertest.SinkMetricsExporter)
	converter := NewOCToInternalMetricsConverter(metricsExporter)

	err := converter.ConsumeMetricsData(context.Background(), ocMetricData)
	assert.NoError(t, err)
	err = converter.ConsumeMetricsData(context.Background(), ocMetricData)
	assert.NoError(t, err)

	ocMetrics := metricsExporter.AllMetrics()
	assert.Equal(t, len(ocMetrics), 2)
	assert.EqualValues(t, pdatautil.MetricsToInternalMetrics(ocMetrics[0]), md)
	assert.EqualValues(t, pdatautil.MetricsToInternalMetrics(ocMetrics[1]), md)

	metricsExporter.SetConsumeMetricsError(fmt.Errorf("consumer error"))
	err = converter.ConsumeMetricsData(context.Background(), ocMetricData)
	assert.NotNil(t, err)
}
