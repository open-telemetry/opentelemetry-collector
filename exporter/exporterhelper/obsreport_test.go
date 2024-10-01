// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/pipeline"
)

var exporterID = component.MustNewID("fakeExporter")

func TestExportEnqueueFailure(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := internal.NewExporter(internal.ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.Settings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)

	logRecords := int64(7)
	obsrep.RecordEnqueueFailure(context.Background(), pipeline.SignalLogs, logRecords)
	require.NoError(t, tt.CheckExporterEnqueueFailedLogs(logRecords))

	spans := int64(12)
	obsrep.RecordEnqueueFailure(context.Background(), pipeline.SignalTraces, spans)
	require.NoError(t, tt.CheckExporterEnqueueFailedTraces(spans))

	metricPoints := int64(21)
	obsrep.RecordEnqueueFailure(context.Background(), pipeline.SignalMetrics, metricPoints)
	require.NoError(t, tt.CheckExporterEnqueueFailedMetrics(metricPoints))
}
