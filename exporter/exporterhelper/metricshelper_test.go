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
package exporterhelper

import (
	"context"
	"errors"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/observability"
	"github.com/open-telemetry/opentelemetry-collector/observability/observabilitytest"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
)

const (
	fakeMetricsReceiverName   = "fake_receiver"
	fakeMetricsExporterType   = "fake_metrics_exporter"
	fakeMetricsExporterName   = "fake_metrics_exporter/with_name"
	fakeMetricsParentSpanName = "fake_metrics_parent_span_name"
)

var (
	fakeMetricsExporterConfig = &configmodels.ExporterSettings{
		TypeVal:  fakeMetricsExporterType,
		NameVal:  fakeMetricsExporterName,
		Disabled: false,
	}
)

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
	md := testdata.GenerateMetricDataEmpty()
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.ConsumeMetrics(context.Background(), md))
	assert.Nil(t, me.Shutdown(context.Background()))
}

func TestMetricsExporter_Default_ReturnError(t *testing.T) {
	md := testdata.GenerateMetricDataEmpty()
	want := errors.New("my_error")
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsExporter_WithRecordMetrics(t *testing.T) {
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, me, nil, 0)
}

func TestMetricsExporter_WithRecordMetrics_NonZeroDropped(t *testing.T) {
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(1, nil))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, me, nil, 1)
}

func TestMetricsExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, me, want, 0)
}

func TestMetricsExporter_WithSpan(t *testing.T) {
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, me, nil, 1)
}

func TestMetricsExporter_WithSpan_NonZeroDropped(t *testing.T) {
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(1, nil))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, me, nil, 1)
}

func TestMetricsExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, me, want, 1)
}

func TestMetricsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsExporter(fakeMetricsExporterConfig, newPushMetricsData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Equal(t, me.Shutdown(context.Background()), want)
}

func TestMetricsExporterOld_InvalidName(t *testing.T) {
	me, err := NewMetricsExporterOld(nil, newPushMetricsDataOld(0, nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestMetricsExporterOld_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetricsData, err)
}

func TestMetricsExporterOld_Default(t *testing.T) {
	md := consumerdata.MetricsData{}
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, nil))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.ConsumeMetricsData(context.Background(), md))
	assert.Nil(t, me.Shutdown(context.Background()))
}

func TestMetricsExporterOld_Default_ReturnError(t *testing.T) {
	md := consumerdata.MetricsData{}
	want := errors.New("my_error")
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetricsData(context.Background(), md))
}

func TestMetricsExporterOld_WithRecordMetrics(t *testing.T) {
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, nil))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporterOld(t, me, nil, 0)
}

func TestMetricsExporterOld_WithRecordMetrics_NonZeroDropped(t *testing.T) {
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(1, nil))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporterOld(t, me, nil, 1)
}

func TestMetricsExporterOld_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporterOld(t, me, want, 0)
}

func TestMetricsExporterOld_WithSpan(t *testing.T) {
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, nil))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporterOld(t, me, nil, 1)
}

func TestMetricsExporterOld_WithSpan_NonZeroDropped(t *testing.T) {
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(1, nil))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporterOld(t, me, nil, 1)
}

func TestMetricsExporterOld_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporterOld(t, me, want, 1)
}

func TestMetricsExporterOld_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsExporterOld_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsExporterOld(fakeMetricsExporterConfig, newPushMetricsDataOld(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Equal(t, me.Shutdown(context.Background()), want)
}

func newPushMetricsData(droppedTimeSeries int, retError error) PushMetricsData {
	return func(ctx context.Context, td data.MetricData) (int, error) {
		return droppedTimeSeries, retError
	}
}

func checkRecordedMetricsForMetricsExporter(t *testing.T, me component.MetricsExporter, wantError error, droppedTimeSeries int) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()

	md := testdata.GenerateMetricDataTwoMetrics()
	ctx := observability.ContextWithReceiverName(context.Background(), fakeMetricsReceiverName)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(ctx, md))
	}

	err := observabilitytest.CheckValueViewExporterReceivedTimeSeries(fakeMetricsReceiverName, fakeMetricsExporterName, numBatches*md.MetricCount())
	require.Nilf(t, err, "CheckValueViewExporterTimeSeries: Want nil Got %v", err)

	err = observabilitytest.CheckValueViewExporterDroppedTimeSeries(fakeMetricsReceiverName, fakeMetricsExporterName, numBatches*droppedTimeSeries)
	require.Nilf(t, err, "CheckValueViewExporterTimeSeries: Want nil Got %v", err)
}

func generateMetricsTraffic(t *testing.T, me component.MetricsExporter, numRequests int, wantError error) {
	md := testdata.GenerateMetricDataOneMetricOneDataPoint()
	ctx, span := trace.StartSpan(context.Background(), fakeMetricsParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(ctx, md))
	}
}

func checkWrapSpanForMetricsExporter(t *testing.T, me component.MetricsExporter, wantError error, numMetricPoints int64) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateMetricsTraffic(t, me, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data")

	gotSpanData := ocSpansSaver.spanData
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeMetricsParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)

		sentMetricPoints := numMetricPoints
		var failedToSendMetricPoints int64
		if wantError != nil {
			sentMetricPoints = 0
			failedToSendMetricPoints = numMetricPoints
		}
		require.Equalf(t, sentMetricPoints, sd.Attributes[obsreport.SentMetricPointsKey], "SpanData %v", sd)
		require.Equalf(t, failedToSendMetricPoints, sd.Attributes[obsreport.FailedToSendMetricPointsKey], "SpanData %v", sd)
	}
}

func newPushMetricsDataOld(droppedTimeSeries int, retError error) PushMetricsDataOld {
	return func(ctx context.Context, td consumerdata.MetricsData) (int, error) {
		return droppedTimeSeries, retError
	}
}

func checkRecordedMetricsForMetricsExporterOld(t *testing.T, me component.MetricsExporterOld, wantError error, droppedTimeSeries int) {
	doneFn := observabilitytest.SetupRecordedMetricsTest()
	defer doneFn()
	metrics := []*metricspb.Metric{
		{
			Timeseries: make([]*metricspb.TimeSeries, 1),
		},
		{
			Timeseries: make([]*metricspb.TimeSeries, 1),
		},
	}
	md := consumerdata.MetricsData{Metrics: metrics}
	ctx := observability.ContextWithReceiverName(context.Background(), fakeMetricsReceiverName)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeMetricsData(ctx, md))
	}

	err := observabilitytest.CheckValueViewExporterReceivedTimeSeries(fakeMetricsReceiverName, fakeMetricsExporterName, numBatches*NumTimeSeries(md))
	require.Nilf(t, err, "CheckValueViewExporterTimeSeries: Want nil Got %v", err)

	err = observabilitytest.CheckValueViewExporterDroppedTimeSeries(fakeMetricsReceiverName, fakeMetricsExporterName, numBatches*droppedTimeSeries)
	require.Nilf(t, err, "CheckValueViewExporterTimeSeries: Want nil Got %v", err)
}

func generateMetricsTrafficOld(t *testing.T, me component.MetricsExporterOld, numRequests int, wantError error) {
	md := consumerdata.MetricsData{Metrics: []*metricspb.Metric{
		{
			// Create a empty timeseries with one point.
			Timeseries: []*metricspb.TimeSeries{
				{
					Points: []*metricspb.Point{{}},
				},
			},
		},
	}}
	ctx, span := trace.StartSpan(context.Background(), fakeMetricsParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, me.ConsumeMetricsData(ctx, md))
	}
}

func checkWrapSpanForMetricsExporterOld(t *testing.T, me component.MetricsExporterOld, wantError error, numMetricPoints int64) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateMetricsTrafficOld(t, me, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data")

	gotSpanData := ocSpansSaver.spanData
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeMetricsParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)

		sentMetricPoints := numMetricPoints
		var failedToSendMetricPoints int64
		if wantError != nil {
			sentMetricPoints = 0
			failedToSendMetricPoints = numMetricPoints
		}
		require.Equalf(t, sentMetricPoints, sd.Attributes[obsreport.SentMetricPointsKey], "SpanData %v", sd)
		require.Equalf(t, failedToSendMetricPoints, sd.Attributes[obsreport.FailedToSendMetricPointsKey], "SpanData %v", sd)
	}
}
