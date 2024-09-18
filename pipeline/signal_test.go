// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Signal_String(t *testing.T) {
	assert.Equal(t, "traces", SignalTraces.String())
	assert.Equal(t, "metrics", SignalMetrics.String())
	assert.Equal(t, "logs", SignalLogs.String())
}

func Test_Signal_MarshalText(t *testing.T) {
	val, err := SignalTraces.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, []byte("traces"), val)

	val, err = SignalMetrics.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, []byte("metrics"), val)

	val, err = SignalLogs.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, []byte("logs"), val)
}
