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
)

// NewNopReceiverCreateSettings returns a new nop settings for Create*Receiver functions.
func NewNopReceiverCreateSettings() component.ReceiverCreateSettings {
	return component.ReceiverCreateSettings{
		TelemetrySettings: NewNopTelemetrySettings(),
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
}

type nopReceiverConfig struct {
	config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
}

// NewNopReceiverFactory returns a component.ReceiverFactory that constructs nop receivers.
func NewNopReceiverFactory() component.ReceiverFactory {
	return component.NewReceiverFactory(
		"nop",
		func() config.Receiver {
			return &nopReceiverConfig{
				ReceiverSettings: config.NewReceiverSettings(config.NewComponentID("nop")),
			}
		},
		component.WithTracesReceiver(createTracesReceiver, component.StabilityLevelStable),
		component.WithMetricsReceiver(createMetricsReceiver, component.StabilityLevelStable),
		component.WithLogsReceiver(createLogsReceiver, component.StabilityLevelStable))
}

func createTracesReceiver(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Traces) (component.TracesReceiver, error) {
	return nopReceiverInstance, nil
}

func createMetricsReceiver(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Metrics) (component.MetricsReceiver, error) {
	return nopReceiverInstance, nil
}

func createLogsReceiver(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Logs) (component.LogsReceiver, error) {
	return nopReceiverInstance, nil
}

var nopReceiverInstance = &nopReceiver{}

// nopReceiver stores consumed traces and metrics for testing purposes.
type nopReceiver struct {
	nopComponent
}
