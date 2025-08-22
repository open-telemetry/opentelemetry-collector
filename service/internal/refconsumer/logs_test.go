// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package refconsumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/telemetry"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func TestLogsNopWhenGateDisabled(t *testing.T) {
	initial := pref.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(pref.UseProtoPooling.ID(), false))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})

	refCons := NewLogs(consumertest.NewNop())
	ld := testdata.GenerateLogs(10)
	assert.Equal(t, 10, ld.LogRecordCount())
	require.NoError(t, refCons.ConsumeLogs(t.Context(), ld))
	assert.Equal(t, 10, ld.LogRecordCount())
}

func TestLogs(t *testing.T) {
	initial := pref.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(pref.UseProtoPooling.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})

	refCons := NewLogs(consumertest.NewNop())
	ld := testdata.GenerateLogs(10)
	assert.Equal(t, 10, ld.LogRecordCount())
	require.NoError(t, refCons.ConsumeLogs(t.Context(), ld))
	// Data should be reset at this point.
	assert.Equal(t, 0, ld.LogRecordCount())
}
