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

package connectortest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestFanoutTracesWithNop(t *testing.T) {
	tr, err := NewTracesRouterSink(
		WithNopTracesSink(component.NewIDWithName(component.DataTypeTraces, "0")),
		WithNopTracesSink(component.NewIDWithName(component.DataTypeTraces, "1")),
	)

	require.NoError(t, err)

	td := testdata.GenerateTraces(1)
	err = tr.(consumer.Traces).ConsumeTraces(context.Background(), td)

	require.NoError(t, err)
}

func TestFanoutTracesWithSink(t *testing.T) {
	var sink0, sink1 consumertest.TracesSink

	tr, err := NewTracesRouterSink(
		WithTracesSink(component.NewIDWithName(component.DataTypeTraces, "0"), &sink0),
		WithTracesSink(component.NewIDWithName(component.DataTypeTraces, "1"), &sink1),
	)

	require.NoError(t, err)
	require.Equal(t, 0, sink0.SpanCount())
	require.Equal(t, 0, sink1.SpanCount())

	td := testdata.GenerateTraces(1)
	err = tr.(consumer.Traces).ConsumeTraces(context.Background(), td)

	require.NoError(t, err)
	require.Equal(t, 1, sink0.SpanCount())
	require.Equal(t, 1, sink1.SpanCount())
}

func TestFanoutTracesErr(t *testing.T) {
	tr, err := NewTracesRouterSink(
		WithNopTracesSink(component.NewIDWithName(component.DataTypeTraces, "0")),
	)

	require.Nil(t, tr)
	require.Error(t, err)
	require.ErrorIs(t, err, errTooFewConsumers)
}

func TestFanoutMetricsWithNop(t *testing.T) {
	mr, err := NewMetricsRouterSink(
		WithNopMetricsSink(component.NewIDWithName(component.DataTypeMetrics, "0")),
		WithNopMetricsSink(component.NewIDWithName(component.DataTypeMetrics, "1")),
	)

	require.NoError(t, err)

	md := testdata.GenerateMetrics(1)
	err = mr.(consumer.Metrics).ConsumeMetrics(context.Background(), md)

	require.NoError(t, err)
}

func TestFanoutMetricsWithSink(t *testing.T) {
	var sink0, sink1 consumertest.MetricsSink

	mr, err := NewMetricsRouterSink(
		WithMetricsSink(component.NewIDWithName(component.DataTypeMetrics, "0"), &sink0),
		WithMetricsSink(component.NewIDWithName(component.DataTypeMetrics, "1"), &sink1),
	)

	require.NoError(t, err)
	require.Len(t, sink0.AllMetrics(), 0)
	require.Len(t, sink1.AllMetrics(), 0)

	md := testdata.GenerateMetrics(1)
	err = mr.(consumer.Metrics).ConsumeMetrics(context.Background(), md)

	require.NoError(t, err)
	require.Len(t, sink0.AllMetrics(), 1)
	require.Len(t, sink1.AllMetrics(), 1)
}

func TestFanoutMetricsErr(t *testing.T) {
	mr, err := NewMetricsRouterSink(
		WithNopMetricsSink(component.NewIDWithName(component.DataTypeMetrics, "0")),
	)

	require.Nil(t, mr)
	require.Error(t, err)
	require.ErrorIs(t, err, errTooFewConsumers)
}

func TestFanoutLogsWithNop(t *testing.T) {
	lr, err := NewLogsRouterSink(
		WithNopLogsSink(component.NewIDWithName(component.DataTypeLogs, "0")),
		WithNopLogsSink(component.NewIDWithName(component.DataTypeLogs, "1")),
	)

	require.NoError(t, err)

	ld := testdata.GenerateLogs(1)
	err = lr.(consumer.Logs).ConsumeLogs(context.Background(), ld)

	require.NoError(t, err)
}

func TestFanoutLogsWithSink(t *testing.T) {
	var sink0, sink1 consumertest.LogsSink

	lr, err := NewLogsRouterSink(
		WithLogsSink(component.NewIDWithName(component.DataTypeLogs, "0"), &sink0),
		WithLogsSink(component.NewIDWithName(component.DataTypeLogs, "1"), &sink1),
	)

	require.NoError(t, err)
	require.Equal(t, 0, sink0.LogRecordCount())
	require.Equal(t, 0, sink1.LogRecordCount())

	ld := testdata.GenerateLogs(1)
	err = lr.(consumer.Logs).ConsumeLogs(context.Background(), ld)

	require.NoError(t, err)
	require.Equal(t, 1, sink0.LogRecordCount())
	require.Equal(t, 1, sink1.LogRecordCount())
}

func TestFanoutLogsErr(t *testing.T) {
	lr, err := NewLogsRouterSink(
		WithNopLogsSink(component.NewIDWithName(component.DataTypeLogs, "0")),
	)

	require.Nil(t, lr)
	require.Error(t, err)
	require.ErrorIs(t, err, errTooFewConsumers)
}
