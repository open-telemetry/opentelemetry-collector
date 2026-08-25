// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

var wrongType = component.MustNewType("wrong")

func TestService_Host_GetFactory(t *testing.T) {
	set := newNopSettings()
	srv, err := New(context.Background(), set, newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	assert.Nil(t, srv.host.GetFactory(component.KindReceiver, wrongType))
	assert.Equal(t, srv.host.Receivers.Factory(nopType), srv.host.GetFactory(component.KindReceiver, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindProcessor, wrongType))
	assert.Equal(t, srv.host.Processors.Factory(nopType), srv.host.GetFactory(component.KindProcessor, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindExporter, wrongType))
	assert.Equal(t, srv.host.Exporters.Factory(nopType), srv.host.GetFactory(component.KindExporter, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindConnector, wrongType))
	assert.Equal(t, srv.host.Connectors.Factory(nopType), srv.host.GetFactory(component.KindConnector, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindExtension, wrongType))
	assert.Equal(t, srv.host.Extensions.Factory(nopType), srv.host.GetFactory(component.KindExtension, nopType))

	// Try retrieve non existing component.Kind.
	assert.Nil(t, srv.host.GetFactory(component.Kind{}, nopType))
}

func TestService_Host_GetExtensions(t *testing.T) {
	srv, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	extMap := srv.host.GetExtensions()

	assert.Len(t, extMap, 1)
	assert.Contains(t, extMap, component.NewID(nopType))
}

func TestService_Host_GetExporters(t *testing.T) {
	srv, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	//nolint:staticcheck
	expMap := srv.host.GetExporters()

	v, ok := expMap[pipeline.SignalTraces]
	assert.True(t, ok)
	assert.NotNil(t, v)

	assert.Len(t, expMap, 4)
	assert.Len(t, expMap[pipeline.SignalTraces], 1)
	assert.Contains(t, expMap[pipeline.SignalTraces], component.NewID(nopType))
	assert.Len(t, expMap[pipeline.SignalMetrics], 1)
	assert.Contains(t, expMap[pipeline.SignalMetrics], component.NewID(nopType))
	assert.Len(t, expMap[pipeline.SignalLogs], 1)
	assert.Contains(t, expMap[pipeline.SignalLogs], component.NewID(nopType))
	assert.Len(t, expMap[xpipeline.SignalProfiles], 1)
	assert.Contains(t, expMap[xpipeline.SignalProfiles], component.NewID(nopType))
}

func TestService_Host_FatalError(t *testing.T) {
	set := newNopSettings()
	set.AsyncErrorChannel = make(chan error)

	srv, err := New(context.Background(), set, newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	go func() {
		ev := componentstatus.NewFatalErrorEvent(assert.AnError)
		srv.host.NotifyComponentStatusChange(&componentstatus.InstanceID{}, ev)
	}()

	err = <-srv.host.AsyncErrorChannel

	require.ErrorIs(t, err, assert.AnError)
}
