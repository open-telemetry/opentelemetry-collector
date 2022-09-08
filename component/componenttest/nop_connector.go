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

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewNopConnectorCreateSettings returns a new nop settings for Create*Connector functions.
func NewNopConnectorCreateSettings() component.ConnectorCreateSettings {
	return component.ConnectorCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopConnectorConfig struct {
	config.ConnectorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewNopConnectorFactory returns a component.ConnectorFactory that constructs nop processors.
func NewNopConnectorFactory() component.ConnectorFactory {
	return component.NewConnectorFactory(
		"nop",
		func() component.Config {
			return &nopConnectorConfig{
				ConnectorSettings: config.NewConnectorSettings(component.NewID("nop")),
			}
		},
		component.WithTracesToTracesConnector(createTracesToTracesConnector, component.StabilityLevelDevelopment),
		component.WithTracesToMetricsConnector(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithTracesToLogsConnector(createTracesToLogsConnector, component.StabilityLevelDevelopment),
		component.WithMetricsToTracesConnector(createMetricsToTracesConnector, component.StabilityLevelDevelopment),
		component.WithMetricsToMetricsConnector(createMetricsToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithMetricsToLogsConnector(createMetricsToLogsConnector, component.StabilityLevelDevelopment),
		component.WithLogsToTracesConnector(createLogsToTracesConnector, component.StabilityLevelDevelopment),
		component.WithLogsToMetricsConnector(createLogsToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithLogsToLogsConnector(createLogsToLogsConnector, component.StabilityLevelDevelopment),
	)
}

func createTracesToTracesConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Traces) (component.TracesToTracesConnector, error) {
	return nopConnectorInstance, nil
}
func createTracesToMetricsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Metrics) (component.TracesToMetricsConnector, error) {
	return nopConnectorInstance, nil
}
func createTracesToLogsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Logs) (component.TracesToLogsConnector, error) {
	return nopConnectorInstance, nil
}

func createMetricsToTracesConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Traces) (component.MetricsToTracesConnector, error) {
	return nopConnectorInstance, nil
}
func createMetricsToMetricsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Metrics) (component.MetricsToMetricsConnector, error) {
	return nopConnectorInstance, nil
}
func createMetricsToLogsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Logs) (component.MetricsToLogsConnector, error) {
	return nopConnectorInstance, nil
}

func createLogsToTracesConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Traces) (component.LogsToTracesConnector, error) {
	return nopConnectorInstance, nil
}
func createLogsToMetricsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Metrics) (component.LogsToMetricsConnector, error) {
	return nopConnectorInstance, nil
}
func createLogsToLogsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Logs) (component.LogsToLogsConnector, error) {
	return nopConnectorInstance, nil
}

var nopConnectorInstance = &nopConnector{
	Consumer: consumertest.NewNop(),
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	nopComponent
	consumertest.Consumer
}

func (c *nopConnector) ConsumeTracesToTraces(ctx context.Context, td ptrace.Traces) error {
	return nil
}
func (c *nopConnector) ConsumeTracesToMetrics(ctx context.Context, td ptrace.Traces) error {
	return nil
}
func (c *nopConnector) ConsumeTracesToLogs(ctx context.Context, td ptrace.Traces) error {
	return nil
}

func (c *nopConnector) ConsumeMetricsToTraces(ctx context.Context, md pmetric.Metrics) error {
	return nil
}
func (c *nopConnector) ConsumeMetricsToMetrics(ctx context.Context, md pmetric.Metrics) error {
	return nil
}
func (c *nopConnector) ConsumeMetricsToLogs(ctx context.Context, md pmetric.Metrics) error {
	return nil
}

func (c *nopConnector) ConsumeLogsToTraces(ctx context.Context, ld plog.Logs) error {
	return nil
}
func (c *nopConnector) ConsumeLogsToMetrics(ctx context.Context, ld plog.Logs) error {
	return nil
}
func (c *nopConnector) ConsumeLogsToLogs(ctx context.Context, ld plog.Logs) error {
	return nil
}
