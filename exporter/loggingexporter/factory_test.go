// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package loggingexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	me, err := factory.CreateMetricsExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, me)
}

func TestCreateTracesExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	te, err := factory.CreateTracesExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, te)
}

func TestCreateLogsExporter(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	te, err := factory.CreateLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, te)
}
