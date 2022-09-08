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

package nopconnector // import "go.opentelemetry.io/collector/receiver/nopconnector"

import (
	"context"
	"fmt"

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
			component.WithTracesReceiver(createTracesReceiver, component.StabilityLevelInDevelopment),
			component.WithMetricsReceiver(createMetricsReceiver, component.StabilityLevelInDevelopment),
			component.WithLogsReceiver(createLogReceiver, component.StabilityLevelInDevelopment),
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
		return newNopConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*nopConnector)
	return conn, nil
}

// createMetricsExporter creates a metrics receiver based on provided config.
func createMetricsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.MetricsExporter, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		return newNopConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*nopConnector)
	return conn, nil
}

// createLogExporter creates a log receiver based on provided config.
func createLogExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter,
) (component.LogsExporter, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		return newNopConnector(cfg.(*Config), set)
	})

	conn := comp.Unwrap().(*nopConnector)
	return conn, nil
}

// createTracesReceiver creates a trace receiver based on provided config.
func createTracesReceiver(
	_ context.Context,
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Traces,
) (component.TracesReceiver, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		// Expect to have already created this component as an exporter
		return nil
	})

	if comp == nil {
		return nil, fmt.Errorf("connector must be initialized as exporter and receiver")
	}

	conn := comp.Unwrap().(*nopConnector)
	conn.tracesConsumer = nextConsumer
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

	conn := comp.Unwrap().(*nopConnector)
	conn.metricsConsumer = nextConsumer
	return conn, nil
}

// createLogReceiver creates a log receiver based on provided config.
func createLogReceiver(
	_ context.Context,
	set component.ReceiverCreateSettings,
	cfg config.Receiver,
	nextConsumer consumer.Logs,
) (component.LogsReceiver, error) {
	comp := connectors.GetOrAdd(cfg.ID(), func() component.Component {
		// Expect to have already created this component as an exporter
		return nil
	})

	if comp == nil {
		return nil, fmt.Errorf("connector must be initialized as exporter and receiver")
	}

	conn := comp.Unwrap().(*nopConnector)
	conn.logsConsumer = nextConsumer
	return conn, nil
}

// This is the map of already created nop connectors for particular configurations.
// We maintain this map because the Factory is asked trace, metric, and log receivers
// separately but they must not create separate objects. When the connector is shutdown
// it should be removed from this map so the same configuration can be recreated successfully.
var connectors = sharedcomponent.NewSharedComponents()

// otlpReceiver is the type that exposes Trace and Metrics reception.
type nopConnector struct {
	cfg *Config

	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	logsConsumer    consumer.Logs

	// Use ExporterCreateSettings because exporters are created first.
	// Receiver settings should be the same anyways.
	settings component.ExporterCreateSettings
}

func newNopConnector(cfg *Config, set component.ExporterCreateSettings) *nopConnector {
	return &nopConnector{cfg: cfg, settings: set}
}

func (c *nopConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *nopConnector) Start(_ context.Context, host component.Host) error {
	// TODO
	return nil
}

func (c *nopConnector) Shutdown(ctx context.Context) error {
	// TODO
	return nil
}

func (c *nopConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return c.tracesConsumer.ConsumeTraces(ctx, td)
}

func (c *nopConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return c.metricsConsumer.ConsumeMetrics(ctx, md)
}

func (c *nopConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return c.logsConsumer.ConsumeLogs(ctx, ld)
}
