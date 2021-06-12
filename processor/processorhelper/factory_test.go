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

package processorhelper

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

var defaultCfg = config.NewProcessorSettings(config.NewID(typeStr))

func TestNewTrace(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig)
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), &defaultCfg, nil)
	assert.Error(t, err)
	_, err = factory.CreateLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), &defaultCfg, nil)
	assert.Error(t, err)
}

func TestNewMetrics_WithConstructors(t *testing.T) {
	factory := NewFactory(
		typeStr,
		defaultConfig,
		WithTraces(createTracesProcessor),
		WithMetrics(createMetricsProcessor),
		WithLogs(createLogsProcessor))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateLogsProcessor(context.Background(), componenttest.NewNopProcessorCreateSettings(), &defaultCfg, nil)
	assert.NoError(t, err)
}

func defaultConfig() config.Processor {
	return &defaultCfg
}

func createTracesProcessor(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Traces) (component.TracesProcessor, error) {
	return nil, nil
}

func createMetricsProcessor(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Metrics) (component.MetricsProcessor, error) {
	return nil, nil
}

func createLogsProcessor(context.Context, component.ProcessorCreateSettings, config.Processor, consumer.Logs) (component.LogsProcessor, error) {
	return nil, nil
}
