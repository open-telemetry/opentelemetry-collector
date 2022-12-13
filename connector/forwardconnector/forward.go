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
	"go.opentelemetry.io/collector/internal/sharedcomponent"
)

const (
	typeStr = "forward"
)

type forwardFactory struct {
	// This is the map of already created forward connectors for particular configurations.
	// We maintain this map because the Factory is asked trace, metric, and log receivers
	// separately but they must not create separate objects. When the connector is shutdown
	// it should be removed from this map so the same configuration can be recreated successfully.
	*sharedcomponent.SharedComponents
}

// NewFactory returns a connector.Factory.
func NewFactory() connector.Factory {
	f := &forwardFactory{sharedcomponent.NewSharedComponents()}
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithTracesToTraces(f.createTracesToTraces, component.StabilityLevelDevelopment),
		connector.WithMetricsToMetrics(f.createMetricsToMetrics, component.StabilityLevelDevelopment),
		connector.WithLogsToLogs(f.createLogsToLogs, component.StabilityLevelDevelopment),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &struct{}{}
}

// createTracesToTraces creates a trace receiver based on provided config.
func (f *forwardFactory) createTracesToTraces(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (connector.Traces, error) {
	comp, _ := f.GetOrAdd(cfg, func() (component.Component, error) {
		return &forward{}, nil
	})

	conn := comp.Unwrap().(*forward)
	conn.Traces = nextConsumer
	return conn, nil
}

// createMetricsToMetrics creates a metrics receiver based on provided config.
func (f *forwardFactory) createMetricsToMetrics(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Metrics, error) {
	comp, _ := f.GetOrAdd(cfg, func() (component.Component, error) {
		return &forward{}, nil
	})

	conn := comp.Unwrap().(*forward)
	conn.Metrics = nextConsumer
	return conn, nil
}

// createLogsToLogs creates a log receiver based on provided config.
func (f *forwardFactory) createLogsToLogs(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (connector.Logs, error) {
	comp, _ := f.GetOrAdd(cfg, func() (component.Component, error) {
		return &forward{}, nil
	})

	conn := comp.Unwrap().(*forward)
	conn.Logs = nextConsumer
	return conn, nil
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
