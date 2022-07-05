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

package component

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

func TestNewReceiverFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewReceiverSettings(config.NewComponentID(typeStr))
	factory := NewReceiverFactory(
		typeStr,
		func() config.Receiver { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewReceiverFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewReceiverSettings(config.NewComponentID(typeStr))
	factory := NewReceiverFactory(
		typeStr,
		func() config.Receiver { return &defaultCfg },
		WithTracesReceiver(createTracesReceiver),
		WithMetricsReceiver(createMetricsReceiver),
		WithLogsReceiver(createLogsReceiver))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateLogsReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func TestNewReceiverFactory_WithStabilityLevels(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewReceiverSettings(config.NewComponentID(typeStr))
	factory := NewReceiverFactory(
		typeStr,
		func() config.Receiver { return &defaultCfg },
		WithTracesReceiverAndStabilityLevel(createTracesReceiver, StabilityLevelDeprecated),
		WithMetricsReceiverAndStabilityLevel(createMetricsReceiver, StabilityLevelAlpha),
		WithLogsReceiverAndStabilityLevel(createLogsReceiver, StabilityLevelStable))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.EqualValues(t, StabilityLevelDeprecated, factory.StabilityLevel(config.TracesDataType))
	_, err := factory.CreateTracesReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.EqualValues(t, StabilityLevelAlpha, factory.StabilityLevel(config.MetricsDataType))
	_, err = factory.CreateMetricsReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.EqualValues(t, StabilityLevelStable, factory.StabilityLevel(config.LogsDataType))
	_, err = factory.CreateLogsReceiver(context.Background(), ReceiverCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func createTracesReceiver(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Traces) (TracesReceiver, error) {
	return nil, nil
}

func createMetricsReceiver(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Metrics) (MetricsReceiver, error) {
	return nil, nil
}

func createLogsReceiver(context.Context, ReceiverCreateSettings, config.Receiver, consumer.Logs) (LogsReceiver, error) {
	return nil, nil
}
