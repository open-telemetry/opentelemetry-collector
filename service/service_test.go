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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/service/featuregate"
	"go.opentelemetry.io/collector/service/internal/configunmarshaler"
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
	assert.Contains(t, extMap, config.NewComponentID("nop"))
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
	assert.Len(t, expMap[config.TracesDataType], 1)
	assert.Contains(t, expMap[config.TracesDataType], config.NewComponentID("nop"))
	assert.Len(t, expMap[config.MetricsDataType], 1)
	assert.Contains(t, expMap[config.MetricsDataType], config.NewComponentID("nop"))
	assert.Len(t, expMap[config.LogsDataType], 1)
	assert.Contains(t, expMap[config.LogsDataType], config.NewComponentID("nop"))
}

func createExampleService(t *testing.T, factories component.Factories) *service {
	// Read yaml config from file
	conf, err := confmaptest.LoadConf(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)
	cfg, err := configunmarshaler.New().Unmarshal(conf, factories)
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
