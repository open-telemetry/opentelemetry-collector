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
	"go.opentelemetry.io/collector/config"
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

type Config struct {
	config.ConnectorSettings `mapstructure:",squash"`
}

var _ component.Config = (*Config)(nil)

// NewFactory returns a ConnectorFactory.
func NewFactory() component.ConnectorFactory {
	return component.NewConnectorFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesToMetricsConnector(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithMetricsToMetricsConnector(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithLogsToMetricsConnector(createLogsToMetricsConnector, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createTracesToMetricsConnector creates a traces to metrics connector based on provided config.
func createTracesToMetricsConnector(
	_ context.Context,
	set component.ConnectorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (component.TracesToMetricsConnector, error) {
	comp := connectors.GetOrAdd(cfg, func() component.Component {
		return newCountConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*countConnector)
	conn.metricsConsumer = nextConsumer
	return conn, nil
}

// createMetricsConnector creates a metrics connector based on provided config.
func createMetricsToMetricsConnector(
	_ context.Context,
	set component.ConnectorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (component.MetricsToMetricsConnector, error) {
	comp := connectors.GetOrAdd(cfg, func() component.Component {
		return newCountConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*countConnector)
	conn.metricsConsumer = nextConsumer
	return conn, nil
}

// createLogsToMetricsConnector creates a logs to metrics connector based on provided config.
func createLogsToMetricsConnector(
	_ context.Context,
	set component.ConnectorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (component.LogsToMetricsConnector, error) {
	comp := connectors.GetOrAdd(cfg, func() component.Component {
		return newCountConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*countConnector)
	conn.metricsConsumer = nextConsumer
	return conn, nil
}

// This is the map of already created count connectors for particular configurations.
// We maintain this map because the Factory is asked trace, metric, and log receivers
// separately but they must not create separate objects. When the connector is shutdown
// it should be removed from this map so the same configuration can be recreated successfully.
var connectors = sharedcomponent.NewSharedComponents()

// countConnector can count spans, data points, or log records and emit
// the count onto a metrics pipeline.
type countConnector struct {
	cfg             *Config
	settings        component.ConnectorCreateSettings
	metricsConsumer consumer.Metrics
}

func newCountConnector(cfg *Config, settings component.ConnectorCreateSettings) *countConnector {
	return &countConnector{
		cfg:      cfg,
		settings: settings,
	}
}

func (c *countConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *countConnector) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (c *countConnector) Shutdown(ctx context.Context) error {
	return nil
}

func (c *countConnector) ConsumeTracesToMetrics(ctx context.Context, td ptrace.Traces) error {
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
	return c.metricsConsumer.ConsumeMetrics(ctx, countMetrics)
}

func (c *countConnector) ConsumeMetricsToMetrics(ctx context.Context, md pmetric.Metrics) error {
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
	return c.metricsConsumer.ConsumeMetrics(ctx, countMetrics)
}

func (c *countConnector) ConsumeLogsToMetrics(ctx context.Context, ld plog.Logs) error {
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
	return c.metricsConsumer.ConsumeMetrics(ctx, countMetrics)
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
