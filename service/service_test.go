// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/component/status"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/service/servicetest"
)

func TestService_GetFactory(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	assert.Nil(t, srv.host.GetFactory(component.KindReceiver, "wrongtype"))
	assert.Equal(t, factories.Receivers["nop"], srv.host.GetFactory(component.KindReceiver, "nop"))

	assert.Nil(t, srv.host.GetFactory(component.KindProcessor, "wrongtype"))
	assert.Equal(t, factories.Processors["nop"], srv.host.GetFactory(component.KindProcessor, "nop"))

	assert.Nil(t, srv.host.GetFactory(component.KindExporter, "wrongtype"))
	assert.Equal(t, factories.Exporters["nop"], srv.host.GetFactory(component.KindExporter, "nop"))

	assert.Nil(t, srv.host.GetFactory(component.KindExtension, "wrongtype"))
	assert.Equal(t, factories.Extensions["nop"], srv.host.GetFactory(component.KindExtension, "nop"))

	// Try retrieve non existing component.Kind.
	assert.Nil(t, srv.host.GetFactory(42, "nop"))
}

func TestService_GetExtensions(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	extMap := srv.host.GetExtensions()

	assert.Len(t, extMap, 1)
	assert.Contains(t, extMap, config.NewComponentID("nop"))
}

func TestService_GetExporters(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	expMap := srv.host.GetExporters()
	assert.Len(t, expMap, 3)
	assert.Len(t, expMap[config.TracesDataType], 1)
	assert.Contains(t, expMap[config.TracesDataType], config.NewComponentID("nop"))
	assert.Len(t, expMap[config.MetricsDataType], 1)
	assert.Contains(t, expMap[config.MetricsDataType], config.NewComponentID("nop"))
	assert.Len(t, expMap[config.LogsDataType], 1)
	assert.Contains(t, expMap[config.LogsDataType], config.NewComponentID("nop"))
}

func TestService_ReportStatus(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)
	host := srv.host

	var readyHandlerCalled, notReadyHandlerCalled, statusEventHandlerCalled bool

	var statusEvent *status.Event

	statusEventHandler := func(ev *status.Event) error {
		statusEvent = ev
		statusEventHandlerCalled = true
		return nil
	}

	readyHandler := func() error {
		readyHandlerCalled = true
		return nil
	}

	notReadyHandler := func() error {
		notReadyHandlerCalled = true
		return nil
	}

	unregister := host.RegisterStatusListener(
		status.WithStatusEventHandler(statusEventHandler),
		status.WithPipelineReadyHandler(readyHandler),
		status.WithPipelineNotReadyHandler(notReadyHandler),
	)

	assert.False(t, statusEventHandlerCalled)
	assert.False(t, readyHandlerCalled)
	assert.False(t, notReadyHandlerCalled)

	assert.NoError(t, srv.Start(context.Background()))
	assert.True(t, readyHandlerCalled)

	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	expectedComponentID := config.NewComponentID("nop")
	expectedError := errors.New("an error")

	host.ReportStatus(
		status.NewEvent(
			status.RecoverableError,
			status.WithComponentID(expectedComponentID),
			status.WithError(expectedError),
		),
	)

	assert.True(t, statusEventHandlerCalled)
	assert.Equal(t, expectedComponentID, statusEvent.ComponentID())
	assert.Equal(t, expectedError, statusEvent.Err())
	assert.NotNil(t, statusEvent.Timestamp)
	assert.NoError(t, unregister())
}

func TestService_ReportStatusWithBuggyHandler(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)
	host := srv.host

	var statusEventHandlerCalled bool

	statusEventHandler := func(ev *status.Event) error {
		statusEventHandlerCalled = true
		return errors.New("an error")
	}

	unregister := host.RegisterStatusListener(
		status.WithStatusEventHandler(statusEventHandler),
	)

	assert.False(t, statusEventHandlerCalled)
	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	// ReportStatus handles errors in handlers (by logging) and does not surface them back to callers
	host.ReportStatus(
		status.NewEvent(
			status.OK,
			status.WithComponentID(config.NewComponentID("nop")),
		),
	)
	assert.True(t, statusEventHandlerCalled)
	assert.NoError(t, unregister())
}

func createExampleService(t *testing.T, factories component.Factories) *service {
	// Create some factories.
	cfg, err := servicetest.LoadConfigAndValidate(filepath.Join("testdata", "otelcol-nop.yaml"), factories)
	require.NoError(t, err)

	srv, err := newService(&svcSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Telemetry: componenttest.NewNopTelemetrySettings(),
		Config:    cfg,
	})
	require.NoError(t, err)
	return srv
}
