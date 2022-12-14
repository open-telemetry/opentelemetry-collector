// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package forwardconnector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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
	assert.Equal(t, &struct{}{}, cfg)

	ctx := context.Background()
	set := connectortest.NewNopCreateSettings()
	host := componenttest.NewNopHost()

	tracesSink := new(consumertest.TracesSink)
	tracesToTraces, err := f.CreateTracesToTraces(ctx, set, cfg, tracesSink)
	assert.NoError(t, err)
	assert.NotNil(t, tracesToTraces)

	metricsSink := new(consumertest.MetricsSink)
	metricsToMetrics, err := f.CreateMetricsToMetrics(ctx, set, cfg, metricsSink)
	assert.NoError(t, err)
	assert.NotNil(t, metricsToMetrics)

	logsSink := new(consumertest.LogsSink)
	logsToLogs, err := f.CreateLogsToLogs(ctx, set, cfg, logsSink)
	assert.NoError(t, err)
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

	assert.Equal(t, 1, len(tracesSink.AllTraces()))
	assert.Equal(t, 2, len(metricsSink.AllMetrics()))
	assert.Equal(t, 3, len(logsSink.AllLogs()))
}
