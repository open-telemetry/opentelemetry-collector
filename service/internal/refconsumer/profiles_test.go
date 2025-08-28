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

func TestProfilesNopWhenGateDisabled(t *testing.T) {
	initial := pref.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(pref.UseProtoPooling.ID(), false))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})

	refCons := NewProfiles(consumertest.NewNop())
	pd := testdata.GenerateProfiles(10)
	assert.Equal(t, 10, pd.SampleCount())
	require.NoError(t, refCons.ConsumeProfiles(t.Context(), pd))
	assert.Equal(t, 10, pd.SampleCount())
}

func TestProfiles(t *testing.T) {
	initial := pref.UseProtoPooling.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(pref.UseProtoPooling.ID(), true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(telemetry.NewPipelineTelemetryGate.ID(), initial))
	})

	refCons := NewProfiles(consumertest.NewNop())
	pd := testdata.GenerateProfiles(10)
	assert.Equal(t, 10, pd.SampleCount())
	require.NoError(t, refCons.ConsumeProfiles(t.Context(), pd))
	// Data shoupd be reset at this point.
	assert.Equal(t, 0, pd.SampleCount())
}
