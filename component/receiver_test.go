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

// TODO: Move tests back to component package after config.*Settings are removed.

package component_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

func TestNewReceiverFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewReceiverSettings(component.NewID(typeStr))
	factory := component.NewReceiverFactory(
		typeStr,
		func() component.ReceiverConfig { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesReceiver(context.Background(), component.ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsReceiver(context.Background(), component.ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewReceiverFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewReceiverSettings(component.NewID(typeStr))
	factory := component.NewReceiverFactory(
		typeStr,
		func() component.ReceiverConfig { return &defaultCfg },
		component.WithTracesReceiver(createTracesReceiver, component.StabilityLevelDeprecated),
		component.WithMetricsReceiver(createMetricsReceiver, component.StabilityLevelAlpha),
		component.WithLogsReceiver(createLogsReceiver, component.StabilityLevelStable))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDeprecated, factory.TracesReceiverStability())
	_, err := factory.CreateTracesReceiver(context.Background(), component.ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsReceiverStability())
	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.LogsReceiverStability())
	_, err = factory.CreateLogsReceiver(context.Background(), component.ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func createTracesReceiver(context.Context, component.ReceiverCreateSettings, component.ReceiverConfig, consumer.Traces) (component.TracesReceiver, error) {
	return nil, nil
}

func createMetricsReceiver(context.Context, component.ReceiverCreateSettings, component.ReceiverConfig, consumer.Metrics) (component.MetricsReceiver, error) {
	return nil, nil
}

func createLogsReceiver(context.Context, component.ReceiverCreateSettings, component.ReceiverConfig, consumer.Logs) (component.LogsReceiver, error) {
	return nil, nil
}
