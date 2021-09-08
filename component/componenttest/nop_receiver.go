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

package componenttest

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

// NewNopReceiverCreateSettings returns a new nop settings for Create*Receiver functions.
func NewNopReceiverCreateSettings() component.ReceiverCreateSettings {
	return component.ReceiverCreateSettings{
		TelemetryCreateSettings: NewNopTelemetryCreateSettings(),
		BuildInfo:               component.DefaultBuildInfo(),
	}
}

type nopReceiverConfig struct {
	config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// nopReceiverFactory is factory for nopReceiver.
type nopReceiverFactory struct{}

var nopReceiverFactoryInstance = &nopReceiverFactory{}

// NewNopReceiverFactory returns a component.ReceiverFactory that constructs nop receivers.
func NewNopReceiverFactory() component.ReceiverFactory {
	return nopReceiverFactoryInstance
}

// Type gets the type of the Receiver config created by this factory.
func (f *nopReceiverFactory) Type() config.Type {
	return config.NewID("nop").Type()
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *nopReceiverFactory) CreateDefaultConfig() config.Receiver {
	return &nopReceiverConfig{
		ReceiverSettings: config.NewReceiverSettings(config.NewID("nop")),
	}
}

// CreateTracesReceiver implements component.ReceiverFactory interface.
func (f *nopReceiverFactory) CreateTracesReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	_ config.Receiver,
	_ consumer.Traces,
) (component.TracesReceiver, error) {
	return nopReceiverInstance, nil
}

// CreateMetricsReceiver implements component.ReceiverFactory interface.
func (f *nopReceiverFactory) CreateMetricsReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	_ config.Receiver,
	_ consumer.Metrics,
) (component.MetricsReceiver, error) {
	return nopReceiverInstance, nil
}

// CreateLogsReceiver implements component.ReceiverFactory interface.
func (f *nopReceiverFactory) CreateLogsReceiver(
	_ context.Context,
	_ component.ReceiverCreateSettings,
	_ config.Receiver,
	_ consumer.Logs,
) (component.LogsReceiver, error) {
	return nopReceiverInstance, nil
}

var nopReceiverInstance = &nopReceiver{
	Component: componenthelper.New(),
}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	component.Component
}
