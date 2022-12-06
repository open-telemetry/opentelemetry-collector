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

package receiver

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
	factory := NewFactory(
		typeStr,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesReceiver(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsReceiver(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewReceiverFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewReceiverSettings(component.NewID(typeStr))
	factory := NewFactory(
		typeStr,
		func() component.Config { return &defaultCfg },
		WithTraces(createTraces, component.StabilityLevelDeprecated),
		WithMetrics(createMetrics, component.StabilityLevelAlpha),
		WithLogs(createLogs, component.StabilityLevelStable))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelDeprecated, factory.TracesReceiverStability())
	_, err := factory.CreateTracesReceiver(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsReceiverStability())
	_, err = factory.CreateMetricsReceiver(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.LogsReceiverStability())
	_, err = factory.CreateLogsReceiver(context.Background(), CreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func createTraces(context.Context, CreateSettings, component.Config, consumer.Traces) (Traces, error) {
	return nil, nil
}

func createMetrics(context.Context, CreateSettings, component.Config, consumer.Metrics) (Metrics, error) {
	return nil, nil
}

func createLogs(context.Context, CreateSettings, component.Config, consumer.Logs) (Logs, error) {
	return nil, nil
}
