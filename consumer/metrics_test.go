// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestDefaultMetrics(t *testing.T) {
	cp, err := NewMetrics(func(context.Context, pmetric.Metrics) error { return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Equal(t, Capabilities{MutatesData: false}, cp.Capabilities())
}

func TestNilFuncMetrics(t *testing.T) {
	_, err := NewMetrics(nil)
	assert.Equal(t, errNilFunc, err)
}

func TestWithCapabilitiesMetrics(t *testing.T) {
	cp, err := NewMetrics(
		func(context.Context, pmetric.Metrics) error { return nil },
		WithCapabilities(Capabilities{MutatesData: true}))
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.Equal(t, Capabilities{MutatesData: true}, cp.Capabilities())
}

func TestConsumeMetrics(t *testing.T) {
	consumeCalled := false
	cp, err := NewMetrics(func(context.Context, pmetric.Metrics) error { consumeCalled = true; return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
	assert.True(t, consumeCalled)
}

func TestConsumeMetrics_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	cp, err := NewMetrics(func(context.Context, pmetric.Metrics) error { return want })
	require.NoError(t, err)
	assert.Equal(t, want, cp.ConsumeMetrics(context.Background(), pmetric.NewMetrics()))
}
