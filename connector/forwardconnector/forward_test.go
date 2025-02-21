// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package forwardconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestForward(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig()
	assert.Equal(t, &Config{}, cfg)

	ctx := context.Background()
	set := connectortest.NewNopSettings(f.Type())
	host := componenttest.NewNopHost()

	tracesSink := new(consumertest.TracesSink)
	tracesToTraces, err := f.CreateTracesToTraces(ctx, set, cfg, tracesSink)
	require.NoError(t, err)
	assert.NotNil(t, tracesToTraces)

	metricsSink := new(consumertest.MetricsSink)
	metricsToMetrics, err := f.CreateMetricsToMetrics(ctx, set, cfg, metricsSink)
	require.NoError(t, err)
	assert.NotNil(t, metricsToMetrics)

	logsSink := new(consumertest.LogsSink)
	logsToLogs, err := f.CreateLogsToLogs(ctx, set, cfg, logsSink)
	require.NoError(t, err)
	assert.NotNil(t, logsToLogs)

	assert.NoError(t, tracesToTraces.Start(ctx, host))
	assert.NoError(t, metricsToMetrics.Start(ctx, host))
	assert.NoError(t, logsToLogs.Start(ctx, host))

	assert.NoError(t, tracesToTraces.ConsumeTraces(ctx, ptrace.NewTraces()))

	assert.NoError(t, metricsToMetrics.ConsumeMetrics(ctx, pmetric.NewMetrics()))
	assert.NoError(t, metricsToMetrics.ConsumeMetrics(ctx, pmetric.NewMetrics()))

	assert.NoError(t, logsToLogs.ConsumeLogs(ctx, plog.NewLogs()))
	assert.NoError(t, logsToLogs.ConsumeLogs(ctx, plog.NewLogs()))
	assert.NoError(t, logsToLogs.ConsumeLogs(ctx, plog.NewLogs()))

	assert.NoError(t, tracesToTraces.Shutdown(ctx))
	assert.NoError(t, metricsToMetrics.Shutdown(ctx))
	assert.NoError(t, logsToLogs.Shutdown(ctx))

	assert.Len(t, tracesSink.AllTraces(), 1)
	assert.Len(t, metricsSink.AllMetrics(), 2)
	assert.Len(t, logsSink.AllLogs(), 3)
}
