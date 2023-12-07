// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreporttest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	scraperID   = component.NewID("fakeScraper")
	receiverID  = component.NewID("fakeReicever")
	processorID = component.NewID("fakeProcessor")
	exporterID  = component.NewID("fakeExporter")
)

func TestCheckScraperMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	s, err := scraperhelper.NewObsReport(scraperhelper.ObsReportSettings{
		ReceiverID:             receiverID,
		Scraper:                scraperID,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := s.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	s.EndMetricsOp(ctx, 7, nil)

	assert.NoError(t, obsreporttest.CheckScraperMetrics(tt, receiverID, scraperID, 7, 0))
	assert.Error(t, obsreporttest.CheckScraperMetrics(tt, receiverID, scraperID, 7, 7))
	assert.Error(t, obsreporttest.CheckScraperMetrics(tt, receiverID, scraperID, 0, 0))
	assert.Error(t, obsreporttest.CheckScraperMetrics(tt, receiverID, scraperID, 0, 7))
}

func TestCheckReceiverTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndTracesOp(ctx, format, 7, nil)

	assert.NoError(t, tt.CheckReceiverTraces(transport, 7, 0))
	assert.Error(t, tt.CheckReceiverTraces(transport, 7, 7))
	assert.Error(t, tt.CheckReceiverTraces(transport, 0, 0))
	assert.Error(t, tt.CheckReceiverTraces(transport, 0, 7))
}

func TestCheckReceiverMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndMetricsOp(ctx, format, 7, nil)

	assert.NoError(t, tt.CheckReceiverMetrics(transport, 7, 0))
	assert.Error(t, tt.CheckReceiverMetrics(transport, 7, 7))
	assert.Error(t, tt.CheckReceiverMetrics(transport, 0, 0))
	assert.Error(t, tt.CheckReceiverMetrics(transport, 0, 7))
}

func TestCheckReceiverLogsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(receiverID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             receiverID,
		Transport:              transport,
		ReceiverCreateSettings: receiver.CreateSettings{ID: receiverID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := rec.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndLogsOp(ctx, format, 7, nil)

	assert.NoError(t, tt.CheckReceiverLogs(transport, 7, 0))
	assert.Error(t, tt.CheckReceiverLogs(transport, 7, 7))
	assert.Error(t, tt.CheckReceiverLogs(transport, 0, 0))
	assert.Error(t, tt.CheckReceiverLogs(transport, 0, 7))
}

func TestCheckProcessorTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(processorID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := processorhelper.NewObsReport(processorhelper.ObsReportSettings{
		ProcessorID:             processorID,
		ProcessorCreateSettings: processor.CreateSettings{ID: processorID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	assert.NoError(t, err)

	por.TracesAccepted(context.Background(), 7)
	por.TracesRefused(context.Background(), 8)
	por.TracesDropped(context.Background(), 9)

	assert.NoError(t, tt.CheckProcessorTraces(7, 8, 9))
	assert.Error(t, tt.CheckProcessorTraces(0, 0, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 0, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 8, 0))
	assert.Error(t, tt.CheckProcessorTraces(7, 0, 9))
	assert.Error(t, tt.CheckProcessorTraces(0, 8, 0))
	assert.Error(t, tt.CheckProcessorTraces(0, 8, 9))
	assert.Error(t, tt.CheckProcessorTraces(0, 0, 9))
}

func TestCheckProcessorMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(processorID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := processorhelper.NewObsReport(processorhelper.ObsReportSettings{
		ProcessorID:             processorID,
		ProcessorCreateSettings: processor.CreateSettings{ID: processorID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	assert.NoError(t, err)

	por.MetricsAccepted(context.Background(), 7)
	por.MetricsRefused(context.Background(), 8)
	por.MetricsDropped(context.Background(), 9)

	assert.NoError(t, tt.CheckProcessorMetrics(7, 8, 9))
	assert.Error(t, tt.CheckProcessorMetrics(0, 0, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 0, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 8, 0))
	assert.Error(t, tt.CheckProcessorMetrics(7, 0, 9))
	assert.Error(t, tt.CheckProcessorMetrics(0, 8, 0))
	assert.Error(t, tt.CheckProcessorMetrics(0, 8, 9))
	assert.Error(t, tt.CheckProcessorMetrics(0, 0, 9))
}

func TestCheckProcessorLogViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(processorID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := processorhelper.NewObsReport(processorhelper.ObsReportSettings{
		ProcessorID:             processorID,
		ProcessorCreateSettings: processor.CreateSettings{ID: processorID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	assert.NoError(t, err)

	por.LogsAccepted(context.Background(), 7)
	por.LogsRefused(context.Background(), 8)
	por.LogsDropped(context.Background(), 9)

	assert.NoError(t, tt.CheckProcessorLogs(7, 8, 9))
	assert.Error(t, tt.CheckProcessorLogs(0, 0, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 0, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 8, 0))
	assert.Error(t, tt.CheckProcessorLogs(7, 0, 9))
	assert.Error(t, tt.CheckProcessorLogs(0, 8, 0))
	assert.Error(t, tt.CheckProcessorLogs(0, 8, 9))
	assert.Error(t, tt.CheckProcessorLogs(0, 0, 9))
}

func TestCheckExporterTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := exporterhelper.NewObsReport(exporterhelper.ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := obsrep.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndTracesOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckExporterTraces(7, 0))
	assert.Error(t, tt.CheckExporterTraces(7, 7))
	assert.Error(t, tt.CheckExporterTraces(0, 0))
	assert.Error(t, tt.CheckExporterTraces(0, 7))
}

func TestCheckExporterMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := exporterhelper.NewObsReport(exporterhelper.ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := obsrep.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndMetricsOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckExporterMetrics(7, 0))
	assert.Error(t, tt.CheckExporterMetrics(7, 7))
	assert.Error(t, tt.CheckExporterMetrics(0, 0))
	assert.Error(t, tt.CheckExporterMetrics(0, 7))
}

func TestCheckExporterLogsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(exporterID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := exporterhelper.NewObsReport(exporterhelper.ObsReportSettings{
		ExporterID:             exporterID,
		ExporterCreateSettings: exporter.CreateSettings{ID: exporterID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()},
	})
	require.NoError(t, err)
	ctx := obsrep.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndLogsOp(ctx, 7, nil)

	assert.NoError(t, tt.CheckExporterLogs(7, 0))
	assert.Error(t, tt.CheckExporterLogs(7, 7))
	assert.Error(t, tt.CheckExporterLogs(0, 0))
	assert.Error(t, tt.CheckExporterLogs(0, 7))
}
