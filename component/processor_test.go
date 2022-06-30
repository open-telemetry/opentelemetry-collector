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

func TestNewProcessorFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewProcessorSettings(config.NewComponentID(typeStr))
	factory := NewProcessorFactory(
		typeStr,
		func() config.Processor { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewProcessorFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewProcessorSettings(config.NewComponentID(typeStr))
	factory := NewProcessorFactory(
		typeStr,
		func() config.Processor { return &defaultCfg },
		WithTracesProcessor(createTracesProcessor),
		WithMetricsProcessor(createMetricsProcessor),
		WithLogsProcessor(createLogsProcessor))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateLogsProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func TestNewProcessorFactory_WithStabilityLevel(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewProcessorSettings(config.NewComponentID(typeStr))
	factory := NewProcessorFactory(
		typeStr,
		func() config.Processor { return &defaultCfg },
		WithTracesProcessorAndStabilityLevel(createTracesProcessor, StabilityLevelAlpha),
		WithMetricsProcessorAndStabilityLevel(createMetricsProcessor, StabilityLevelBeta),
		WithLogsProcessorAndStabilityLevel(createLogsProcessor, StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.EqualValues(t, StabilityLevelAlpha, factory.StabilityLevel(config.TracesDataType))
	_, err := factory.CreateTracesProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.EqualValues(t, StabilityLevelBeta, factory.StabilityLevel(config.MetricsDataType))
	_, err = factory.CreateMetricsProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.EqualValues(t, StabilityLevelUnmaintained, factory.StabilityLevel(config.LogsDataType))
	_, err = factory.CreateLogsProcessor(context.Background(), ProcessorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func createTracesProcessor(context.Context, ProcessorCreateSettings, config.Processor, consumer.Traces) (TracesProcessor, error) {
	return nil, nil
}

func createMetricsProcessor(context.Context, ProcessorCreateSettings, config.Processor, consumer.Metrics) (MetricsProcessor, error) {
	return nil, nil
}

func createLogsProcessor(context.Context, ProcessorCreateSettings, config.Processor, consumer.Logs) (LogsProcessor, error) {
	return nil, nil
}
