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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const receiverType = component.Type("examplereceiver")

// ExampleReceiverConfig config for ExampleReceiver.
type ExampleReceiverConfig struct {
	config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// ExampleReceiverFactory is factory for ExampleReceiver.
var ExampleReceiverFactory = component.NewReceiverFactory(
	receiverType,
	createReceiverDefaultConfig,
	component.WithTracesReceiver(createTracesReceiver, component.StabilityLevelDevelopment),
	component.WithMetricsReceiver(createMetricsReceiver, component.StabilityLevelDevelopment),
	component.WithLogsReceiver(createLogsReceiver, component.StabilityLevelDevelopment))

func createReceiverDefaultConfig() component.Config {
	return &ExampleReceiverConfig{
		ReceiverSettings: config.NewReceiverSettings(component.NewID(receiverType)),
	}
}

// createTracesReceiver creates a trace receiver based on this config.
func createTracesReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (component.TracesReceiver, error) {
	receiver := createReceiver(cfg)
	receiver.Traces = nextConsumer
	return receiver, nil
}

// createMetricsReceiver creates a metrics receiver based on this config.
func createMetricsReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (component.MetricsReceiver, error) {
	receiver := createReceiver(cfg)
	receiver.Metrics = nextConsumer
	return receiver, nil
}

func createLogsReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (component.LogsReceiver, error) {
	receiver := createReceiver(cfg)
	receiver.Logs = nextConsumer
	return receiver, nil
}

func createReceiver(cfg component.Config) *ExampleReceiver {
	// There must be one receiver for all data types. We maintain a map of
	// receivers per config.

	// Check to see if there is already a receiver for this config.
	receiver, ok := exampleReceivers[cfg]
	if !ok {
		receiver = &ExampleReceiver{}
		// Remember the receiver in the map
		exampleReceivers[cfg] = receiver
	}

	return receiver
}

var _ StatefulComponent = &ExampleReceiver{}

// ExampleReceiver allows producing traces and metrics for testing purposes.
type ExampleReceiver struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
	componentState
}

// Start tells the receiver to start its processing.
func (erp *ExampleReceiver) Start(_ context.Context, _ component.Host) error {
	erp.started = true
	return nil
}

// Shutdown tells the receiver that should stop reception,
func (erp *ExampleReceiver) Shutdown(context.Context) error {
	erp.stopped = true
	return nil
}

// This is the map of already created example receivers for particular configurations.
// We maintain this map because the ReceiverFactory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exampleReceivers = map[component.Config]*ExampleReceiver{}
