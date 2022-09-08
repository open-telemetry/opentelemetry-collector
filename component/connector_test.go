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

package component_test // import "go.opentelemetry.io/collector/component"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

func TestNewConnectorFactory_NoOptions(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewConnectorSettings(component.NewID(typeStr))
	factory := component.NewConnectorFactory(
		typeStr,
		func() component.Config { return &defaultCfg })
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrTracesToTraces)
	_, err = factory.CreateTracesToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrTracesToMetrics)
	_, err = factory.CreateTracesToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrTracesToLogs)

	_, err = factory.CreateMetricsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrMetricsToTraces)
	_, err = factory.CreateMetricsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrMetricsToMetrics)
	_, err = factory.CreateMetricsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrMetricsToLogs)

	_, err = factory.CreateLogsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrLogsToTraces)
	_, err = factory.CreateLogsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrLogsToMetrics)
	_, err = factory.CreateLogsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrLogsToLogs)
}

func TestNewConnectorFactory_WithSameTypes(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewConnectorSettings(component.NewID(typeStr))
	factory := component.NewConnectorFactory(
		typeStr,
		func() component.Config { return &defaultCfg },
		component.WithTracesToTracesConnector(createTracesToTracesConnector, component.StabilityLevelAlpha),
		component.WithMetricsToMetricsConnector(createMetricsToMetricsConnector, component.StabilityLevelBeta),
		component.WithLogsToLogsConnector(createLogsToLogsConnector, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesConnectorStability())
	_, err := factory.CreateTracesToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsConnectorStability())
	_, err = factory.CreateMetricsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsConnectorStability())
	_, err = factory.CreateLogsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	_, err = factory.CreateTracesToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrTracesToMetrics)
	_, err = factory.CreateTracesToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrTracesToLogs)

	_, err = factory.CreateMetricsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrMetricsToTraces)
	_, err = factory.CreateMetricsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrMetricsToLogs)

	_, err = factory.CreateLogsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrLogsToTraces)
	_, err = factory.CreateLogsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrLogsToMetrics)
}

func TestNewConnectorFactory_WithTranslateTypes(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewConnectorSettings(component.NewID(typeStr))
	factory := component.NewConnectorFactory(
		typeStr,
		func() component.Config { return &defaultCfg },
		component.WithTracesToMetricsConnector(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithTracesToLogsConnector(createTracesToLogsConnector, component.StabilityLevelAlpha),
		component.WithMetricsToTracesConnector(createMetricsToTracesConnector, component.StabilityLevelBeta),
		component.WithMetricsToLogsConnector(createMetricsToLogsConnector, component.StabilityLevelStable),
		component.WithLogsToTracesConnector(createLogsToTracesConnector, component.StabilityLevelDeprecated),
		component.WithLogsToMetricsConnector(createLogsToMetricsConnector, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrTracesToTraces)
	_, err = factory.CreateMetricsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrMetricsToMetrics)
	_, err = factory.CreateLogsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.Equal(t, err, component.ErrLogsToLogs)

	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsConnectorStability())
	_, err = factory.CreateTracesToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsConnectorStability())
	_, err = factory.CreateTracesToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesConnectorStability())
	_, err = factory.CreateMetricsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsConnectorStability())
	_, err = factory.CreateMetricsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesConnectorStability())
	_, err = factory.CreateLogsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsConnectorStability())
	_, err = factory.CreateLogsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func TestNewConnectorFactory_WithAllTypes(t *testing.T) {
	const typeStr = "test"
	defaultCfg := config.NewConnectorSettings(component.NewID(typeStr))
	factory := component.NewConnectorFactory(
		typeStr,
		func() component.Config { return &defaultCfg },
		component.WithTracesToTracesConnector(createTracesToTracesConnector, component.StabilityLevelAlpha),
		component.WithTracesToMetricsConnector(createTracesToMetricsConnector, component.StabilityLevelDevelopment),
		component.WithTracesToLogsConnector(createTracesToLogsConnector, component.StabilityLevelAlpha),
		component.WithMetricsToTracesConnector(createMetricsToTracesConnector, component.StabilityLevelBeta),
		component.WithMetricsToMetricsConnector(createMetricsToMetricsConnector, component.StabilityLevelBeta),
		component.WithMetricsToLogsConnector(createMetricsToLogsConnector, component.StabilityLevelStable),
		component.WithLogsToTracesConnector(createLogsToTracesConnector, component.StabilityLevelDeprecated),
		component.WithLogsToMetricsConnector(createLogsToMetricsConnector, component.StabilityLevelUnmaintained),
		component.WithLogsToLogsConnector(createLogsToLogsConnector, component.StabilityLevelUnmaintained))
	assert.EqualValues(t, typeStr, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToTracesConnectorStability())
	_, err := factory.CreateTracesToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelDevelopment, factory.TracesToMetricsConnectorStability())
	_, err = factory.CreateTracesToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelAlpha, factory.TracesToLogsConnectorStability())
	_, err = factory.CreateTracesToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToTracesConnectorStability())
	_, err = factory.CreateMetricsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelBeta, factory.MetricsToMetricsConnectorStability())
	_, err = factory.CreateMetricsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelStable, factory.MetricsToLogsConnectorStability())
	_, err = factory.CreateMetricsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDeprecated, factory.LogsToTracesConnectorStability())
	_, err = factory.CreateLogsToTracesConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToMetricsConnectorStability())
	_, err = factory.CreateLogsToMetricsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
	assert.Equal(t, component.StabilityLevelUnmaintained, factory.LogsToLogsConnectorStability())
	_, err = factory.CreateLogsToLogsConnector(context.Background(), component.ConnectorCreateSettings{}, &defaultCfg, nil)
	assert.NoError(t, err)
}

func createTracesToTracesConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Traces) (component.TracesToTracesConnector, error) {
	return nil, nil
}
func createTracesToMetricsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Metrics) (component.TracesToMetricsConnector, error) {
	return nil, nil
}
func createTracesToLogsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Logs) (component.TracesToLogsConnector, error) {
	return nil, nil
}

func createMetricsToTracesConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Traces) (component.MetricsToTracesConnector, error) {
	return nil, nil
}
func createMetricsToMetricsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Metrics) (component.MetricsToMetricsConnector, error) {
	return nil, nil
}
func createMetricsToLogsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Logs) (component.MetricsToLogsConnector, error) {
	return nil, nil
}

func createLogsToTracesConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Traces) (component.LogsToTracesConnector, error) {
	return nil, nil
}
func createLogsToMetricsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Metrics) (component.LogsToMetricsConnector, error) {
	return nil, nil
}
func createLogsToLogsConnector(context.Context, component.ConnectorCreateSettings, component.Config, consumer.Logs) (component.LogsToLogsConnector, error) {
	return nil, nil
}
