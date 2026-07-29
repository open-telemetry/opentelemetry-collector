// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplemigrationscraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplemigrationscraper/internal/metadata"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/scraper/scrapertest"
)

func TestDifferentNameVersionedMetricNoCollision(t *testing.T) {
	require.NoError(t, featuregate.GlobalRegistry().Set("scraper.samplemigration.EmitV1SystemConventions", true))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set("scraper.samplemigration.EmitV1SystemConventions", false))
	})

	cfg := metadata.NewDefaultMetricsBuilderConfig()
	cm := confmap.NewFromStringMap(map[string]any{
		"metrics": map[string]any{
			"linux.memory.available": map[string]any{
				"enabled": true,
			},
			"system.memory.linux.available@v1": map[string]any{
				"enabled": true,
			},
		},
	})
	require.NoError(t, cm.Unmarshal(&cfg))

	observedZapCore, observedLogs := observer.New(zap.WarnLevel)
	settings := scrapertest.NewNopSettings(metadata.Type)
	settings.Logger = zap.New(observedZapCore)

	ts := pcommon.Timestamp(1_000_001_000)
	mb := metadata.NewMetricsBuilder(cfg, settings)

	mb.RecordLinuxMemoryAvailableDataPoint(ts, 1)

	m := mb.Emit()
	require.Equal(t, 1, m.ResourceMetrics().Len())
	ms := m.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()

	var foundLegacy, foundV1 bool
	for i := 0; i < ms.Len(); i++ {
		switch ms.At(i).Name() {
		case "linux.memory.available":
			foundLegacy = true
		case "system.memory.linux.available":
			foundV1 = true
		}
	}
	assert.True(t, foundLegacy, "legacy metric should still be emitted for a rename")
	assert.True(t, foundV1, "v1 metric should be emitted for a rename")

	for _, log := range observedLogs.All() {
		assert.NotContains(t, log.Message, "same emitted name",
			"should not log same-name collision warning for metrics with different names")
	}
}
