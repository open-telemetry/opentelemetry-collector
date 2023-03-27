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

package forwardconnector // import "go.opentelemetry.io/collector/connector/forwardconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

const (
	typeStr = "forward"
)

// NewFactory returns a connector.Factory.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesToTraces, component.StabilityLevelDevelopment),
		connector.WithMetricsToMetrics(createMetricsToMetrics, component.StabilityLevelDevelopment),
		connector.WithLogsToLogs(createLogsToLogs, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

// createTracesToTraces creates a trace receiver based on provided config.
func createTracesToTraces(
	_ context.Context,
	_ connector.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	return &forward{Traces: nextConsumer}, nil
}

// createMetricsToMetrics creates a metrics receiver based on provided config.
func createMetricsToMetrics(
	_ context.Context,
	_ connector.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Metrics,
) (connector.Metrics, error) {
	return &forward{Metrics: nextConsumer}, nil
}

// createLogsToLogs creates a log receiver based on provided config.
func createLogsToLogs(
	_ context.Context,
	_ connector.CreateSettings,
	_ component.Config,
	nextConsumer consumer.Logs,
) (connector.Logs, error) {
	return &forward{Logs: nextConsumer}, nil
}

// forward is used to pass signals directly from one pipeline to another.
// This is useful when there is a need to replicate data and process it in more
// than one way. It can also be used to join pipelines together.
type forward struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
	component.StartFunc
	component.ShutdownFunc
}

func (c *forward) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
