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
package countconnector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestCount(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig()
	assert.Equal(t, &Config{}, cfg)

	ctx := context.Background()
	set := componenttest.NewNopConnectorCreateSettings()
	host := componenttest.NewNopHost()

	countsSink := new(consumertest.MetricsSink)

	spansCounter, err := f.CreateTracesToMetricsConnector(ctx, set, cfg, countsSink)
	assert.NoError(t, err)
	assert.NotNil(t, spansCounter)

	metricsCounter, err := f.CreateMetricsToMetricsConnector(ctx, set, cfg, countsSink)
	assert.NoError(t, err)
	assert.NotNil(t, metricsCounter)

	logsCounter, err := f.CreateLogsToMetricsConnector(ctx, set, cfg, countsSink)
	assert.NoError(t, err)
	assert.NotNil(t, logsCounter)

	assert.NoError(t, spansCounter.Start(ctx, host))
	assert.NoError(t, metricsCounter.Start(ctx, host))
	assert.NoError(t, logsCounter.Start(ctx, host))

	spanCounts := []int64{4, 8, 2, 25, 100}
	assert.NoError(t, spansCounter.ConsumeTracesToMetrics(ctx, dummySpans(spanCounts)))

	metricCounts := []int64{2, 5, 1, 10}
	assert.NoError(t, metricsCounter.ConsumeMetricsToMetrics(ctx, dummyMetrics(metricCounts)))

	logCounts := []int64{1, 1, 2, 3, 5, 8, 13}
	assert.NoError(t, logsCounter.ConsumeLogsToMetrics(ctx, dummyLogs(logCounts)))

	assert.NoError(t, spansCounter.Shutdown(ctx))
	assert.NoError(t, metricsCounter.Shutdown(ctx))
	assert.NoError(t, logsCounter.Shutdown(ctx))

	assert.Equal(t, 3, len(countsSink.AllMetrics()))
	assertEqualCounts(t, spanCounts, "span.count", countsSink.AllMetrics()[0].ResourceMetrics())
	assertEqualCounts(t, metricCounts, "metric.count", countsSink.AllMetrics()[1].ResourceMetrics())
	assertEqualCounts(t, logCounts, "log.count", countsSink.AllMetrics()[2].ResourceMetrics())

}

func assertEqualCounts(t *testing.T, expectedCounts []int64, expectedName string, resources pmetric.ResourceMetricsSlice) {
	for i, c := range expectedCounts {
		scopeMetric := resources.At(i).ScopeMetrics().At(0)
		assert.Equal(t, scopeName, scopeMetric.Scope().Name())
		assert.Equal(t, 1, scopeMetric.Metrics().Len())
		countMetric := scopeMetric.Metrics().At(0)
		assert.Equal(t, expectedName, countMetric.Name())
		assert.Equal(t, 1, countMetric.Sum().DataPoints().Len())
		countDataPoint := countMetric.Sum().DataPoints().At(0)
		assert.Equal(t, c, countDataPoint.IntValue())
	}
}

func dummySpans(spanCounts []int64) ptrace.Traces {
	result := ptrace.NewTraces()
	resources := result.ResourceSpans()
	for i := 0; i < len(spanCounts); i++ {
		resource := resources.AppendEmpty()
		spans := resource.ScopeSpans().AppendEmpty().Spans()
		for j := int64(0); j < spanCounts[i]; j++ {
			spans.AppendEmpty().SetName(fmt.Sprintf("some.span.%d", j))
		}
	}
	return result
}

func dummyMetrics(metricCounts []int64) pmetric.Metrics {
	result := pmetric.NewMetrics()
	resources := result.ResourceMetrics()
	for i := 0; i < len(metricCounts); i++ {
		resource := resources.AppendEmpty()
		metrics := resource.ScopeMetrics().AppendEmpty().Metrics()
		for j := int64(0); j < metricCounts[i]; j++ {
			metrics.AppendEmpty().SetName(fmt.Sprintf("some.metric.%d", j))
		}
	}
	return result
}

func dummyLogs(logCounts []int64) plog.Logs {
	result := plog.NewLogs()
	resources := result.ResourceLogs()
	for i := 0; i < len(logCounts); i++ {
		resource := resources.AppendEmpty()
		logs := resource.ScopeLogs().AppendEmpty().LogRecords()
		for j := int64(0); j < logCounts[i]; j++ {
			logs.AppendEmpty().Body().SetStr(fmt.Sprintf("some log %d", j))
		}
	}
	return result
}
