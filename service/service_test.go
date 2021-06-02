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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestService_GetFactory(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	srv := createExampleService(t)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	factory := srv.GetFactory(component.KindReceiver, "nop")
	assert.EqualValues(t, factories.Receivers["nop"], factory)
	factory = srv.GetFactory(component.KindReceiver, "wrongtype")
	assert.EqualValues(t, nil, factory)

	factory = srv.GetFactory(component.KindProcessor, "nop")
	assert.EqualValues(t, factories.Processors["nop"], factory)
	factory = srv.GetFactory(component.KindProcessor, "wrongtype")
	assert.EqualValues(t, nil, factory)

	factory = srv.GetFactory(component.KindExporter, "nop")
	assert.EqualValues(t, factories.Exporters["nop"], factory)
	factory = srv.GetFactory(component.KindExporter, "wrongtype")
	assert.EqualValues(t, nil, factory)

	factory = srv.GetFactory(component.KindExtension, "nop")
	assert.EqualValues(t, factories.Extensions["nop"], factory)
	factory = srv.GetFactory(component.KindExtension, "wrongtype")
	assert.EqualValues(t, nil, factory)

	// Try retrieve non existing component.Kind.
	factory = srv.GetFactory(42, "nop")
	assert.EqualValues(t, nil, factory)
}

func TestService_GetExtensions(t *testing.T) {
	srv := createExampleService(t)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	extMap := srv.GetExtensions()

	assert.Len(t, extMap, 1)
	assert.Contains(t, extMap, config.NewID("nop"))
}

func TestService_GetExporters(t *testing.T) {
	srv := createExampleService(t)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	expMap := srv.GetExporters()
	assert.Len(t, expMap, 3)
	assert.Len(t, expMap[config.TracesDataType], 1)
	assert.Contains(t, expMap[config.TracesDataType], config.NewID("nop"))
	assert.Len(t, expMap[config.MetricsDataType], 1)
	assert.Contains(t, expMap[config.MetricsDataType], config.NewID("nop"))
	assert.Len(t, expMap[config.LogsDataType], 1)
	assert.Contains(t, expMap[config.LogsDataType], config.NewID("nop"))
}

func createExampleService(t *testing.T) *service {
	// Create some factories.
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "otelcol-nop.yaml"), factories)
	require.NoError(t, err)

	srv, err := newService(&svcSettings{
		BuildInfo: component.DefaultBuildInfo(),
		Factories: factories,
		Logger:    zap.NewNop(),
		Config:    cfg,
	})
	require.NoError(t, err)
	return srv
}
