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
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const receiverType = component.Type("examplereceiver")

// ExampleReceiverFactory is factory for ExampleReceiver.
var ExampleReceiverFactory = receiver.NewFactory(
	receiverType,
	createReceiverDefaultConfig,
	receiver.WithTraces(createTracesReceiver, component.StabilityLevelDevelopment),
	receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
	receiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment))

func createReceiverDefaultConfig() component.Config {
	return &struct{}{}
}

// createTracesReceiver creates a trace receiver based on this config.
func createTracesReceiver(
	_ context.Context,
	_ receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	tr := createReceiver(cfg)
	tr.ConsumeTracesFunc = nextConsumer.ConsumeTraces
	return tr, nil
}

// createMetricsReceiver creates a metrics receiver based on this config.
func createMetricsReceiver(
	_ context.Context,
	_ receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	mr := createReceiver(cfg)
	mr.ConsumeMetricsFunc = nextConsumer.ConsumeMetrics
	return mr, nil
}

func createLogsReceiver(
	_ context.Context,
	_ receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	lr := createReceiver(cfg)
	lr.ConsumeLogsFunc = nextConsumer.ConsumeLogs
	return lr, nil
}

func createReceiver(cfg component.Config) *ExampleReceiver {
	// There must be one receiver for all data types. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	er, ok := exampleReceivers[cfg]
	if !ok {
		er = &ExampleReceiver{}
		// Remember the receiver in the map
		exampleReceivers[cfg] = er
	}

	return er
}

// ExampleReceiver allows producing traces and metrics for testing purposes.
type ExampleReceiver struct {
	componentState
	consumer.ConsumeTracesFunc
	consumer.ConsumeMetricsFunc
	consumer.ConsumeLogsFunc
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the ReceiverFactory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[component.Config]*ExampleReceiver{}
