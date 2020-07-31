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
package exporterhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	fakeMetricsExporterType   = "fake_metrics_exporter"
	fakeMetricsExporterName   = "fake_metrics_exporter/with_name"
	fakeMetricsParentSpanName = "fake_metrics_parent_span_name"
)

var (
	fakeMetricsExporterConfig = &configmodels.ExporterSettings{
		TypeVal: fakeMetricsExporterType,
		NameVal: fakeMetricsExporterName,
	}
)

func TestMetricsRequest(t *testing.T) {
	mr := newMetricsRequest(context.Background(), testdata.GenerateMetricsEmpty(), nil)

	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataOneSpan())
	assert.Same(t, mr, mr.onPartialError(partialErr.(consumererror.PartialError)))
	assert.Equal(t, 0, mr.count())
}

func TestMetricsExporter_InvalidName(t *testing.T) {
	me, err := NewMetricsExporter(nil, newPushMetricsData(0, nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestMetricsExporter_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetricsData, err)
}

func TestMetricsExporter_Default(t *testing.T) {
	md := testdata.GenerateMetricsEmpty()
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Nil(t, me.ConsumeMetrics(context.Background(), md))
	assert.Nil(t, me.Shutdown(context.Background()))
}

func TestMetricsExporter_Default_ReturnError(t *testing.T) {
	md := testdata.GenerateMetricsEmpty()
	want := errors.New("my_error")
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsExporter_Observability(t *testing.T) {
	checkObservabilityForMetricsExporter(t, nil, 0)
}

func TestMetricsExporter_Observability_NonZeroDropped(t *testing.T) {
	checkObservabilityForMetricsExporter(t, nil, 1)
}

func TestMetricsExporter_Observability_ReturnError(t *testing.T) {
	checkObservabilityForMetricsExporter(t, errors.New("my_error"), 1)
}

func TestMetricsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Nil(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Equal(t, me.Shutdown(context.Background()), want)
}

func newPushMetricsData(droppedTimeSeries int, retError error) PushMetricsData {
	return func(ctx context.Context, td pdata.Metrics) (int, error) {
		return droppedTimeSeries, retError
	}
}

func checkObservabilityForMetricsExporter(t *testing.T, wantError error, droppedMetricPoints int) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	traceProvider, ime := obsreporttest.SetupSdkTraceProviderTest(t)
	tracer := traceProvider.Tracer("go.opentelemetry.io/collector/exporter/metricshelper")

	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(droppedMetricPoints, wantError), withOtelProviders(traceProvider))
	require.Nil(t, err)
	require.NotNil(t, me)

	const numRequests = 5
	const numMetricPoints = 4
	md := testdata.GenerateMetricsTwoMetrics()
	ctx, span := tracer.Start(context.Background(), fakeMetricsParentSpanName)
	for i := 0; i < numRequests; i++ {
		assert.Equal(t, wantError, me.ConsumeMetrics(ctx, md))
	}
	span.End()

	// Inspection time!
	gotSpanData := ime.GetSpans()
	require.Len(t, gotSpanData, numRequests+1, "No exported span data")

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeMetricsParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		assert.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		obsreporttest.CheckSpanStatus(t, wantError, sd)
		sentMetricPoints := numMetricPoints
		failedToSendMetricPoints := 0
		if wantError != nil {
			sentMetricPoints = 0
			failedToSendMetricPoints = numMetricPoints
		}
		obsreporttest.CheckExporterMetricsSpanAttributes(t, sd, int64(sentMetricPoints), int64(failedToSendMetricPoints))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	numPoints := int64(numRequests * numMetricPoints)
	if wantError != nil {
		obsreporttest.CheckExporterMetricsViews(t, fakeMetricsExporterName, 0, numPoints)
	} else {
		obsreporttest.CheckExporterMetricsViews(t, fakeMetricsExporterName, numPoints, 0)
	}
}
