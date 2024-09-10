// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/processor"
)

var (
	processorID = component.MustNewID("fakeProcessor")
)

func TestProcessorTraceData(t *testing.T) {
	testTelemetry(t, processorID, func(t *testing.T, tt componenttest.TestTelemetry) {
		const acceptedSpans = 27
		const refusedSpans = 19
		const droppedSpans = 13
		const insertedSpans = 5

		obsrep, err := newObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		obsrep.TracesAccepted(context.Background(), acceptedSpans)
		obsrep.TracesRefused(context.Background(), refusedSpans)
		obsrep.TracesDropped(context.Background(), droppedSpans)
		obsrep.TracesInserted(context.Background(), insertedSpans)

		require.NoError(t, tt.CheckProcessorTraces(acceptedSpans, refusedSpans, droppedSpans, insertedSpans))
	})
}

func TestProcessorMetricsData(t *testing.T) {
	testTelemetry(t, processorID, func(t *testing.T, tt componenttest.TestTelemetry) {
		const acceptedPoints = 29
		const refusedPoints = 11
		const droppedPoints = 17
		const insertedPoints = 4

		obsrep, err := newObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		obsrep.MetricsAccepted(context.Background(), acceptedPoints)
		obsrep.MetricsRefused(context.Background(), refusedPoints)
		obsrep.MetricsDropped(context.Background(), droppedPoints)
		obsrep.MetricsInserted(context.Background(), insertedPoints)

		require.NoError(t, tt.CheckProcessorMetrics(acceptedPoints, refusedPoints, droppedPoints, insertedPoints))
	})
}

func TestBuildProcessorCustomMetricName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "firstMeasure",
			want: "processor_test_type_firstMeasure",
		},
		{
			name: "secondMeasure",
			want: "processor_test_type_secondMeasure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCustomMetricName("test_type", tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessorLogRecords(t *testing.T) {
	testTelemetry(t, processorID, func(t *testing.T, tt componenttest.TestTelemetry) {
		const acceptedRecords = 29
		const refusedRecords = 11
		const droppedRecords = 17
		const insertedRecords = 3

		obsrep, err := newObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		obsrep.LogsAccepted(context.Background(), acceptedRecords)
		obsrep.LogsRefused(context.Background(), refusedRecords)
		obsrep.LogsDropped(context.Background(), droppedRecords)
		obsrep.LogsInserted(context.Background(), insertedRecords)

		require.NoError(t, tt.CheckProcessorLogs(acceptedRecords, refusedRecords, droppedRecords, insertedRecords))
	})
}

func TestCheckProcessorTracesViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(processorID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := NewObsReport(ObsReportSettings{
		ProcessorID:             processorID,
		ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	assert.NoError(t, err)

	por.TracesAccepted(context.Background(), 7)
	por.TracesRefused(context.Background(), 8)
	por.TracesDropped(context.Background(), 9)
	por.TracesInserted(context.Background(), 10)

	assert.NoError(t, tt.CheckProcessorTraces(7, 8, 9, 10))
	assert.Error(t, tt.CheckProcessorTraces(0, 0, 0, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 0, 0, 0))
	assert.Error(t, tt.CheckProcessorTraces(0, 8, 0, 0))
	assert.Error(t, tt.CheckProcessorTraces(0, 0, 9, 0))
	assert.Error(t, tt.CheckProcessorTraces(0, 0, 0, 10))
	assert.Error(t, tt.CheckProcessorTraces(7, 8, 0, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 0, 9, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 0, 0, 10))
	assert.Error(t, tt.CheckProcessorTraces(0, 8, 9, 0))
	assert.Error(t, tt.CheckProcessorTraces(0, 8, 0, 10))
	assert.Error(t, tt.CheckProcessorTraces(0, 0, 9, 10))
	assert.Error(t, tt.CheckProcessorTraces(7, 8, 9, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 8, 0, 10))
	assert.Error(t, tt.CheckProcessorTraces(7, 0, 9, 10))
	assert.Error(t, tt.CheckProcessorTraces(0, 8, 9, 10))
}

func TestCheckProcessorMetricsViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(processorID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := NewObsReport(ObsReportSettings{
		ProcessorID:             processorID,
		ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	assert.NoError(t, err)

	por.MetricsAccepted(context.Background(), 7)
	por.MetricsRefused(context.Background(), 8)
	por.MetricsDropped(context.Background(), 9)
	por.MetricsInserted(context.Background(), 10)

	assert.NoError(t, tt.CheckProcessorMetrics(7, 8, 9, 10))
	assert.Error(t, tt.CheckProcessorMetrics(0, 0, 0, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 0, 0, 0))
	assert.Error(t, tt.CheckProcessorMetrics(0, 8, 0, 0))
	assert.Error(t, tt.CheckProcessorMetrics(0, 0, 9, 0))
	assert.Error(t, tt.CheckProcessorMetrics(0, 0, 0, 10))
	assert.Error(t, tt.CheckProcessorMetrics(7, 8, 0, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 0, 9, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 0, 0, 10))
	assert.Error(t, tt.CheckProcessorMetrics(0, 8, 9, 0))
	assert.Error(t, tt.CheckProcessorMetrics(0, 8, 0, 10))
	assert.Error(t, tt.CheckProcessorMetrics(0, 0, 9, 10))
	assert.Error(t, tt.CheckProcessorMetrics(7, 8, 9, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 8, 0, 10))
	assert.Error(t, tt.CheckProcessorMetrics(7, 0, 9, 10))
	assert.Error(t, tt.CheckProcessorMetrics(0, 8, 9, 10))
}

func TestCheckProcessorLogViews(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(processorID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := NewObsReport(ObsReportSettings{
		ProcessorID:             processorID,
		ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	assert.NoError(t, err)

	por.LogsAccepted(context.Background(), 7)
	por.LogsRefused(context.Background(), 8)
	por.LogsDropped(context.Background(), 9)
	por.LogsInserted(context.Background(), 10)

	assert.NoError(t, tt.CheckProcessorLogs(7, 8, 9, 10))
	assert.Error(t, tt.CheckProcessorLogs(0, 0, 0, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 0, 0, 0))
	assert.Error(t, tt.CheckProcessorLogs(0, 8, 0, 0))
	assert.Error(t, tt.CheckProcessorLogs(0, 0, 9, 0))
	assert.Error(t, tt.CheckProcessorLogs(0, 0, 0, 10))
	assert.Error(t, tt.CheckProcessorLogs(7, 8, 0, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 0, 9, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 0, 0, 10))
	assert.Error(t, tt.CheckProcessorLogs(0, 8, 9, 0))
	assert.Error(t, tt.CheckProcessorLogs(0, 8, 0, 10))
	assert.Error(t, tt.CheckProcessorLogs(0, 0, 9, 10))
	assert.Error(t, tt.CheckProcessorLogs(7, 8, 9, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 8, 0, 10))
	assert.Error(t, tt.CheckProcessorLogs(7, 0, 9, 10))
	assert.Error(t, tt.CheckProcessorLogs(0, 8, 9, 10))
}

func TestNoMetrics(t *testing.T) {
	// ensure if LevelNone is configured, no metrics are emitted by the component
	testTelemetry(t, processorID, func(t *testing.T, tt componenttest.TestTelemetry) {
		const accepted = 29
		const refused = 11
		const dropped = 17
		const inserted = 5

		set := tt.TelemetrySettings()
		set.LeveledMeterProvider = func(_ configtelemetry.Level) metric.MeterProvider {
			return noop.MeterProvider{}
		}

		por, err := NewObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: set, BuildInfo: component.NewDefaultBuildInfo()},
		})
		assert.NoError(t, err)

		por.TracesAccepted(context.Background(), accepted)
		por.TracesRefused(context.Background(), refused)
		por.TracesDropped(context.Background(), dropped)
		por.TracesInserted(context.Background(), inserted)

		require.Error(t, tt.CheckProcessorTraces(accepted, refused, dropped, inserted))
	})
	testTelemetry(t, processorID, func(t *testing.T, tt componenttest.TestTelemetry) {
		const accepted = 29
		const refused = 11
		const dropped = 17
		const inserted = 4

		set := tt.TelemetrySettings()
		set.LeveledMeterProvider = func(_ configtelemetry.Level) metric.MeterProvider {
			return noop.MeterProvider{}
		}

		por, err := NewObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: set, BuildInfo: component.NewDefaultBuildInfo()},
		})
		assert.NoError(t, err)

		por.MetricsAccepted(context.Background(), accepted)
		por.MetricsRefused(context.Background(), refused)
		por.MetricsDropped(context.Background(), dropped)
		por.MetricsInserted(context.Background(), inserted)

		require.Error(t, tt.CheckProcessorMetrics(accepted, refused, dropped, inserted))
	})
	testTelemetry(t, processorID, func(t *testing.T, tt componenttest.TestTelemetry) {
		const accepted = 29
		const refused = 11
		const dropped = 17
		const inserted = 3

		set := tt.TelemetrySettings()
		set.LeveledMeterProvider = func(_ configtelemetry.Level) metric.MeterProvider {
			return noop.MeterProvider{}
		}

		por, err := NewObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.Settings{ID: processorID, TelemetrySettings: set, BuildInfo: component.NewDefaultBuildInfo()},
		})
		assert.NoError(t, err)

		por.LogsAccepted(context.Background(), accepted)
		por.LogsRefused(context.Background(), refused)
		por.LogsDropped(context.Background(), dropped)
		por.LogsInserted(context.Background(), inserted)

		require.Error(t, tt.CheckProcessorLogs(accepted, refused, dropped, inserted))
	})
}

func testTelemetry(t *testing.T, id component.ID, testFunc func(t *testing.T, tt componenttest.TestTelemetry)) {
	tt, err := componenttest.SetupTelemetry(id)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	testFunc(t, tt)
}
