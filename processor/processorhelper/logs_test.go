// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testLogsCfg = struct{}{}

func TestNewLogsProcessor(t *testing.T) {
	lp, err := NewLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(nil))
	require.NoError(t, err)

	assert.True(t, lp.Capabilities().MutatesData)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, lp.Shutdown(context.Background()))
}

func TestNewLogsProcessor_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.NoError(t, err)

	assert.Equal(t, want, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, lp.Shutdown(context.Background()))
	assert.False(t, lp.Capabilities().MutatesData)
}

func TestNewLogsProcessor_NilRequiredFields(t *testing.T) {
	_, err := NewLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testLogsCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)

	_, err = NewLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testLogsCfg, nil, newTestLProcessor(nil))
	assert.Equal(t, component.ErrNilNextConsumer, err)
}

func TestNewLogsProcessor_ProcessLogError(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, lp.ConsumeLogs(context.Background(), plog.NewLogs()))
}

func TestNewLogsProcessor_ProcessLogsErrSkipProcessingData(t *testing.T) {
	lp, err := NewLogsProcessor(context.Background(), processortest.NewNopCreateSettings(), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.Equal(t, nil, lp.ConsumeLogs(context.Background(), plog.NewLogs()))
}

func newTestLProcessor(retError error) ProcessLogsFunc {
	return func(_ context.Context, ld plog.Logs) (plog.Logs, error) {
		return ld, retError
	}
}
