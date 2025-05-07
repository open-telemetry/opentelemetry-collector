// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	creationSet := processortest.NewNopSettings(factory.Type())
	tp, err := factory.CreateTraces(context.Background(), creationSet, cfg, nil)
	assert.NotNil(t, tp)
	assert.NoError(t, err, "cannot create trace processor")
	assert.NoError(t, tp.Shutdown(context.Background()))

	mp, err := factory.CreateMetrics(context.Background(), creationSet, cfg, nil)
	assert.NotNil(t, mp)
	assert.NoError(t, err, "cannot create metric processor")
	assert.NoError(t, mp.Shutdown(context.Background()))

	lp, err := factory.CreateLogs(context.Background(), creationSet, cfg, nil)
	assert.NotNil(t, lp)
	assert.NoError(t, err, "cannot create logs processor")
	assert.NoError(t, lp.Shutdown(context.Background()))
}
