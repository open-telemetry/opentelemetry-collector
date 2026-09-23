// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelpertest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"
	"go.opentelemetry.io/collector/pipeline"
)

func TestNewNop(t *testing.T) {
	nop := NewNop()

	// Must satisfy extension.Extension lifecycle methods.
	require.NoError(t, nop.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, nop.Shutdown(context.Background()))

	// Must satisfy RequestMiddleware: WrapSender returns next unchanged.
	mw, ok := nop.(xexporterhelper.RequestMiddleware)
	require.True(t, ok, "NewNop() must implement xexporterhelper.RequestMiddleware")

	settings := xexporterhelper.NewRequestMiddlewareSettings(
		component.NewID(component.MustNewType("test")),
		pipeline.SignalTraces,
		componenttest.NewNopTelemetrySettings(),
	)
	next := &mockSender{}
	result, err := mw.WrapSender(settings, next)
	require.NoError(t, err)
	assert.Same(t, next, result, "NewNop WrapSender should return next unchanged")
}

func TestNewErr(t *testing.T) {
	expectedErr := errors.New("test error")
	errExt := NewErr(expectedErr)

	// Lifecycle methods remain no-ops.
	require.NoError(t, errExt.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, errExt.Shutdown(context.Background()))

	// WrapSender must return the error.
	mw, ok := errExt.(xexporterhelper.RequestMiddleware)
	require.True(t, ok, "NewErr() must implement xexporterhelper.RequestMiddleware")

	settings := xexporterhelper.NewRequestMiddlewareSettings(
		component.NewID(component.MustNewType("test")),
		pipeline.SignalTraces,
		componenttest.NewNopTelemetrySettings(),
	)
	result, err := mw.WrapSender(settings, &mockSender{})
	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
}

// mockSender is a minimal Sender[Request] for use in tests.
type mockSender struct {
	component.StartFunc
	component.ShutdownFunc
}

func (m *mockSender) Send(_ context.Context, _ xexporterhelper.Request) error {
	return nil
}
