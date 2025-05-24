// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package globalsignal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignal_String(t *testing.T) {
	assert.Equal(t, "traces", SignalTraces.String())
	assert.Equal(t, "metrics", SignalMetrics.String())
	assert.Equal(t, "logs", SignalLogs.String())
	assert.Equal(t, "profiles", SignalProfiles.String())
}

func TestSignal_MarshalText(t *testing.T) {
	b, err := SignalTraces.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("traces"), b)

	b, err = SignalMetrics.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("metrics"), b)

	b, err = SignalLogs.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("logs"), b)

	b, err = SignalProfiles.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte("profiles"), b)

	var s Signal
	b, err = s.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, []byte(""), b)
}

func TestSignal_UnmarshalText(t *testing.T) {
	var s Signal
	require.NoError(t, s.UnmarshalText([]byte("traces")))
	assert.Equal(t, SignalTraces, s)

	require.NoError(t, s.UnmarshalText([]byte("metrics")))
	assert.Equal(t, SignalMetrics, s)

	require.NoError(t, s.UnmarshalText([]byte("logs")))
	assert.Equal(t, SignalLogs, s)

	require.NoError(t, s.UnmarshalText([]byte("profiles")))
	assert.Equal(t, SignalProfiles, s)

	require.Error(t, s.UnmarshalText([]byte("unknown")))
}
