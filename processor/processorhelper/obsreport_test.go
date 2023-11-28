// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/processor"
)

var (
	processorID = component.NewID("fakeProcessor")
)

func TestProcessorTraceData(t *testing.T) {
	testTelemetry(t, processorID, func(t *testing.T, tt obsreporttest.TestTelemetry) {
		const acceptedSpans = 27
		const refusedSpans = 19
		const droppedSpans = 13
		obsrep, err := newObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.CreateSettings{ID: processorID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		obsrep.TracesAccepted(context.Background(), acceptedSpans)
		obsrep.TracesRefused(context.Background(), refusedSpans)
		obsrep.TracesDropped(context.Background(), droppedSpans)

		require.NoError(t, tt.CheckProcessorTraces(acceptedSpans, refusedSpans, droppedSpans))
	})
}

func TestProcessorMetricsData(t *testing.T) {
	testTelemetry(t, processorID, func(t *testing.T, tt obsreporttest.TestTelemetry) {
		const acceptedPoints = 29
		const refusedPoints = 11
		const droppedPoints = 17

		obsrep, err := newObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.CreateSettings{ID: processorID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		obsrep.MetricsAccepted(context.Background(), acceptedPoints)
		obsrep.MetricsRefused(context.Background(), refusedPoints)
		obsrep.MetricsDropped(context.Background(), droppedPoints)

		require.NoError(t, tt.CheckProcessorMetrics(acceptedPoints, refusedPoints, droppedPoints))
	})
}

func TestBuildProcessorCustomMetricName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "firstMeasure",
			want: "processor/test_type/firstMeasure",
		},
		{
			name: "secondMeasure",
			want: "processor/test_type/secondMeasure",
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
	testTelemetry(t, processorID, func(t *testing.T, tt obsreporttest.TestTelemetry) {
		const acceptedRecords = 29
		const refusedRecords = 11
		const droppedRecords = 17

		obsrep, err := newObsReport(ObsReportSettings{
			ProcessorID:             processorID,
			ProcessorCreateSettings: processor.CreateSettings{ID: processorID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
		})
		require.NoError(t, err)
		obsrep.LogsAccepted(context.Background(), acceptedRecords)
		obsrep.LogsRefused(context.Background(), refusedRecords)
		obsrep.LogsDropped(context.Background(), droppedRecords)

		require.NoError(t, tt.CheckProcessorLogs(acceptedRecords, refusedRecords, droppedRecords))
	})
}

func testTelemetry(t *testing.T, id component.ID, testFunc func(t *testing.T, tt obsreporttest.TestTelemetry)) {
	t.Run("WithOTel", func(t *testing.T) {
		tt, err := obsreporttest.SetupTelemetry(id)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

		testFunc(t, tt)
	})
}
