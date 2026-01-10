// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !windows && !darwin

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestRegisterProcessMetrics_UnsupportedOS(t *testing.T) {
	srv := &Service{
		telemetrySettings: componenttest.NewNopTelemetrySettings(),
	}

	// On unsupported OSes, registerProcessMetrics should log a warning and return nil.
	err := registerProcessMetrics(srv)
	require.NoError(t, err)
}

func TestRegisterProcessMetrics_LogsWarning(t *testing.T) {
	tel := componenttest.NewNopTelemetrySettings()
	srv := &Service{
		telemetrySettings: tel,
	}

	err := registerProcessMetrics(srv)
	assert.NoError(t, err)
}
