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

package exporter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter"
)

func TestNewFactory(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExporterSettings(component.NewID(typeStr))
	factory := exporter.NewFactory(
		typeStr,
		func() exporter.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())
	_, err := factory.CreateTracesExporter(context.Background(), exporter.CreateSettings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateMetricsExporter(context.Background(), exporter.CreateSettings{}, &defaultCfg)
	assert.Error(t, err)
	_, err = factory.CreateLogsExporter(context.Background(), exporter.CreateSettings{}, &defaultCfg)
	assert.Error(t, err)
}

func TestNewFactory_WithOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewExporterSettings(component.NewID(typeStr))
	factory := exporter.NewFactory(
		typeStr,
		func() exporter.Config { return &defaultCfg },
		exporter.WithTraces(createTracesExporter, component.StabilityLevelInDevelopment),
		exporter.WithMetrics(createMetricsExporter, component.StabilityLevelAlpha),
		exporter.WithLogs(createLogsExporter, component.StabilityLevelDeprecated))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelInDevelopment, factory.TracesExporterStability())
	_, err := factory.CreateTracesExporter(context.Background(), exporter.CreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.MetricsExporterStability())
	_, err = factory.CreateMetricsExporter(context.Background(), exporter.CreateSettings{}, &defaultCfg)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsExporterStability())
	_, err = factory.CreateLogsExporter(context.Background(), exporter.CreateSettings{}, &defaultCfg)
	assert.NoError(t, err)
}

func createTracesExporter(context.Context, exporter.CreateSettings, exporter.Config) (exporter.Traces, error) {
	return nil, nil
}

func createMetricsExporter(context.Context, exporter.CreateSettings, exporter.Config) (exporter.Metrics, error) {
	return nil, nil
}

func createLogsExporter(context.Context, exporter.CreateSettings, exporter.Config) (exporter.Logs, error) {
	return nil, nil
}
