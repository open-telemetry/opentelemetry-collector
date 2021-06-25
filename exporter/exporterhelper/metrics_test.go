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
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumerhelper"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	fakeMetricsParentSpanName = "fake_metrics_parent_span_name"
)

var (
	fakeMetricsExporterName   = config.NewIDWithName("fake_metrics_exporter", "with_name")
	fakeMetricsExporterConfig = config.NewExporterSettings(fakeMetricsExporterName)
)

func TestMetricsRequest(t *testing.T) {
	mr := newMetricsRequest(context.Background(), testdata.GenerateMetricsOneMetric(), nil)

	metricsErr := consumererror.NewMetrics(errors.New("some error"), pdata.NewMetrics())
	assert.EqualValues(
		t,
		newMetricsRequest(context.Background(), pdata.NewMetrics(), nil),
		mr.onError(metricsErr),
	)
}

func TestMetricsExporter_InvalidName(t *testing.T) {
	me, err := NewMetricsExporter(nil, zap.NewNop(), newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestMetricsExporter_NilLogger(t *testing.T) {
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, nil, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetricsExporter_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetricsData, err)
}

func TestMetricsExporter_Default(t *testing.T) {
	md := pdata.NewMetrics()
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil))
	assert.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil), WithCapabilities(capabilities))
	assert.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetricsExporter_Default_ReturnError(t *testing.T) {
	md := pdata.NewMetrics()
	want := errors.New("my_error")
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsExporter_WithRecordMetrics(t *testing.T) {
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, me, nil)
}

func TestMetricsExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, me, want)
}

func TestMetricsExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	rCfg := DefaultRetrySettings()
	qCfg := DefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateMetricsOneMetricOneDataPoint()
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		te.ConsumeMetrics(context.Background(), md)
	}

	// 2 batched must be in queue, and 5 metric points rejected due to queue overflow
	checkExporterEnqueueFailedMetricsStats(t, fakeMetricsExporterName, int64(5))
}

func TestMetricsExporter_WithSpan(t *testing.T) {
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, me, nil, 1)
}

func TestMetricsExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, me, want, 1)
}

func TestMetricsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsExporter_WithResourceToTelemetryConversionDisabled(t *testing.T) {
	md := testdata.GenerateMetricsTwoMetrics()
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil), WithResourceToTelemetryConversion(defaultResourceToTelemetrySettings()))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsExporter_WithResourceToTelemetryConversionEnabled(t *testing.T) {
	md := testdata.GenerateMetricsTwoMetrics()
	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil), WithResourceToTelemetryConversion(ResourceToTelemetrySettings{Enabled: true}))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsExporter(&fakeMetricsExporterConfig, zap.NewNop(), newPushMetricsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, me.Shutdown(context.Background()))
}

func newPushMetricsData(retError error) consumerhelper.ConsumeMetricsFunc {
	return func(ctx context.Context, td pdata.Metrics) error {
		return retError
	}
}

func checkRecordedMetricsForMetricsExporter(t *testing.T, me component.MetricsExporter, wantError error) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	md := testdata.GenerateMetricsTwoMetrics()
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(context.Background(), md))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	numPoints := int64(numBatches * md.MetricCount() * 2) /* 2 points per metric*/
	if wantError != nil {
		obsreporttest.CheckExporterMetrics(t, fakeMetricsExporterName, 0, numPoints)
	} else {
		obsreporttest.CheckExporterMetrics(t, fakeMetricsExporterName, numPoints, 0)
	}
}

func generateMetricsTraffic(t *testing.T, me component.MetricsExporter, numRequests int, wantError error) {
	md := testdata.GenerateMetricsOneMetricOneDataPoint()
	ctx, span := trace.StartSpan(context.Background(), fakeMetricsParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(ctx, md))
	}
}

func checkWrapSpanForMetricsExporter(t *testing.T, me component.MetricsExporter, wantError error, numMetricPoints int64) {
	ocSpansSaver := new(testOCTracesExporter)
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
		require.Equalf(t, sentMetricPoints, sd.Attributes[obsmetrics.SentMetricPointsKey], "SpanData %v", sd)
		require.Equalf(t, failedToSendMetricPoints, sd.Attributes[obsmetrics.FailedToSendMetricPointsKey], "SpanData %v", sd)
	}
}
