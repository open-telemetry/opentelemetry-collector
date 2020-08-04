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
	"errors"
	"reflect"
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
	_, err := factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.Error(t, err)
	lfactory := factory.(component.LogsReceiverFactory)
	_, err = lfactory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
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

	_, err := factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)

	lfactory := factory.(component.LogsReceiverFactory)
	_, err = lfactory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
}

func TestNewFactory_WithSharedPerConfig(t *testing.T) {
	secondConfig := *defaultCfg
	secondConfig.SetName("test2")

	var lastShared interface{}

	factory := NewFactory(
		typeStr,
		defaultConfig,
		WithSharedTraces(createSharedTraceReceiver),
		WithSharedMetrics(createSharedMetricsReceiver),
		WithSharedLogs(createSharedLogsReceiver),
		WithSharedPerConfig(func(cfg configmodels.Receiver, params component.ReceiverCreateParams) (interface{}, error) {
			if cfg.Name() == "error" {
				return nil, errors.New("failed")
			}
			lastShared = []byte("test")
			return lastShared, nil
		}))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, defaultCfg, factory.CreateDefaultConfig())

	r, err := factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, r.(*DummyReceiver).Shared, []byte("test"))
	assert.Equal(t, lastShared, []byte("test"))
	// Make sure they are actually shared instances and not just the same
	// value.
	assert.Equal(t, reflect.ValueOf(&r.(*DummyReceiver).Shared).Elem().InterfaceData(), reflect.ValueOf(&lastShared).Elem().InterfaceData())

	r, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, r.(*DummyReceiver).Shared, []byte("test"))
	assert.Equal(t, lastShared, []byte("test"))
	assert.Equal(t, reflect.ValueOf(&r.(*DummyReceiver).Shared).Elem().InterfaceData(), reflect.ValueOf(&lastShared).Elem().InterfaceData())

	lfactory := factory.(component.LogsReceiverFactory)
	r, err = lfactory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{}, defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, r.(*DummyReceiver).Shared, []byte("test"))
	assert.Equal(t, lastShared, []byte("test"))
	assert.Equal(t, reflect.ValueOf(&r.(*DummyReceiver).Shared).Elem().InterfaceData(), reflect.ValueOf(&lastShared).Elem().InterfaceData())

	prevShared := lastShared
	r, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, &secondConfig, nil)
	assert.NoError(t, err)
	assert.Equal(t, r.(*DummyReceiver).Shared, []byte("test"))
	assert.Equal(t, lastShared, []byte("test"))
	assert.NotEqual(t, reflect.ValueOf(&r.(*DummyReceiver).Shared).Elem().InterfaceData(), reflect.ValueOf(&prevShared).Elem().InterfaceData())

	errCfg := *defaultCfg
	errCfg.SetName("error")
	_, err = factory.CreateTraceReceiver(context.Background(), component.ReceiverCreateParams{}, &errCfg, nil)
	assert.Error(t, err)

	_, err = factory.CreateMetricsReceiver(context.Background(), component.ReceiverCreateParams{}, &errCfg, nil)
	assert.Error(t, err)

	_, err = lfactory.CreateLogsReceiver(context.Background(), component.ReceiverCreateParams{}, &errCfg, nil)
	assert.Error(t, err)
}

func defaultConfig() configmodels.Receiver {
	return defaultCfg
}

func createTraceReceiver(ctx context.Context, params component.ReceiverCreateParams, cfg configmodels.Receiver, next consumer.TraceConsumer) (component.TraceReceiver, error) {
	return nil, nil
}

func createMetricsReceiver(ctx context.Context, params component.ReceiverCreateParams, cfg configmodels.Receiver, next consumer.MetricsConsumer) (component.MetricsReceiver, error) {
	return nil, nil
}

func createLogsReceiver(ctx context.Context, params component.ReceiverCreateParams, cfg configmodels.Receiver, next consumer.LogsConsumer) (component.LogsReceiver, error) {
	return nil, nil
}

func createSharedTraceReceiver(ctx context.Context, params component.ReceiverCreateParams, cfg configmodels.Receiver, next consumer.TraceConsumer, shared interface{}) (component.TraceReceiver, error) {
	return &DummyReceiver{
		Params:   &params,
		Config:   &cfg,
		Consumer: next,
		Shared:   shared,
	}, nil
}

func createSharedMetricsReceiver(ctx context.Context, params component.ReceiverCreateParams, cfg configmodels.Receiver, next consumer.MetricsConsumer, shared interface{}) (component.MetricsReceiver, error) {
	return &DummyReceiver{
		Params:   &params,
		Config:   &cfg,
		Consumer: next,
		Shared:   shared,
	}, nil
}

func createSharedLogsReceiver(ctx context.Context, params component.ReceiverCreateParams, cfg configmodels.Receiver, next consumer.LogsConsumer, shared interface{}) (component.LogsReceiver, error) {
	return &DummyReceiver{
		Params:   &params,
		Config:   &cfg,
		Consumer: next,
		Shared:   shared,
	}, nil
}

func customUnmarshaler(*viper.Viper, interface{}) error {
	return errors.New("my error")
}

type DummyReceiver struct {
	Params   *component.ReceiverCreateParams
	Config   *configmodels.Receiver
	Consumer interface{}
	Shared   interface{}
}

func (r *DummyReceiver) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (r *DummyReceiver) Shutdown(ctx context.Context) error {
	return nil
}
