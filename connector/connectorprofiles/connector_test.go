// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectorprofiles // import "go.opentelemetry.io/collector/connector/connectorprofiles"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/internal"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

var (
	testType = component.MustNewType("test")
	testID   = component.MustNewIDWithName("type", "name")
)

func TestNewFactoryNoOptions(t *testing.T) {
	defaultCfg := struct{}{}
	factory := connector.NewFactory(testType, func() component.Config { return &defaultCfg })
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateTracesToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, component.DataTypeTraces, componentprofiles.DataTypeProfiles))
	_, err = factory.CreateMetricsToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, component.DataTypeMetrics, componentprofiles.DataTypeProfiles))
	_, err = factory.CreateLogsToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, component.DataTypeLogs, componentprofiles.DataTypeProfiles))

	_, err = factory.CreateProfilesToTraces(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, component.DataTypeTraces))
	_, err = factory.CreateProfilesToMetrics(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, component.DataTypeMetrics))
	_, err = factory.CreateProfilesToLogs(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, component.DataTypeLogs))
}

func TestNewFactoryWithSameTypes(t *testing.T) {
	defaultCfg := struct{}{}
	factory := connector.NewFactory(testType, func() component.Config { return &defaultCfg },
		WithProfilesToProfiles(createProfilesToProfiles, component.StabilityLevelAlpha),
	)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	assert.Equal(t, component.StabilityLevelAlpha, factory.ProfilesToProfilesStability())
	_, err := factory.CreateProfilesToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)

	_, err = factory.CreateProfilesToTraces(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, component.DataTypeTraces))
	_, err = factory.CreateProfilesToMetrics(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, component.DataTypeMetrics))
	_, err = factory.CreateProfilesToLogs(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, component.DataTypeLogs))
}

func TestNewFactoryWithTranslateTypes(t *testing.T) {
	defaultCfg := struct{}{}
	factory := connector.NewFactory(testType, func() component.Config { return &defaultCfg },
		WithTracesToProfiles(createTracesToProfiles, component.StabilityLevelBeta),
		WithMetricsToProfiles(createMetricsToProfiles, component.StabilityLevelDevelopment),
		WithLogsToProfiles(createLogsToProfiles, component.StabilityLevelAlpha),

		WithProfilesToTraces(createProfilesToTraces, component.StabilityLevelBeta),
		WithProfilesToMetrics(createProfilesToMetrics, component.StabilityLevelDevelopment),
		WithProfilesToLogs(createProfilesToLogs, component.StabilityLevelAlpha),
	)
	assert.EqualValues(t, testType, factory.Type())
	assert.EqualValues(t, &defaultCfg, factory.CreateDefaultConfig())

	_, err := factory.CreateProfilesToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.Equal(t, err, internal.ErrDataTypes(testID, componentprofiles.DataTypeProfiles, componentprofiles.DataTypeProfiles))

	assert.Equal(t, component.StabilityLevelBeta, factory.TracesToProfilesStability())
	_, err = factory.CreateTracesToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDevelopment, factory.MetricsToProfilesStability())
	_, err = factory.CreateMetricsToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.LogsToProfilesStability())
	_, err = factory.CreateLogsToProfiles(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelBeta, factory.ProfilesToTracesStability())
	_, err = factory.CreateProfilesToTraces(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelDevelopment, factory.ProfilesToMetricsStability())
	_, err = factory.CreateProfilesToMetrics(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)

	assert.Equal(t, component.StabilityLevelAlpha, factory.ProfilesToLogsStability())
	_, err = factory.CreateProfilesToLogs(context.Background(), connector.Settings{ID: testID}, &defaultCfg, consumertest.NewNop())
	assert.NoError(t, err)
}

var nopInstance = &nopConnector{
	Consumer: consumertest.NewNop(),
}

// nopConnector stores consumed traces and metrics for testing purposes.
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumertest.Consumer
}

func createTracesToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Traces, error) {
	return nopInstance, nil
}

func createMetricsToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Metrics, error) {
	return nopInstance, nil
}

func createLogsToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (connector.Logs, error) {
	return nopInstance, nil
}

func createProfilesToProfiles(context.Context, connector.Settings, component.Config, consumerprofiles.Profiles) (Profiles, error) {
	return nopInstance, nil
}

func createProfilesToTraces(context.Context, connector.Settings, component.Config, consumer.Traces) (Profiles, error) {
	return nopInstance, nil
}
func createProfilesToMetrics(context.Context, connector.Settings, component.Config, consumer.Metrics) (Profiles, error) {
	return nopInstance, nil
}
func createProfilesToLogs(context.Context, connector.Settings, component.Config, consumer.Logs) (Profiles, error) {
	return nopInstance, nil
}
