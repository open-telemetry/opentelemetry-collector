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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

const (
	transport = "fakeTransport"
	format    = "fakeFormat"
)

var (
	receiver = config.NewComponentID("fakeReicever")
	exporter = config.NewComponentID("fakeExporter")
)

func TestCheckReceiverTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	rec := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             receiver,
		Transport:              transport,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
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

	rec := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             receiver,
		Transport:              transport,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
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

	rec := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             receiver,
		Transport:              transport,
		ReceiverCreateSettings: tt.ToReceiverCreateSettings(),
	})
	ctx := rec.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	rec.EndLogsOp(ctx, format, 7, nil)

	assert.NoError(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 7, 0))
	assert.Error(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 7, 7))
	assert.Error(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 0, 0))
	assert.Error(t, obsreporttest.CheckReceiverLogs(tt, receiver, transport, 0, 7))
}

func TestCheckExporterTracesViews(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	obsrep := obsreport.NewExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	})
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

	obsrep := obsreport.NewExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	})
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

	obsrep := obsreport.NewExporter(obsreport.ExporterSettings{
		ExporterID:             exporter,
		ExporterCreateSettings: tt.ToExporterCreateSettings(),
	})
	ctx := obsrep.StartLogsOp(context.Background())
	require.NotNil(t, ctx)
	obsrep.EndLogsOp(ctx, 7, nil)

	assert.NoError(t, obsreporttest.CheckExporterLogs(tt, exporter, 7, 0))
	assert.Error(t, obsreporttest.CheckExporterLogs(tt, exporter, 7, 7))
	assert.Error(t, obsreporttest.CheckExporterLogs(tt, exporter, 0, 0))
	assert.Error(t, obsreporttest.CheckExporterLogs(tt, exporter, 0, 7))
}
