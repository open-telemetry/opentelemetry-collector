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
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

const typeStr = "test"

var defaultCfg = &configmodels.ReceiverSettings{
	TypeVal: typeStr,
	NameVal: typeStr,
}

func TestNewFactory(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())
	_, ok := factory.(component.ConfigUnmarshaler)
	assert.False(t, ok)
	_, err := factory.CreateTracesReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewFactory_WithConstructors(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		WithTraces(createTraceReceiver),
		WithMetrics(createMetricsReceiver),
		WithLogs(createLogsReceiver),
		WithCustomUnmarshaler(customUnmarshaler))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())

	fu, ok := factory.(component.ConfigUnmarshaler)
	assert.True(t, ok)
	assert.Equal(t, errors.New("my error"), fu.Unmarshal(nil, nil))

	_, err := factory.CreateTracesReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
}

func defaultConfig() configmodels.Receiver {
	return defaultCfg
}

func createTraceReceiver(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.TracesConsumer) (component.TracesReceiver, error) {
	return nil, nil
}

func createMetricsReceiver(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.MetricsConsumer) (component.MetricsReceiver, error) {
	return nil, nil
}

func createLogsReceiver(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.LogsConsumer) (component.LogsReceiver, error) {
	return nil, nil
}

func customUnmarshaler(*viper.Viper, interface{}) error {
	return errors.New("my error")
}
