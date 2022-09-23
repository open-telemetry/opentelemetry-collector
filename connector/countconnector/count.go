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
	typeStr = "count"
)

type Config struct {
	config.ConnectorSettings `mapstructure:",squash"`
}

var _ config.Connector = (*Config)(nil)

// NewFactory returns a ConnectorFactory.
func NewFactory() component.ConnectorFactory {
	return component.NewConnectorFactory(
		typeStr,
		createDefaultConfig,
		[]component.ExporterFactoryOption{
			component.WithTracesExporter(createTracesExporter, component.StabilityLevelInDevelopment),
			component.WithMetricsExporter(createMetricsExporter, component.StabilityLevelInDevelopment),
			component.WithLogsExporter(createLogExporter, component.StabilityLevelInDevelopment),
		},
		[]component.ReceiverFactoryOption{
			component.WithMetricsReceiver(createMetricsReceiver, component.StabilityLevelInDevelopment),
		},
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() config.Connector {
	return &Config{}
}

// createTracesExporter creates a trace receiver based on provided config.
func createTracesExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.TracesExporter, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		return newCountConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*countConnector)
	return conn, nil
}

// createMetricsExporter creates a metrics receiver based on provided config.
func createMetricsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		return newCountConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*countConnector)
	return conn, nil
}

// createLogExporter creates a log receiver based on provided config.
func createLogExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.LogsExporter, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		return newCountConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*countConnector)
	return conn, nil
}

// createMetricsReceiver creates a metrics receiver based on provided config.
func createMetricsReceiver(
	_ context.Context,
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Metrics,
) (component.MetricsReceiver, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		// Expect to have already created this component as an exporter
		return nil
	})

	if comp == nil {
		return nil, fmt.Errorf("connector must be initialized as exporter and receiver")
	}

	conn := comp.Unwrap().(*countConnector)
	conn.metricsConsumer = nextConsumer
	return conn, nil
}

// This is the map of already created count connectors for particular configurations.
// We maintain this map because the Factory is asked trace, metric, and log receivers
// separately but they must not create separate objects. When the connector is shutdown
// it should be removed from this map so the same configuration can be recreated successfully.
var connectors = sharedcomponent.NewSharedComponents()

// otlpReceiver is the type that exposes Trace and Metrics reception.
type countConnector struct {
	cfg *Config

	metricsConsumer consumer.Metrics

	// Use ExporterCreateSettings because exporters are created first.
	// Receiver settings should be the same anyways.
	settings component.ExporterCreateSettings
}

func newCountConnector(cfg *Config, set component.ExporterCreateSettings) *countConnector {
	return &countConnector{
		cfg:      cfg,
		settings: set,
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

func (c *countConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return c.metricsConsumer.ConsumeMetrics(ctx, newCountMetric("spans", td.SpanCount()))
}

func (c *countConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return c.metricsConsumer.ConsumeMetrics(ctx, newCountMetric("metrics", md.MetricCount()))
}

func (c *countConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return c.metricsConsumer.ConsumeMetrics(ctx, newCountMetric("logs", ld.LogRecordCount()))
}

func newCountMetric(signalType string, count int) pmetric.Metrics {
	ms := pmetric.NewMetrics()
	rms := ms.ResourceMetrics().AppendEmpty()
	sms := rms.ScopeMetrics().AppendEmpty()
	cm := sms.Metrics().AppendEmpty()
	cm.SetName(fmt.Sprintf("count.%s", signalType))
	cm.SetDescription(fmt.Sprintf("The number of %ss observed.", signalType))
	sum := cm.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.MetricAggregationTemporalityDelta)
	dp := sum.DataPoints().AppendEmpty()
	dp.Attributes().PutString("signal.type", signalType)
	dp.SetIntValue(int64(count))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return ms
}
