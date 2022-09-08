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

package testcomponents // import "go.opentelemetry.io/collector/service/internal/testcomponents"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// ExampleComponents registers example factories. This is only used by tests.
func ExampleComponents() (component.Factories, error) {
	return component.Factories{
		Receivers: map[component.Type]component.ReceiverFactory{
			ExampleReceiverFactory.Type(): ExampleReceiverFactory,
		},
		Processors: map[component.Type]component.ProcessorFactory{
			ExampleProcessorFactory.Type(): ExampleProcessorFactory,
		},
		Exporters: map[component.Type]component.ExporterFactory{
			ExampleExporterFactory.Type(): ExampleExporterFactory,
		},
		Connectors: map[component.Type]component.ConnectorFactory{
			ExampleConnectorFactory.Type(): ExampleConnectorFactory,
		},
	}, nil
}

type StatefulComponent interface {
	component.Component
	Started() bool
	Stopped() bool
	RecallTraces() []ptrace.Traces
	RecallMetrics() []pmetric.Metrics
	RecallLogs() []plog.Logs
}

type componentState struct {
	started bool
	stopped bool
	traces  []ptrace.Traces
	metrics []pmetric.Metrics
	logs    []plog.Logs
}

func (cs *componentState) Started() bool {
	return cs.started
}

func (cs *componentState) Stopped() bool {
	return cs.stopped
}

func (cs *componentState) RecallTraces() []ptrace.Traces {
	return cs.traces
}

func (cs *componentState) RecallMetrics() []pmetric.Metrics {
	return cs.metrics
}

func (cs *componentState) RecallLogs() []plog.Logs {
	return cs.logs
}
