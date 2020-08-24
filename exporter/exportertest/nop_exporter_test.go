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
package exportertest

import (
	"context"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
)

func TestNopTraceExporterOld(t *testing.T) {
	nte := NewNopTraceExporterOld()
	require.NoError(t, nte.Start(context.Background(), nil))
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
	}
	require.NoError(t, nte.ConsumeTraceData(context.Background(), td))
	require.NoError(t, nte.Shutdown(context.Background()))
}

func TestNopMetricsExporterOld(t *testing.T) {
	nme := NewNopMetricsExporterOld()
	require.NoError(t, nme.Start(context.Background(), nil))
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}
	require.NoError(t, nme.ConsumeMetricsData(context.Background(), md))
	require.NoError(t, nme.Shutdown(context.Background()))
}

func TestNopTraceExporter(t *testing.T) {
	nte := NewNopTraceExporter()
	require.NoError(t, nte.Start(context.Background(), nil))
	require.NoError(t, nte.ConsumeTraces(context.Background(), pdata.NewTraces()))
	require.NoError(t, nte.Shutdown(context.Background()))
}

func TestNopMetricsExporter(t *testing.T) {
	nme := NewNopMetricsExporter()
	require.NoError(t, nme.Start(context.Background(), nil))
	require.NoError(t, nme.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(data.NewMetricData())))
	require.NoError(t, nme.Shutdown(context.Background()))
}

func TestNopLogsExporter(t *testing.T) {
	nme := NewNopLogsExporter()
	require.NoError(t, nme.Start(context.Background(), nil))
	require.NoError(t, nme.ConsumeLogs(context.Background(), pdata.NewLogs()))
	require.NoError(t, nme.Shutdown(context.Background()))
}
