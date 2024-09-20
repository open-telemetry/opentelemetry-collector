// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestDefaultTraces(t *testing.T) {
	cp, err := NewTraces(func(context.Context, ptrace.Traces) error { return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.Equal(t, Capabilities{MutatesData: false}, cp.Capabilities())
}

func TestNilFuncTraces(t *testing.T) {
	_, err := NewTraces(nil)
	assert.Equal(t, errNilFunc, err)
}

func TestWithCapabilitiesTraces(t *testing.T) {
	cp, err := NewTraces(
		func(context.Context, ptrace.Traces) error { return nil },
		WithCapabilities(Capabilities{MutatesData: true}))
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.Equal(t, Capabilities{MutatesData: true}, cp.Capabilities())
}

func TestConsumeTraces(t *testing.T) {
	consumeCalled := false
	cp, err := NewTraces(func(context.Context, ptrace.Traces) error { consumeCalled = true; return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
	assert.True(t, consumeCalled)
}

func TestConsumeTraces_ReturnError(t *testing.T) {
	want := errors.New("my_error")
	cp, err := NewTraces(func(context.Context, ptrace.Traces) error { return want })
	require.NoError(t, err)
	assert.Equal(t, want, cp.ConsumeTraces(context.Background(), ptrace.NewTraces()))
}
