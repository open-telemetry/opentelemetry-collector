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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewNopConnectorFactory(t *testing.T) {
	factory := NewNopFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopConfig{}, cfg)

	tracesToTraces, err := factory.CreateTracesToTraces(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToTraces.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToTraces.Shutdown(context.Background()))

	tracesToMetrics, err := factory.CreateTracesToMetrics(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToMetrics.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToMetrics.Shutdown(context.Background()))

	tracesToLogs, err := factory.CreateTracesToLogs(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToLogs.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToLogs.Shutdown(context.Background()))

	metricsToTraces, err := factory.CreateMetricsToTraces(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToTraces.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToTraces.Shutdown(context.Background()))

	metricsToMetrics, err := factory.CreateMetricsToMetrics(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToMetrics.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToMetrics.Shutdown(context.Background()))

	metricsToLogs, err := factory.CreateMetricsToLogs(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToLogs.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToLogs.Shutdown(context.Background()))

	logsToTraces, err := factory.CreateLogsToTraces(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToTraces.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToTraces.Shutdown(context.Background()))

	logsToMetrics, err := factory.CreateLogsToMetrics(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToMetrics.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToMetrics.Shutdown(context.Background()))

	logsToLogs, err := factory.CreateLogsToLogs(context.Background(), NewNopCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToLogs.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToLogs.Shutdown(context.Background()))
}

func TestNewNopBuilder(t *testing.T) {
	builder := NewNopBuilder()
	require.NotNil(t, builder)

	factory := NewNopFactory()
	cfg := factory.CreateDefaultConfig()
	set := NewNopCreateSettings()
	set.ID = component.NewIDWithName(typeStr, "conn")

	tracesToTraces, err := factory.CreateTracesToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToTraces, err := builder.CreateTracesToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToTraces, bTracesToTraces)

	tracesToMetrics, err := factory.CreateTracesToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToMetrics, err := builder.CreateTracesToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToMetrics, bTracesToMetrics)

	tracesToLogs, err := factory.CreateTracesToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bTracesToLogs, err := builder.CreateTracesToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, tracesToLogs, bTracesToLogs)

	metricsToTraces, err := factory.CreateMetricsToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToTraces, err := builder.CreateMetricsToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToTraces, bMetricsToTraces)

	metricsToMetrics, err := factory.CreateMetricsToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToMetrics, err := builder.CreateMetricsToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToMetrics, bMetricsToMetrics)

	metricsToLogs, err := factory.CreateMetricsToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bMetricsToLogs, err := builder.CreateMetricsToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, metricsToLogs, bMetricsToLogs)

	logsToTraces, err := factory.CreateLogsToTraces(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToTraces, err := builder.CreateLogsToTraces(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToTraces, bLogsToTraces)

	logsToMetrics, err := factory.CreateLogsToMetrics(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToMetrics, err := builder.CreateLogsToMetrics(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToMetrics, bLogsToMetrics)

	logsToLogs, err := factory.CreateLogsToLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	bLogsToLogs, err := builder.CreateLogsToLogs(context.Background(), set, consumertest.NewNop())
	require.NoError(t, err)
	assert.IsType(t, logsToLogs, bLogsToLogs)
}
