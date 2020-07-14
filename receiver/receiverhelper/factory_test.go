// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	assert.Nil(t, factory.CustomUnmarshaler())
	_, err := factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewFactory_WithConstructors(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		WithTraces(createTraceReceiver),
		WithMetrics(createMetricsReceiver),
		WithCustomUnmarshaler(customUnmarshaler))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())
	assert.NotNil(t, factory.CustomUnmarshaler())
	tr, err := factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, component.TraceReceiver(nil), tr)
	mr, err := factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, component.MetricsReceiver(nil), mr)
}

func defaultConfig() configmodels.Receiver {
	return defaultCfg
}

func createTraceReceiver(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.TraceConsumer) (component.TraceReceiver, error) {
	return nil, nil
}

func createMetricsReceiver(context.Context, component.ReceiverCreateParams, configmodels.Receiver, consumer.MetricsConsumer) (component.MetricsReceiver, error) {
	return nil, nil
}

func customUnmarshaler(*viper.Viper, interface{}) error {
	return nil
}
