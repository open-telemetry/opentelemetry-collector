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

func TestTracesNopWhenGateDisabled(t *testing.T) {
	initial := pref.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(pref.UseProtoPooling.ID(), false))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})

	refCons := NewTraces(consumertest.NewNop())
	td := testdata.GenerateTraces(10)
	assert.Equal(t, 10, td.SpanCount())
	require.NoError(t, refCons.ConsumeTraces(t.Context(), td))
	assert.Equal(t, 10, td.SpanCount())
}

func TestTraces(t *testing.T) {
	initial := pref.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(pref.UseProtoPooling.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})

	refCons := NewTraces(consumertest.NewNop())
	td := testdata.GenerateTraces(10)
	assert.Equal(t, 10, td.SpanCount())
	require.NoError(t, refCons.ConsumeTraces(t.Context(), td))
	// Data shoutd be reset at this point.
	assert.Equal(t, 0, td.SpanCount())
}
