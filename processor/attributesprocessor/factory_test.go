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

package attributesprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, factory.Type(), config.Type(typeStr))
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, cfg, &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
	})
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestFactoryCreateTracesProcessor_EmptyActions(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	ap, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, ap)
}

func TestFactoryCreateTracesProcessor_InvalidActions(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	// Missing key
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "", Value: 123, Action: processorhelper.UPSERT},
	}
	ap, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, ap)
}

func TestFactoryCreateTracesProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "a key", Action: processorhelper.DELETE},
	}

	tp, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.NotNil(t, tp)
	assert.NoError(t, err)

	tp, err = factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, nil)
	assert.Nil(t, tp)
	assert.Error(t, err)

	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Action: processorhelper.DELETE},
	}
	tp, err = factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.Nil(t, tp)
	assert.Error(t, err)
}

func TestFactory_CreateMetricsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	mp, err := factory.CreateMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, nil)
	require.Nil(t, mp)
	assert.Equal(t, err, componenterror.ErrDataTypeIsNotSupported)
}

func TestFactoryCreateLogsProcessor_EmptyActions(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	ap, err := factory.CreateLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, ap)
}

func TestFactoryCreateLogsProcessor_InvalidActions(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	// Missing key
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "", Value: 123, Action: processorhelper.UPSERT},
	}
	ap, err := factory.CreateLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, ap)
}

func TestFactoryCreateLogsProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "a key", Action: processorhelper.DELETE},
	}

	tp, err := factory.CreateLogsProcessor(
		context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.NotNil(t, tp)
	assert.NoError(t, err)

	tp, err = factory.CreateLogsProcessor(
		context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, nil)
	assert.Nil(t, tp)
	assert.Error(t, err)

	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Action: processorhelper.DELETE},
	}
	tp, err = factory.CreateLogsProcessor(
		context.Background(), componenttest.NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	assert.Nil(t, tp)
	assert.Error(t, err)
}
