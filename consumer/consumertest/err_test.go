// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest

import (
	"context"
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
	assert.Equal(t, err, ec.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.Equal(t, err, ec.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Equal(t, err, ec.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.Equal(t, err, ec.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}
