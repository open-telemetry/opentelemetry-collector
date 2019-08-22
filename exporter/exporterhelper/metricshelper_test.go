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

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter"
)

func TestMetricsExporter_InvalidName(t *testing.T) {
	me, err := NewMetricsExporter("", newPushMetricsData(0, nil))
	require.Nil(t, me)
	require.Equal(t, errEmptyExporterName, err)
}

func TestMetricsExporter_NilPushMetricsData(t *testing.T) {
	me, err := NewMetricsExporter(fakeExporterName, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushMetricsData, err)
}

func TestMetricsExporter_Default(t *testing.T) {
	md := consumerdata.MetricsData{}
	me, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.ConsumeMetricsData(context.Background(), md))

	assert.Equal(t, me.Name(), fakeExporterName)

	assert.Nil(t, me.Shutdown())
}

func TestMetricsExporter_Default_ReturnError(t *testing.T) {
	td := consumerdata.MetricsData{}
	want := errors.New("my_error")
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)
	require.Equal(t, want, te.ConsumeMetricsData(context.Background(), td))
}

func TestMetricsExporter_WithSpan(t *testing.T) {
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil), WithSpanName(fakeSpanName))
	require.Nil(t, err)
	require.NotNil(t, te)
	checkWrapSpanForMetricsExporter(t, te, nil, 0)
}

func TestMetricsExporter_WithSpan_NonZeroDropped(t *testing.T) {
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(1, nil), WithSpanName(fakeSpanName))
	require.Nil(t, err)
	require.NotNil(t, te)
	checkWrapSpanForMetricsExporter(t, te, nil, 1)
}

func TestMetricsExporter_WithSpan_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	te, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, want), WithSpanName(fakeSpanName))
	require.Nil(t, err)
	require.NotNil(t, te)
	checkWrapSpanForMetricsExporter(t, te, want, 0)
}

func TestMetricsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func() error { shutdownCalled = true; return nil }

	me, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Nil(t, me.Shutdown())
	assert.True(t, shutdownCalled)
}

func TestMetricsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func() error { return want }

	me, err := NewMetricsExporter(fakeExporterName, newPushMetricsData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.Nil(t, err)

	assert.Equal(t, me.Shutdown(), want)
}

func newPushMetricsData(droppedSpans int, retError error) PushMetricsData {
	return func(ctx context.Context, td consumerdata.MetricsData) (int, error) {
		return droppedSpans, retError
	}
}

func generateMetricsTraffic(t *testing.T, te exporter.MetricsExporter, numRequests int, wantError error) {
	td := consumerdata.MetricsData{Metrics: make([]*metricspb.Metric, 1)}
	ctx, span := trace.StartSpan(context.Background(), fakeParentSpanName, trace.WithSampler(trace.AlwaysSample()))
	defer span.End()
	for i := 0; i < numRequests; i++ {
		require.Equal(t, wantError, te.ConsumeMetricsData(ctx, td))
	}
}

func checkWrapSpanForMetricsExporter(t *testing.T, te exporter.MetricsExporter, wantError error, droppedSpans int) {
	ocSpansSaver := new(testOCTraceExporter)
	trace.RegisterExporter(ocSpansSaver)
	defer trace.UnregisterExporter(ocSpansSaver)

	const numRequests = 5
	generateMetricsTraffic(t, te, numRequests, wantError)

	// Inspection time!
	ocSpansSaver.mu.Lock()
	defer ocSpansSaver.mu.Unlock()

	require.NotEqual(t, 0, len(ocSpansSaver.spanData), "No exported span data")

	gotSpanData := ocSpansSaver.spanData[:]
	require.Equal(t, numRequests+1, len(gotSpanData))

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)

	for _, sd := range gotSpanData[:numRequests] {
		require.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		require.Equalf(t, errToStatus(wantError), sd.Status, "SpanData %v", sd)
		require.Equalf(t, int64(1), sd.Attributes[numReceivedMetricsAttribute], "SpanData %v", sd)
		require.Equalf(t, int64(droppedSpans), sd.Attributes[numDroppedMetricsAttribute], "SpanData %v", sd)
	}
}
