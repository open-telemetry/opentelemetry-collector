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
)

func TestNewExporterFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExporterSettings(config.NewComponentID(typeStr))
	factory := NewExporterFactory(
		typeStr,
		func() config.Exporter { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateMetricsExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateLogsExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.Error(t, err)
}

func TestNewExporterFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExporterSettings(config.NewComponentID(typeStr))
	factory := NewExporterFactory(
		typeStr,
		func() config.Exporter { return &defaultCfg },
		WithTracesExporter(createTracesExporter),
		WithMetricsExporter(createMetricsExporter),
		WithLogsExporter(createLogsExporter))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	_, err = factory.CreateMetricsExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	_, err = factory.CreateLogsExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)
}

func TestNewExporterFactory_WithStabilityLevel(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExporterSettings(config.NewComponentID(typeStr))
	factory := NewExporterFactory(
		typeStr,
		func() config.Exporter { return &defaultCfg },
		WithTracesExporterAndStabilityLevel(createTracesExporter, StabilityLevelInDevelopment),
		WithMetricsExporterAndStabilityLevel(createMetricsExporter, StabilityLevelAlpha),
		WithLogsExporterAndStabilityLevel(createLogsExporter, StabilityLevelDeprecated))

	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.EqualValues(t, StabilityLevelInDevelopment, factory.StabilityLevel(config.TracesDataType))
	_, err := factory.CreateTracesExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.EqualValues(t, StabilityLevelAlpha, factory.StabilityLevel(config.MetricsDataType))
	_, err = factory.CreateMetricsExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.EqualValues(t, StabilityLevelDeprecated, factory.StabilityLevel(config.LogsDataType))
	_, err = factory.CreateLogsExporter(context.Background(), ExporterCreateSettings{}, &defaultCfg)
	assert.NoError(t, err)
}

func createTracesExporter(context.Context, ExporterCreateSettings, config.Exporter) (TracesExporter, error) {
	return nil, nil
}

func createMetricsExporter(context.Context, ExporterCreateSettings, config.Exporter) (MetricsExporter, error) {
	return nil, nil
}

func createLogsExporter(context.Context, ExporterCreateSettings, config.Exporter) (LogsExporter, error) {
	return nil, nil
}
