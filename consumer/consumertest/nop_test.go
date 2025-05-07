// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNop(t *testing.T) {
	nc := NewNop()
	require.NotNil(t, nc)
	assert.NotPanics(t, nc.unexported)
	assert.NoError(t, nc.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, nc.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.NoError(t, nc.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.NoError(t, nc.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}
