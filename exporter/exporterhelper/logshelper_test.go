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
	fakeLogsExporterType   = "fake_logs_exporter"
	fakeLogsExporterName   = "fake_logs_exporter/with_name"
	fakeLogsParentSpanName = "fake_logs_parent_span_name"
)

var (
	fakeLogsExporterConfig = &configmodels.ExporterSettings{
		TypeVal: fakeLogsExporterType,
		NameVal: fakeLogsExporterName,
	}
)

func TestLogsRequest(t *testing.T) {
	mr := newLogsRequest(context.Background(), testdata.GenerateLogDataEmpty(), nil)

	partialErr := consumererror.PartialTracesError(errors.New("some error"), testdata.GenerateTraceDataOneSpan())
	assert.Same(t, mr, mr.onPartialError(partialErr.(consumererror.PartialError)))
}

func TestLogsExporter_InvalidName(t *testing.T) {
	me, err := NewLogsExporter(nil, newPushLogsData(0, nil))
	require.Nil(t, me)
	require.Equal(t, errNilConfig, err)
}

func TestLogsExporter_NilPushLogsData(t *testing.T) {
	me, err := NewLogsExporter(fakeLogsExporterConfig, nil)
	require.Nil(t, me)
	require.Equal(t, errNilPushLogsData, err)
}

func TestLogsExporter_Default(t *testing.T) {
	ld := testdata.GenerateLogDataEmpty()
	me, err := NewLogsExporter(fakeLogsExporterConfig, newPushLogsData(0, nil))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Nil(t, me.ConsumeLogs(context.Background(), ld))
	assert.Nil(t, me.Shutdown(context.Background()))
}

func TestLogsExporter_Default_ReturnError(t *testing.T) {
	ld := testdata.GenerateLogDataEmpty()
	want := errors.New("my_error")
	me, err := NewLogsExporter(fakeLogsExporterConfig, newPushLogsData(0, want))
	require.Nil(t, err)
	require.NotNil(t, me)
	require.Equal(t, want, me.ConsumeLogs(context.Background(), ld))
}

func TestLogsExporter_Observability(t *testing.T) {
	checkObservabilityForLogsExporter(t, nil, 1, 0)
}

func TestLogsExporter_Observability_NonZeroDropped(t *testing.T) {
	checkObservabilityForLogsExporter(t, nil, 1, 1)
}

func TestLogsExporter_Observability_ReturnError(t *testing.T) {
	checkObservabilityForLogsExporter(t, errors.New("my_error"), 1, 0)
}

func TestLogsExporter_WithShutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func(context.Context) error { shutdownCalled = true; return nil }

	me, err := NewLogsExporter(fakeLogsExporterConfig, newPushLogsData(0, nil), WithShutdown(shutdown))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Nil(t, me.Shutdown(context.Background()))
	assert.True(t, shutdownCalled)
}

func TestLogsExporter_WithShutdown_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	shutdownErr := func(context.Context) error { return want }

	me, err := NewLogsExporter(fakeLogsExporterConfig, newPushLogsData(0, nil), WithShutdown(shutdownErr))
	assert.NotNil(t, me)
	assert.NoError(t, err)

	assert.Equal(t, me.Shutdown(context.Background()), want)
}

func newPushLogsData(droppedLogRecords int, retError error) PushLogsData {
	return func(ctx context.Context, td pdata.Logs) (int, error) {
		return droppedLogRecords, retError
	}
}

func checkObservabilityForLogsExporter(t *testing.T, wantError error, numLogRecords, droppedLogRecords int) {
	doneFn, err := obsreporttest.SetupRecordedMetricsTest()
	require.NoError(t, err)
	defer doneFn()

	traceProvider, ime := obsreporttest.SetupSdkTraceProviderTest(t)
	tracer := traceProvider.Tracer("go.opentelemetry.io/collector/exporter/logshelper")

	me, err := NewLogsExporter(fakeLogsExporterConfig, newPushLogsData(droppedLogRecords, wantError), withOtelProviders(traceProvider))
	require.Nil(t, err)
	require.NotNil(t, me)

	const numRequests = 5
	ld := testdata.GenerateLogDataOneLog()
	ctx, span := tracer.Start(context.Background(), fakeLogsParentSpanName)
	for i := 0; i < numRequests; i++ {
		assert.Equal(t, wantError, me.ConsumeLogs(ctx, ld))
	}
	span.End()

	// Inspection time!
	gotSpanData := ime.GetSpans()
	require.Len(t, gotSpanData, numRequests+1, "No exported span data")

	parentSpan := gotSpanData[numRequests]
	require.Equalf(t, fakeLogsParentSpanName, parentSpan.Name, "SpanData %v", parentSpan)
	for _, sd := range gotSpanData[:numRequests] {
		assert.Equalf(t, parentSpan.SpanContext.SpanID, sd.ParentSpanID, "Exporter span not a child\nSpanData %v", sd)
		obsreporttest.CheckSpanStatus(t, wantError, sd)

		sentLogRecords := numLogRecords
		failedToSendLogRecords := 0
		if wantError != nil {
			sentLogRecords = 0
			failedToSendLogRecords = numLogRecords
		}
		obsreporttest.CheckExporterLogsSpanAttributes(t, sd, int64(sentLogRecords), int64(failedToSendLogRecords))
	}

	// TODO: When the new metrics correctly count partial dropped fix this.
	if wantError != nil {
		obsreporttest.CheckExporterLogsViews(t, fakeLogsExporterName, 0, int64(numRequests*ld.LogRecordCount()))
	} else {
		obsreporttest.CheckExporterLogsViews(t, fakeLogsExporterName, int64(numRequests*ld.LogRecordCount()), 0)
	}
}
