// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateProcessor(t *testing.T) {
	t.Run(t.Name()+"Legacy", func(t *testing.T) { testCreateProcessor(t, false) }) // Test legacy implementation
	t.Run(t.Name()+"Helper", func(t *testing.T) { testCreateProcessor(t, true) })  // Test new implementation
}

func testCreateProcessor(t *testing.T, useExporterHelper bool) {
	defer setFeatureGateForTest(t, useExporterHelper)()

	factory := NewFactory()

	cfg := factory.CreateDefaultConfig()
	creationSet := processortest.NewNopSettings(factory.Type())

	tc, _ := consumer.NewTraces(func(context.Context, ptrace.Traces) error {
		return nil
	})
	tp, err := factory.CreateTraces(context.Background(), creationSet, cfg, tc)
	assert.NotNil(t, tp)
	assert.NoError(t, err, "cannot create trace processor")
	assert.NoError(t, tp.Shutdown(context.Background()))

	mc, _ := consumer.NewMetrics(func(context.Context, pmetric.Metrics) error {
		return nil
	})
	mp, err := factory.CreateMetrics(context.Background(), creationSet, cfg, mc)
	assert.NotNil(t, mp)
	assert.NoError(t, err, "cannot create metric processor")
	assert.NoError(t, mp.Shutdown(context.Background()))

	lc, _ := consumer.NewLogs(func(context.Context, plog.Logs) error {
		return nil
	})
	lp, err := factory.CreateLogs(context.Background(), creationSet, cfg, lc)
	assert.NotNil(t, lp)
	assert.NoError(t, err, "cannot create logs processor")
	assert.NoError(t, lp.Shutdown(context.Background()))
}
