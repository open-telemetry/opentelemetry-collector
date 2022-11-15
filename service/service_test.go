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
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/component/id"
	"go.opentelemetry.io/collector/component/status"
	"go.opentelemetry.io/collector/featuregate"
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

func TestServiceGetExtensions(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	extMap := srv.host.GetExtensions()

	assert.Len(t, extMap, 1)
	assert.Contains(t, extMap, id.NewID("nop"))
}

func TestServiceGetExporters(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	expMap := srv.host.GetExporters()
	assert.Len(t, expMap, 3)
	assert.Len(t, expMap[component.DataTypeTraces], 1)
	assert.Contains(t, expMap[component.DataTypeTraces], id.NewID("nop"))
	assert.Len(t, expMap[component.DataTypeMetrics], 1)
	assert.Contains(t, expMap[component.DataTypeMetrics], id.NewID("nop"))
	assert.Len(t, expMap[component.DataTypeLogs], 1)
	assert.Contains(t, expMap[component.DataTypeLogs], id.NewID("nop"))
}

// TestServiceTelemetryCleanupOnError tests that if newService errors due to an invalid config telemetry is cleaned up
// and another service with a valid config can be started right after.
func TestServiceTelemetryCleanupOnError(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	// Read invalid yaml config from file
	invalidProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-invalid.yaml")}))
	require.NoError(t, err)
	invalidCfg, err := invalidProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Read valid yaml config from file
	validProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	validCfg, err := validProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Create a service with an invalid config and expect an error
	telemetryOne := newColTelemetry(featuregate.NewRegistry())
	_, err = newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    invalidCfg,
		telemetry: telemetryOne,
	})
	require.Error(t, err)

	// Create a service with a valid config and expect no error
	telemetryTwo := newColTelemetry(featuregate.NewRegistry())
	srv, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    validCfg,
		telemetry: telemetryTwo,
	})
	require.NoError(t, err)

	// For safety ensure everything is cleaned up
	t.Cleanup(func() {
		assert.NoError(t, telemetryOne.shutdown())
		assert.NoError(t, telemetryTwo.shutdown())
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

}

// TestServiceTelemetryReusable tests that a single telemetryInitializer can be reused in multiple services
func TestServiceTelemetryReusable(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	// Read valid yaml config from file
	validProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	validCfg, err := validProvider.Get(context.Background(), factories)
	require.NoError(t, err)

	// Create a service
	telemetry := newColTelemetry(featuregate.NewRegistry())
	// For safety ensure everything is cleaned up
	t.Cleanup(func() {
		assert.NoError(t, telemetry.shutdown())
	})

	srvOne, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    validCfg,
		telemetry: telemetry,
	})
	require.NoError(t, err)

	// URL of the telemetry service metrics endpoint
	telemetryURL := fmt.Sprintf("http://%s/metrics", telemetry.server.Addr)

	// Start the service
	require.NoError(t, srvOne.Start(context.Background()))

	// check telemetry server to ensure we get a response
	var resp *http.Response

	// #nosec G107
	resp, err = http.Get(telemetryURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown the service
	require.NoError(t, srvOne.Shutdown(context.Background()))

	// Create a new service with the same telemetry
	srvTwo, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    validCfg,
		telemetry: telemetry,
	})
	require.NoError(t, err)

	// Start the new service
	require.NoError(t, srvTwo.Start(context.Background()))

	// check telemetry server to ensure we get a response
	// #nosec G107
	resp, err = http.Get(telemetryURL)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown the new service
	require.NoError(t, srvTwo.Shutdown(context.Background()))
}

func createExampleService(t *testing.T, factories component.Factories) *service {
	// Read yaml config from file
	prov, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	cfg, err := prov.Get(context.Background(), factories)
	require.NoError(t, err)

	telemetry := newColTelemetry(featuregate.NewRegistry())
	srv, err := newService(&settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		Config:    cfg,
		telemetry: telemetry,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, telemetry.shutdown())
	})
	return srv
}

func TestService_ReportStatus(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	srv := createExampleService(t, factories)
	host := srv.host

	var readyHandlerCalled, notReadyHandlerCalled, statusEventHandlerCalled bool

	var statusEvent *status.ComponentEvent

	statusEventHandler := func(ev *status.ComponentEvent) error {
		statusEvent = ev
		statusEventHandlerCalled = true
		return nil
	}

	pipelineStatusHandler := func(s status.PipelineReadiness) error {
		if s == status.PipelineReady {
			readyHandlerCalled = true
		} else {
			notReadyHandlerCalled = true
		}
		return nil
	}

	unregister := host.RegisterStatusListener(
		status.WithComponentEventHandler(statusEventHandler),
		status.WithPipelineStatusHandler(pipelineStatusHandler),
	)

	assert.False(t, statusEventHandlerCalled)
	assert.False(t, readyHandlerCalled)
	assert.False(t, notReadyHandlerCalled)

	assert.NoError(t, srv.Start(context.Background()))
	assert.True(t, readyHandlerCalled)

	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	expectedError := errors.New("an error")

	ev, err := status.NewComponentEvent(
		status.ComponentError,
		status.WithError(expectedError),
	)
	assert.NoError(t, err)
	host.ReportComponentStatus(ev)

	assert.True(t, statusEventHandlerCalled)
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

	statusEventHandler := func(ev *status.ComponentEvent) error {
		statusEventHandlerCalled = true
		return errors.New("an error")
	}

	unregister := host.RegisterStatusListener(
		status.WithComponentEventHandler(statusEventHandler),
	)

	assert.False(t, statusEventHandlerCalled)
	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	// SetComponentStatus handles errors in handlers (by logging) and does not surface them back to callers
	ev, err := status.NewComponentEvent(
		status.ComponentOK,
	)
	assert.NoError(t, err)
	host.ReportComponentStatus(ev)
	assert.True(t, statusEventHandlerCalled)
	assert.NoError(t, unregister())
}
