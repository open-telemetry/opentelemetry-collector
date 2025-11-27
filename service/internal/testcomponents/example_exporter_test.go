// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestExampleExporter(t *testing.T) {
	exp := &ExampleExporter{}
	host := componenttest.NewNopHost()
	assert.False(t, exp.Started())
	require.NoError(t, exp.Start(context.Background(), host))
	assert.True(t, exp.Started())

	assert.Empty(t, exp.Traces)
	require.NoError(t, exp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.Len(t, exp.Traces, 1)

	assert.Empty(t, exp.Metrics)
	require.NoError(t, exp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Len(t, exp.Metrics, 1)

	assert.Empty(t, exp.Logs)
	require.NoError(t, exp.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.Len(t, exp.Logs, 1)

	assert.Empty(t, exp.Profiles)
	require.NoError(t, exp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.Len(t, exp.Profiles, 1)

	assert.False(t, exp.Stopped())
	require.NoError(t, exp.Shutdown(context.Background()))
	assert.True(t, exp.Stopped())
}
