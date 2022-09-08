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

package nopconnector // import "go.opentelemetry.io/collector/connector/nopconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	typeStr = "nop"
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
		component.WithTracesToTracesConnector(createTracesToTracesConnector, component.StabilityLevelDevelopment),
		component.WithMetricsToMetricsConnector(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithLogsToLogsConnector(createLogsToLogsConnector, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &Config{}
}

// createTracesToTracesConnector creates a trace receiver based on provided config.
func createTracesToTracesConnector(
	_ context.Context,
	set component.ConnectorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (component.TracesToTracesConnector, error) {
	comp := connectors.GetOrAdd(cfg, func() component.Component {
		return newNopConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*nopConnector)
	conn.tracesConsumer = nextConsumer
	return conn, nil
}

// createMetricsToMetricsConnector creates a metrics receiver based on provided config.
func createMetricsToMetricsConnector(
	_ context.Context,
	set component.ConnectorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (component.MetricsToMetricsConnector, error) {
	comp := connectors.GetOrAdd(cfg, func() component.Component {
		return newNopConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*nopConnector)
	conn.metricsConsumer = nextConsumer
	return conn, nil
}

// createLogsToLogsConnector creates a log receiver based on provided config.
func createLogsToLogsConnector(
	_ context.Context,
	set component.ConnectorCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (component.LogsToLogsConnector, error) {
	comp := connectors.GetOrAdd(cfg, func() component.Component {
		return newNopConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*nopConnector)
	conn.logsConsumer = nextConsumer
	return conn, nil
}

// This is the map of already created nop connectors for particular configurations.
// We maintain this map because the Factory is asked trace, metric, and log receivers
// separately but they must not create separate objects. When the connector is shutdown
// it should be removed from this map so the same configuration can be recreated successfully.
var connectors = sharedcomponent.NewSharedComponents()

// nopConnector is used to pass signals directly from one pipeline to another.
// This is useful when there is a need to replicate data and process it in more
// than one way. It can also be used to join pipelines together.
type nopConnector struct {
	cfg *Config

	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs

	settings component.ConnectorCreateSettings
}

func newNopConnector(cfg *Config, settings component.ConnectorCreateSettings) *nopConnector {
	return &nopConnector{cfg: cfg, settings: settings}
}

func (c *nopConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *nopConnector) Start(_ context.Context, host component.Host) error {
	return nil
}

func (c *nopConnector) Shutdown(ctx context.Context) error {
	return nil
}

func (c *nopConnector) ConsumeTracesToTraces(ctx context.Context, td ptrace.Traces) error {
	return c.tracesConsumer.ConsumeTraces(ctx, td)
}

func (c *nopConnector) ConsumeMetricsToMetrics(ctx context.Context, md pmetric.Metrics) error {
	return c.metricsConsumer.ConsumeMetrics(ctx, md)
}

func (c *nopConnector) ConsumeLogsToLogs(ctx context.Context, ld plog.Logs) error {
	return c.logsConsumer.ConsumeLogs(ctx, ld)
}
