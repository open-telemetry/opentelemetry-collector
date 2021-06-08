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

package receiverhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const typeStr = "test"

var defaultCfg = config.NewReceiverSettings(config.NewID(typeStr))

func TestNewFactory(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), factory.CreateDefaultConfig(), nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), factory.CreateDefaultConfig(), nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), factory.CreateDefaultConfig(), nil)
	assert.Error(t, err)
}

func TestNewFactory_WithConstructors(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		WithTraces(createTracesReceiver),
		WithMetrics(createMetricsReceiver),
		WithLogs(createLogsReceiver))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), factory.CreateDefaultConfig(), nil)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), factory.CreateDefaultConfig(), nil)
	assert.NoError(t, err)

	_, err = factory.CreateLogsReceiver(context.Background(), componenttest.NewNopReceiverCreateSettings(), factory.CreateDefaultConfig(), nil)
	assert.NoError(t, err)
}

func defaultConfig() config.Receiver {
	return &defaultCfg
}

func createTracesReceiver(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Traces) (component.TracesReceiver, error) {
	return nil, nil
}

func createMetricsReceiver(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Metrics) (component.MetricsReceiver, error) {
	return nil, nil
}

func createLogsReceiver(context.Context, component.ReceiverCreateSettings, config.Receiver, consumer.Logs) (component.LogsReceiver, error) {
	return nil, nil
}
