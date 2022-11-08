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

func TestNewProcessorFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewProcessorSettings(component.NewID(typeStr))
	factory := component.NewProcessorFactory(
		typeStr,
		func() component.ProcessorConfig { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewProcessorFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewProcessorSettings(component.NewID(typeStr))
	factory := component.NewProcessorFactory(
		typeStr,
		func() component.ProcessorConfig { return &defaultCfg },
		component.WithTracesProcessor(createTracesProcessor, component.StabilityLevelAlpha),
		component.WithMetricsProcessor(createMetricsProcessor, component.StabilityLevelBeta),
		component.WithLogsProcessor(createLogsProcessor, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesProcessorStability())
	_, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsProcessorStability())
	_, err = factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsProcessorStability())
	_, err = factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func createTracesProcessor(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Traces) (component.TracesProcessor, error) {
	return nil, nil
}

func createMetricsProcessor(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Metrics) (component.MetricsProcessor, error) {
	return nil, nil
}

func createLogsProcessor(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Logs) (component.LogsProcessor, error) {
	return nil, nil
}
