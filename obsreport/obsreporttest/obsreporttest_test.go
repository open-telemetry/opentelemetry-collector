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

package obsreporttest_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	scraper  = component.NewID("fakeScraper")
	receiver = component.NewID("fakeReicever")
	processor= component.NewID("fakeProcessor")
	exporter = component.NewID("fakeExporter")
)

func TestCheckScraperMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	s, err := obsreport.NewScraper(obsreport.ScraperSettings{
		ReceiverID:             receiver,
		Scraper:                scraper,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
	require.NoError(t, err)
	ctx := s.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	s.EndMetricsOp(ctx, 7, nil)

	assert.NoError(t, obsreporttest.CheckScraperMetrics(tt, receiver, scraper, 7, 0))
	assert.Error(t, obsreporttest.CheckScraperMetrics(tt, receiver, scraper, 7, 7))
	assert.Error(t, obsreporttest.CheckScraperMetrics(tt, receiver, scraper, 0, 0))
	assert.Error(t, obsreporttest.CheckScraperMetrics(tt, receiver, scraper, 0, 7))
}

func TestCheckReceiverTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             receiver,
		Transport:              transport,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
	require.NoError(t, err)
	ctx := rec.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndTracesOp(ctx, format, 7, nil)

	assert.NoError(t, obsreporttest.CheckReceiverTraces(tt, receiver, transport, 7, 0))
	assert.Error(t, obsreporttest.CheckReceiverTraces(tt, receiver, transport, 7, 7))
	assert.Error(t, obsreporttest.CheckReceiverTraces(tt, receiver, transport, 0, 0))
	assert.Error(t, obsreporttest.CheckReceiverTraces(tt, receiver, transport, 0, 7))
}

func TestCheckReceiverMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             receiver,
		Transport:              transport,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
	require.NoError(t, err)
	ctx := rec.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndMetricsOp(ctx, format, 7, nil)

	assert.NoError(t, obsreporttest.CheckReceiverMetrics(tt, receiver, transport, 7, 0))
	assert.Error(t, obsreporttest.CheckReceiverMetrics(tt, receiver, transport, 7, 7))
	assert.Error(t, obsreporttest.CheckReceiverMetrics(tt, receiver, transport, 0, 0))
	assert.Error(t, obsreporttest.CheckReceiverMetrics(tt, receiver, transport, 0, 7))
}

func TestCheckReceiverLogsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             receiver,
		Transport:              transport,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
	require.NoError(t, err)
	ctx := rec.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndLogsOp(ctx, format, 7, nil)

	assert.NoError(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 7, 0))
	assert.Error(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 7, 7))
	assert.Error(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 0, 0))
	assert.Error(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 0, 7))
}

func TestCheckProcessorTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := obsreport.NewProcessor(obsreport.ProcessorSettings{
		ProcessorID:             processor,
		ProcessorCreateSettings: tt.ToProcessorCreateSettings(),
	})
	assert.NoError(t, err)

	por.TracesAccepted(context.Background(), 7)
	por.TracesRefused(context.Background(), 8)
	por.TracesDropped(context.Background(), 9)

	assert.NoError(t, obsreporttest.CheckProcessorTraces(tt, processor, 7, 8, 9))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 0, 0, 0))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 7, 0, 0))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 7, 8, 0))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 7, 0, 9))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 0, 8, 0))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 0, 8, 9))
	assert.Error(t, obsreporttest.CheckProcessorTraces(tt, processor, 0, 0, 9))
}

func TestCheckProcessorMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := obsreport.NewProcessor(obsreport.ProcessorSettings{
		ProcessorID:             processor,
		ProcessorCreateSettings: tt.ToProcessorCreateSettings(),
	})
	assert.NoError(t, err)

	por.MetricsAccepted(context.Background(), 7)
	por.MetricsRefused(context.Background(), 8)
	por.MetricsDropped(context.Background(), 9)

	assert.NoError(t, obsreporttest.CheckProcessorMetrics(tt, processor, 7, 8, 9))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 0, 0, 0))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 7, 0, 0))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 7, 8, 0))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 7, 0, 9))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 0, 8, 0))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 0, 8, 9))
	assert.Error(t, obsreporttest.CheckProcessorMetrics(tt, processor, 0, 0, 9))
}

func TestCheckProcessorLogViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	por, err := obsreport.NewProcessor(obsreport.ProcessorSettings{
		ProcessorID:             processor,
		ProcessorCreateSettings: tt.ToProcessorCreateSettings(),
	})
	assert.NoError(t, err)

	por.LogsAccepted(context.Background(), 7)
	por.LogsRefused(context.Background(), 8)
	por.LogsDropped(context.Background(), 9)

	assert.NoError(t, obsreporttest.CheckProcessorLogs(tt, processor, 7, 8, 9))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 0, 0, 0))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 7, 0, 0))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 7, 8, 0))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 7, 0, 9))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 0, 8, 0))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 0, 8, 9))
	assert.Error(t, obsreporttest.CheckProcessorLogs(tt, processor, 0, 0, 9))
}

func TestCheckExporterTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := obsreport.NewExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	})
	require.NoError(t, err)
	ctx := obsrep.StartTracesOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndTracesOp(ctx, 7, nil)

	assert.NoError(t, obsreporttest.CheckExporterTraces(tt, exporter, 7, 0))
	assert.Error(t, obsreporttest.CheckExporterTraces(tt, exporter, 7, 7))
	assert.Error(t, obsreporttest.CheckExporterTraces(tt, exporter, 0, 0))
	assert.Error(t, obsreporttest.CheckExporterTraces(tt, exporter, 0, 7))
}

func TestCheckExporterMetricsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := obsreport.NewExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	})
	require.NoError(t, err)
	ctx := obsrep.StartMetricsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndMetricsOp(ctx, 7, nil)

	assert.NoError(t, obsreporttest.CheckExporterMetrics(tt, exporter, 7, 0))
	assert.Error(t, obsreporttest.CheckExporterMetrics(tt, exporter, 7, 7))
	assert.Error(t, obsreporttest.CheckExporterMetrics(tt, exporter, 0, 0))
	assert.Error(t, obsreporttest.CheckExporterMetrics(tt, exporter, 0, 7))
}

func TestCheckExporterLogsViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep, err := obsreport.NewExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	})
	require.NoError(t, err)
	ctx := obsrep.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndLogsOp(ctx, 7, nil)

	assert.NoError(t, obsreporttest.CheckExporterLogs(tt, exporter, 7, 0))
	assert.Error(t, obsreporttest.CheckExporterLogs(tt, exporter, 7, 7))
	assert.Error(t, obsreporttest.CheckExporterLogs(tt, exporter, 0, 0))
	assert.Error(t, obsreporttest.CheckExporterLogs(tt, exporter, 0, 7))
}
