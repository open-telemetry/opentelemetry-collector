// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	fakeMetricsParentSpanName = "fake_metrics_parent_span_name"
)

var (
	fakeMetricsExporterName   = component.NewIDWithName("fake_metrics_exporter", "with_name")
	fakeMetricsExporterConfig = struct{}{}
)

func TestMetricsRequest(t *testing.T) {
	mr := newMetricsRequest(context.Background(), testdata.GenerateMetrics(1), nil)

	metricsErr := consumererror.NewMetrics(errors.New("some error"), pmetric.NewMetrics())
	assert.EqualValues(
		t,
		newMetricsRequest(context.Background(), pmetric.NewMetrics(), nil),
		mr.OnError(metricsErr),
	)
}

func TestMetricsExporter_InvalidName(t *testing.T) {
	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), nil, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestMetricsExporter_NilLogger(t *testing.T) {
	me, err := NewMetricsExporter(context.Background(), exporter.CreateSettings{}, &fakeMetricsExporterConfig, newPushMetricsData(nil))
	require.Nil(t, me)
	require.Equal(t, errNilLogger, err)
}

func TestMetricsExporter_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeMetricsExporterConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetricsData, err)
}

func TestMetricsExporter_Default(t *testing.T) {
	md := pmetric.NewMetrics()
	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(nil))
	assert.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, consumer.Capabilities{MutatesData: false}, me.Capabilities())
	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, me.Shutdown(context.Background()))
}

func TestMetricsExporter_WithCapabilities(t *testing.T) {
	capabilities := consumer.Capabilities{MutatesData: true}
	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(nil), WithCapabilities(capabilities))
	assert.NoError(t, err)
	assert.NotNil(t, me)

	assert.Equal(t, capabilities, me.Capabilities())
}

func TestMetricsExporter_Default_ReturnError(t *testing.T) {
	md := pmetric.NewMetrics()
	want := errors.New("my_error")
	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeMetrics(context.Background(), md))
}

func TestMetricsExporter_WithRecordMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeMetricsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetricsExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, tt, me, nil)
}

func TestMetricsExporter_WithRecordMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	tt, err := obsreporttest.SetupTelemetry(fakeMetricsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	me, err := NewMetricsExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)

	checkRecordedMetricsForMetricsExporter(t, tt, me, want)
}

func TestMetricsExporter_WithRecordEnqueueFailedMetrics(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(fakeMetricsExporterName)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rCfg := NewDefaultRetrySettings()
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	qCfg.QueueSize = 2
	wantErr := errors.New("some-error")
	te, err := NewMetricsExporter(context.Background(), tt.ToExporterCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(wantErr), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NotNil(t, te)

	md := testdata.GenerateMetrics(1)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		// errors are checked in the checkExporterEnqueueFailedMetricsStats function below.
		_ = te.ConsumeMetrics(context.Background(), md)
	}

	// 2 batched must be in queue, and 10 metric points rejected due to queue overflow
	checkExporterEnqueueFailedMetricsStats(t, globalInstruments, fakeMetricsExporterName, int64(10))
}

func TestMetricsExporter_WithSpan(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	me, err := NewMetricsExporter(context.Background(), set, &fakeMetricsExporterConfig, newPushMetricsData(nil))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, sr, set.TracerProvider.Tracer("test"), me, nil, 2)
}

func TestMetricsExporter_WithSpan_ReturnError(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	sr := new(tracetest.SpanRecorder)
	set.TracerProvider = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(set.TracerProvider)
	defer otel.SetTracerProvider(trace.NewNoopTracerProvider())

	want := errors.New("my_error")
	me, err := NewMetricsExporter(context.Background(), set, &fakeMetricsExporterConfig, newPushMetricsData(want))
	require.NoError(t, err)
	require.NotNil(t, me)
	checkWrapSpanForMetricsExporter(t, sr, set.TracerProvider.Tracer("test"), me, want, 2)
}

func TestMetricsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestMetricsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), &fakeMetricsExporterConfig, newPushMetricsData(nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.NoError(t, me.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, me.Shutdown(context.Background()))
}

func newPushMetricsData(retError error) consumer.ConsumeMetricsFunc {
	return func(ctx context.Context, td pmetric.Metrics) error {
		return retError
	}
}

func checkRecordedMetricsForMetricsExporter(t *testing.T, tt obsreporttest.TestTelemetry, me exporter.Metrics, wantError error) {
	md := testdata.GenerateMetrics(2)
	const numBatches = 7
	for i := 0; i < numBatches; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(context.Background(), md))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	numPoints := int64(numBatches * md.MetricCount() * 2) /* 2 points per metric*/
	if wantError != nil {
		require.NoError(t, tt.CheckExporterMetrics(0, numPoints))
	} else {
		require.NoError(t, tt.CheckExporterMetrics(numPoints, 0))
	}
}

func generateMetricsTraffic(t *testing.T, tracer trace.Tracer, me exporter.Metrics, numRequests int, wantError error) {
	md := testdata.GenerateMetrics(1)
	ctx, span := tracer.Start(context.Background(), fakeMetricsParentSpanName)
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, me.ConsumeMetrics(ctx, md))
	}
}

func checkWrapSpanForMetricsExporter(t *testing.T, sr *tracetest.SpanRecorder, tracer trace.Tracer, me exporter.Metrics, wantError error, numMetricPoints int64) {
	const numRequests = 5
	generateMetricsTraffic(t, tracer, me, numRequests, wantError)

	// Inspection time!
	gotSpanData := sr.Ended()
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeMetricsParentSpanName, parentSpan.Name(), "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext(), sd.Parent(), "Exporter span not a child\nSpanData %v", sd)
		checkStatus(t, sd, wantError)

		sentMetricPoints := numMetricPoints
		var failedToSendMetricPoints int64
		if wantError != nil {
			sentMetricPoints = 0
			failedToSendMetricPoints = numMetricPoints
		}
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.SentMetricPointsKey, Value: attribute.Int64Value(sentMetricPoints)}, "SpanData %v", sd)
		require.Containsf(t, sd.Attributes(), attribute.KeyValue{Key: obsmetrics.FailedToSendMetricPointsKey, Value: attribute.Int64Value(failedToSendMetricPoints)}, "SpanData %v", sd)
	}
}
