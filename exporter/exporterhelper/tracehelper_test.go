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
	fakeTraceExporterType   = "fake_trace_exporter"
	fakeTraceExporterName   = "fake_trace_exporter/with_name"
	fakeTraceParentSpanName = "fake_trace_parent_span_name"
)

var (
	fakeTraceExporterConfig = &configmodels.ExporterSettings{
		TypeVal: fakeTraceExporterType,
		NameVal: fakeTraceExporterName,
	}
)

func TestTracesRequest(t *testing.T) {
	mr := newTracesRequest(context.Background(), testdata.GenerateTraceDataOneSpan(), nil)

	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataEmpty())
	assert.EqualValues(t, newTracesRequest(context.Background(), testdata.GenerateTraceDataEmpty(), nil), mr.onPartialError(partialErr.(consumererror.PartialError)))
}

func TestTraceExporter_InvalidName(t *testing.T) {
	te, err := NewTraceExporter(nil, newTraceDataPusher(0, nil))
	require.Nil(t, te)
	require.Equal(t, errNilConfig, err)
}

func TestTraceExporter_NilPushTraceData(t *testing.T) {
	te, err := NewTraceExporter(fakeTraceExporterConfig, nil)
	require.Nil(t, te)
	require.Equal(t, errNilPushTraceData, err)
}

func TestTraceExporter_Default(t *testing.T) {
	td := pdata.NewTraces()
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.ConsumeTraces(context.Background(), td))
	assert.Nil(t, te.Shutdown(context.Background()))
}

func TestTraceExporter_Default_ReturnError(t *testing.T) {
	td := pdata.NewTraces()
	want := errors.New("my_error")
	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, want))
	require.Nil(t, err)
	require.NotNil(t, te)

	err = te.ConsumeTraces(context.Background(), td)
	require.Equalf(t, want, err, "ConsumeTraceData returns: Want %v Got %v", want, err)
}

func TestTraceExporter_Observability(t *testing.T) {
	checkObservabilityForTraceExporter(t, nil, 0)
}

func TestTraceExporter_Observability_NonZeroDropped(t *testing.T) {
	checkObservabilityForTraceExporter(t, nil, 1)
}

func TestTraceExporter_Observability_ReturnError(t *testing.T) {
	checkObservabilityForTraceExporter(t, errors.New("my_error"), 0)
}

func TestTraceExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Nil(t, te.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestTraceExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, te)
	assert.NoError(t, err)

	assert.Equal(t, te.Shutdown(context.Background()), want)
}

func newTraceDataPusher(droppedSpans int, retError error) traceDataPusher {
	return func(ctx context.Context, td pdata.Traces) (int, error) {
		return droppedSpans, retError
	}
}

func checkObservabilityForTraceExporter(t *testing.T, wantError error, droppedSpans int) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	traceProvider, ime := obsreporttest.SetupSdkTraceProviderTest(t)
	tracer := traceProvider.Tracer("go.opentelemetry.io/collector/exporter/tracesshelper")

	te, err := NewTraceExporter(fakeTraceExporterConfig, newTraceDataPusher(droppedSpans, wantError), withOtelProviders(traceProvider))
	require.Nil(t, err)
	require.NotNil(t, te)

	const numRequests = 5
	const numSpans = 2
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	ctx, span := tracer.Start(context.Background(), fakeTraceParentSpanName)
	for i := 0; i < numRequests; i++ {
		assert.Equal(t, wantError, te.ConsumeTraces(ctx, td))
	}
	span.End()

	// Inspection time!
	gotSpanData := ime.GetSpans()
	require.Len(t, gotSpanData, numRequests+1, "No exported span data")

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeTraceParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		assert.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		obsreporttest.CheckSpanStatus(t, wantError, sd)
		sentSpans := numSpans
		failedToSendSpans := 0
		if wantError != nil {
			sentSpans = 0
			failedToSendSpans = numSpans
		}
		obsreporttest.CheckExporterTracesSpanAttributes(t, sd, int64(sentSpans), int64(failedToSendSpans))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		obsreporttest.CheckExporterTracesViews(t, fakeTraceExporterName, 0, int64(numRequests*td.SpanCount()))
	} else {
		obsreporttest.CheckExporterTracesViews(t, fakeTraceExporterName, int64(numRequests*td.SpanCount()), 0)
	}
}
