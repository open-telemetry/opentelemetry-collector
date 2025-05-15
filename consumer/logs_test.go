// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/plog"
)

func TestDefaultLogs(t *testing.T) {
	cp, err := NewLogs(func(context.Context, plog.Logs) error { return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.Equal(t, Capabilities{MutatesData: false}, cp.Capabilities())
}

func TestNilFuncLogs(t *testing.T) {
	_, err := NewLogs(nil)
	assert.Equal(t, errNilFunc, err)
}

func TestWithCapabilitiesLogs(t *testing.T) {
	cp, err := NewLogs(
		func(context.Context, plog.Logs) error { return nil },
		WithCapabilities(Capabilities{MutatesData: true}))
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.Equal(t, Capabilities{MutatesData: true}, cp.Capabilities())
}

func TestConsumeLogs(t *testing.T) {
	consumeCalled := false
	cp, err := NewLogs(func(context.Context, plog.Logs) error { consumeCalled = true; return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.True(t, consumeCalled)
}

func TestConsumeLogs_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	cp, err := NewLogs(func(context.Context, plog.Logs) error { return want })
	require.NoError(t, err)
	assert.Equal(t, want, cp.ConsumeLogs(context.Background(), plog.NewLogs()))
}
