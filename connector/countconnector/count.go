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

package countconnector // import "go.opentelemetry.io/collector/connector/countconnector"

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	typeStr   = "count"
	scopeName = "otelcol/countconnector"
)

type countFactory struct {
	// This is the map of already created count connectors for particular configurations.
	// We maintain this map because the Factory is asked trace, metric, and log receivers
	// separately but they must not create separate objects. When the connector is shutdown
	// it should be removed from this map so the same configuration can be recreated successfully.
	*sharedcomponent.SharedComponents
}

// NewFactory returns a ConnectorFactory.
func NewFactory() connector.Factory {
	f := &countFactory{sharedcomponent.NewSharedComponents()}
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToMetrics(f.createTracesToMetrics, component.StabilityLevelDevelopment),
		connector.WithMetricsToMetrics(f.createMetricsToMetrics, component.StabilityLevelDevelopment),
		connector.WithLogsToMetrics(f.createLogsToMetrics, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

// createTracesToMetrics creates a traces to metrics connector based on provided config.
func (f *countFactory) createTracesToMetrics(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Traces, error) {
	comp := f.GetOrAdd(cfg, func() component.Component {
		return &count{}
	})

	conn := comp.Unwrap().(*count)
	conn.Metrics = nextConsumer
	return conn, nil
}

// createMetricsToMetrics creates a metrics connector based on provided config.
func (f *countFactory) createMetricsToMetrics(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Metrics, error) {
	comp := f.GetOrAdd(cfg, func() component.Component {
		return &count{}
	})

	conn := comp.Unwrap().(*count)
	conn.Metrics = nextConsumer
	return conn, nil
}

// createLogsToMetrics creates a logs to metrics connector based on provided config.
func (f *countFactory) createLogsToMetrics(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Logs, error) {
	comp := f.GetOrAdd(cfg, func() component.Component {
		return &count{}
	})

	conn := comp.Unwrap().(*count)
	conn.Metrics = nextConsumer
	return conn, nil
}

// count can count spans, data points, or log records and emit
// the count onto a metrics pipeline.
type count struct {
	consumer.Metrics
	component.StartFunc
	component.ShutdownFunc
}

func (c *count) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *count) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	countMetrics := pmetric.NewMetrics()
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)
		countResource := countMetrics.ResourceMetrics().AppendEmpty()
		resourceSpan.Resource().Attributes().CopyTo(countResource.Resource().Attributes())

		countScope := countResource.ScopeMetrics().AppendEmpty()
		countScope.Scope().SetName(scopeName)

		count := 0
		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			count += resourceSpan.ScopeSpans().At(j).Spans().Len()
		}
		setCountMetric(countScope.Metrics().AppendEmpty(), "span", count)
	}
	return c.Metrics.ConsumeMetrics(ctx, countMetrics)
}

func (c *count) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	countMetrics := pmetric.NewMetrics()
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resourceMetric := md.ResourceMetrics().At(i)
		countResource := countMetrics.ResourceMetrics().AppendEmpty()
		resourceMetric.Resource().Attributes().CopyTo(countResource.Resource().Attributes())

		countScope := countResource.ScopeMetrics().AppendEmpty()
		countScope.Scope().SetName(scopeName)

		count := 0
		for j := 0; j < resourceMetric.ScopeMetrics().Len(); j++ {
			count += resourceMetric.ScopeMetrics().At(j).Metrics().Len()
		}
		setCountMetric(countScope.Metrics().AppendEmpty(), "metric", count)
	}
	return c.Metrics.ConsumeMetrics(ctx, countMetrics)
}

func (c *count) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	countMetrics := pmetric.NewMetrics()
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		countResource := countMetrics.ResourceMetrics().AppendEmpty()
		resourceLog.Resource().Attributes().CopyTo(countResource.Resource().Attributes())

		countScope := countResource.ScopeMetrics().AppendEmpty()
		countScope.Scope().SetName(scopeName)

		count := 0
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			count += resourceLog.ScopeLogs().At(j).LogRecords().Len()
		}
		setCountMetric(countScope.Metrics().AppendEmpty(), "log", count)
	}
	return c.Metrics.ConsumeMetrics(ctx, countMetrics)
}

func setCountMetric(countMetric pmetric.Metric, signalType string, count int) {
	countMetric.SetName(fmt.Sprintf("%s.count", signalType))
	countMetric.SetDescription(fmt.Sprintf("The number of %s observed.", signalType))
	sum := countMetric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	dp := sum.DataPoints().AppendEmpty()
	dp.SetIntValue(int64(count))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
}
