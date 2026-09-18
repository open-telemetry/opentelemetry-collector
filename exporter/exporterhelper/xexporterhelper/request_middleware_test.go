// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestWrapSenderFunc_NilSafe(t *testing.T) {
	// Test that nil WrapSenderFunc returns next unchanged
	var f WrapSenderFunc
	settings := NewRequestMiddlewareSettings(
		component.NewID(component.MustNewType("test")),
		pipeline.SignalTraces,
		componenttest.NewNopTelemetrySettings(),
	)

	next := &mockSender{}
	result, err := f.WrapSender(settings, next)

	require.NoError(t, err)
	assert.Same(t, next, result, "nil WrapSenderFunc should return next unchanged")
}

func TestWrapSenderFunc_Implementation(t *testing.T) {
	// Test that WrapSenderFunc can be used to satisfy the interface
	settings := NewRequestMiddlewareSettings(
		component.NewID(component.MustNewType("test")),
		pipeline.SignalTraces,
		componenttest.NewNopTelemetrySettings(),
	)

	called := false
	f := WrapSenderFunc(func(settings RequestMiddlewareSettings, next Sender[Request]) (Sender[Request], error) {
		called = true
		return next, nil
	})

	var mw RequestMiddleware = f // Verify it satisfies the interface
	next := &mockSender{}
	result, err := mw.WrapSender(settings, next)

	require.NoError(t, err)
	assert.True(t, called, "WrapSenderFunc should be called")
	assert.Same(t, next, result)
}

func TestWrapSenderFunc_ErrorPropagation(t *testing.T) {
	settings := NewRequestMiddlewareSettings(
		component.NewID(component.MustNewType("test")),
		pipeline.SignalTraces,
		componenttest.NewNopTelemetrySettings(),
	)

	expectedErr := errors.New("wrap error")
	f := WrapSenderFunc(func(settings RequestMiddlewareSettings, next Sender[Request]) (Sender[Request], error) {
		return nil, expectedErr
	})

	next := &mockSender{}
	result, err := f.WrapSender(settings, next)

	require.Error(t, err)
	assert.Same(t, expectedErr, err)
	assert.Nil(t, result)
}

// mockSender is a minimal implementation of Sender[Request] for testing
type mockSender struct {
	component.StartFunc
	component.ShutdownFunc
}

func (m *mockSender) Send(ctx context.Context, req Request) error {
	return nil
}
