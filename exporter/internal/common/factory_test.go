// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func createDefaultConfig() component.Config {
	return &struct{}{}
}

func TestCreateMetricsExporter(t *testing.T) {
	me, err := CreateMetricsExporter(context.Background(), exportertest.NewNopSettings(), createDefaultConfig(), &Common{})
	require.NoError(t, err)
	assert.NotNil(t, me)
}

func TestCreateTracesExporter(t *testing.T) {
	te, err := CreateTracesExporter(context.Background(), exportertest.NewNopSettings(), createDefaultConfig(), &Common{})
	require.NoError(t, err)
	assert.NotNil(t, te)
}

func TestCreateLogsExporter(t *testing.T) {
	te, err := CreateLogsExporter(context.Background(), exportertest.NewNopSettings(), createDefaultConfig(), &Common{})
	require.NoError(t, err)
	assert.NotNil(t, te)
}
