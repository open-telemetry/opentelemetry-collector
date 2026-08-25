// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectortest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewNopConnectorFactory(t *testing.T) {
	factory := NewNopFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopConfig{}, cfg)

	tracesToTraces, err := factory.CreateTracesToTraces(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToTraces.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToTraces.Shutdown(context.Background()))

	tracesToMetrics, err := factory.CreateTracesToMetrics(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToMetrics.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToMetrics.Shutdown(context.Background()))

	tracesToLogs, err := factory.CreateTracesToLogs(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToLogs.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToLogs.Shutdown(context.Background()))

	tracesToProfiles, err := factory.(xconnector.Factory).CreateTracesToProfiles(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, tracesToProfiles.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, tracesToProfiles.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, tracesToProfiles.Shutdown(context.Background()))

	metricsToTraces, err := factory.CreateMetricsToTraces(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToTraces.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToTraces.Shutdown(context.Background()))

	metricsToMetrics, err := factory.CreateMetricsToMetrics(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToMetrics.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToMetrics.Shutdown(context.Background()))

	metricsToLogs, err := factory.CreateMetricsToLogs(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToLogs.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToLogs.Shutdown(context.Background()))

	metricsToProfiles, err := factory.(xconnector.Factory).CreateMetricsToProfiles(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, metricsToProfiles.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, metricsToProfiles.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, metricsToProfiles.Shutdown(context.Background()))

	logsToTraces, err := factory.CreateLogsToTraces(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToTraces.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToTraces.Shutdown(context.Background()))

	logsToMetrics, err := factory.CreateLogsToMetrics(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToMetrics.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToMetrics.Shutdown(context.Background()))

	logsToLogs, err := factory.CreateLogsToLogs(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToLogs.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToLogs.Shutdown(context.Background()))

	logsToProfiles, err := factory.(xconnector.Factory).CreateLogsToProfiles(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, logsToProfiles.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, logsToProfiles.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, logsToProfiles.Shutdown(context.Background()))

	profilesToTraces, err := factory.(xconnector.Factory).CreateProfilesToTraces(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, profilesToTraces.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, profilesToTraces.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, profilesToTraces.Shutdown(context.Background()))

	profilesToMetrics, err := factory.(xconnector.Factory).CreateProfilesToMetrics(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, profilesToMetrics.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, profilesToMetrics.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, profilesToMetrics.Shutdown(context.Background()))

	profilesToLogs, err := factory.(xconnector.Factory).CreateProfilesToProfiles(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, profilesToLogs.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, profilesToLogs.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, profilesToLogs.Shutdown(context.Background()))

	profilesToProfiles, err := factory.(xconnector.Factory).CreateProfilesToProfiles(context.Background(), NewNopSettings(NopType), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.NoError(t, profilesToProfiles.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, profilesToProfiles.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, profilesToProfiles.Shutdown(context.Background()))
}
