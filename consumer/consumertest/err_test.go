// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestErr(t *testing.T) {
	err := errors.New("my error")
	ec := NewErr(err)
	require.NotNil(t, ec)
	assert.NotPanics(t, ec.unexported)
	assert.Equal(t, err, ec.ConsumeLogs(t.Context(), plog.NewLogs()))
	assert.Equal(t, err, ec.ConsumeMetrics(t.Context(), pmetric.NewMetrics()))
	assert.Equal(t, err, ec.ConsumeTraces(t.Context(), ptrace.NewTraces()))
	assert.Equal(t, err, ec.ConsumeProfiles(t.Context(), pprofile.NewProfiles()))
}
