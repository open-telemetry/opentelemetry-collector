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

package connectortest // import "go.opentelemetry.io/collector/connector/connectortest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

const typeStr = "nop"

// NewNopCreateSettings returns a new nop settings for Create* functions.
func NewNopCreateSettings() connector.CreateSettings {
	return connector.CreateSettings{
		TelemetrySettings: componenttest.NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopConfig struct{}

// NewNopFactory returns a connector.Factory that constructs nop processors.
func NewNopFactory() connector.Factory {
	return connector.NewFactory(
		"nop",
		func() component.Config {
			return &nopConfig{}
		},
		connector.WithTracesToTraces(createTracesToTracesConnector, component.StabilityLevelDevelopment),
		connector.WithTracesToMetrics(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		connector.WithTracesToLogs(createTracesToLogsConnector, component.StabilityLevelDevelopment),
		connector.WithMetricsToTraces(createMetricsToTracesConnector, component.StabilityLevelDevelopment),
		connector.WithMetricsToMetrics(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		connector.WithMetricsToLogs(createMetricsToLogsConnector, component.StabilityLevelDevelopment),
		connector.WithLogsToTraces(createLogsToTracesConnector, component.StabilityLevelDevelopment),
		connector.WithLogsToMetrics(createLogsToMetricsConnector, component.StabilityLevelDevelopment),
		connector.WithLogsToLogs(createLogsToLogsConnector, component.StabilityLevelDevelopment),
	)
}

func createTracesToTracesConnector(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToMetricsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createTracesToLogsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Traces, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToTracesConnector(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToMetricsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createMetricsToLogsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Metrics, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToTracesConnector(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToMetricsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

func createLogsToLogsConnector(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Logs, error) {
	return &nopConnector{Consumer: consumertest.NewNop()}, nil
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

// NewNopBuilder returns a connector.Builder that constructs nop receivers.
func NewNopBuilder() *connector.Builder {
	nopFactory := NewNopFactory()
	// Use a different ID than receivertest and exportertest to avoid ambiguous
	// configuration scenarios. Ambiguous IDs are detected in the 'otelcol' package,
	// but lower level packages such as 'service' assume that IDs are disambiguated.
	connID := component.NewIDWithName(typeStr, "conn")
	return connector.NewBuilder(
		map[component.ID]component.Config{connID: nopFactory.CreateDefaultConfig()},
		map[component.Type]connector.Factory{typeStr: nopFactory})
}
