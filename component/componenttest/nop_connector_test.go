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

package componenttest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewNopConnectorFactory(t *testing.T) {
	factory := NewNopConnectorFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopConnectorConfig{ConnectorSettings: config.NewConnectorSettings(component.NewID("nop"))}, cfg)

	tracesToTraces, err := factory.CreateTracesToTracesConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToTraces.Start(context.Background(), NewNopHost()))
	assert.NoError(t, tracesToTraces.ConsumeTracesToTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToTraces.Shutdown(context.Background()))

	tracesToMetrics, err := factory.CreateTracesToMetricsConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToMetrics.Start(context.Background(), NewNopHost()))
	assert.NoError(t, tracesToMetrics.ConsumeTracesToMetrics(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToMetrics.Shutdown(context.Background()))

	tracesToLogs, err := factory.CreateTracesToLogsConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToLogs.Start(context.Background(), NewNopHost()))
	assert.NoError(t, tracesToLogs.ConsumeTracesToLogs(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToLogs.Shutdown(context.Background()))

	metricsToTraces, err := factory.CreateMetricsToTracesConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToTraces.Start(context.Background(), NewNopHost()))
	assert.NoError(t, metricsToTraces.ConsumeMetricsToTraces(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToTraces.Shutdown(context.Background()))

	metricsToMetrics, err := factory.CreateMetricsToMetricsConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToMetrics.Start(context.Background(), NewNopHost()))
	assert.NoError(t, metricsToMetrics.ConsumeMetricsToMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToMetrics.Shutdown(context.Background()))

	metricsToLogs, err := factory.CreateMetricsToLogsConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToLogs.Start(context.Background(), NewNopHost()))
	assert.NoError(t, metricsToLogs.ConsumeMetricsToLogs(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToLogs.Shutdown(context.Background()))

	logsToTraces, err := factory.CreateLogsToTracesConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToTraces.Start(context.Background(), NewNopHost()))
	assert.NoError(t, logsToTraces.ConsumeLogsToTraces(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToTraces.Shutdown(context.Background()))

	logsToMetrics, err := factory.CreateLogsToMetricsConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToMetrics.Start(context.Background(), NewNopHost()))
	assert.NoError(t, logsToMetrics.ConsumeLogsToMetrics(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToMetrics.Shutdown(context.Background()))

	logsToLogs, err := factory.CreateLogsToLogsConnector(context.Background(), NewNopConnectorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToLogs.Start(context.Background(), NewNopHost()))
	assert.NoError(t, logsToLogs.ConsumeLogsToLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToLogs.Shutdown(context.Background()))
}
