// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package samplemigrationscraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/cmd/mdatagen/internal/samplemigrationscraper/internal/metadata"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/scraper/scrapertest"
)

// Versioned metrics should be auto enabled programmatically
// This tests that when the v1 metric is disabled in metadata.yaml
// it is auto enabled via the generated code. This ensures a user
// doesn't have to manually add the v1 metric to their config.
func TestV1MetricAutoEnabled(t *testing.T) {
	userDisabledV1 := metadata.NewDefaultMetricsBuilderConfig()
	cm := confmap.NewFromStringMap(map[string]any{
		"metrics": map[string]any{
			"system.cpu.utilization@v1": map[string]any{
				"enabled": false,
			},
		},
	})
	require.NoError(t, cm.Unmarshal(&userDisabledV1))

	tests := []struct {
		name     string
		md       metadata.MetricsBuilderConfig
		expectV1 bool
	}{
		{
			name:     "Auto enable v1 metric",
			md:       metadata.NewDefaultMetricsBuilderConfig(),
			expectV1: true,
		},
		{
			// If the user explicitly disables the v1 metric
			// it is not auto enabled
			name:     "User disables v1 metric",
			md:       userDisabledV1,
			expectV1: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set("scraper.samplemigration.EmitV1SystemConventions", true))
			t.Cleanup(func() {
				require.NoError(t, featuregate.GlobalRegistry().Set("scraper.samplemigration.EmitV1SystemConventions", false))
			})

			observedZapCore, observedLogs := observer.New(zapcore.WarnLevel)
			settings := scrapertest.NewNopSettings(metadata.Type)
			settings.Logger = zap.New(observedZapCore)

			ts := pcommon.Timestamp(1_000_001_000)

			mb := metadata.NewMetricsBuilder(tt.md, settings)

			mb.RecordSystemCPUUtilizationDataPoint(ts, 1, "cpu-val", metadata.AttributeStateIdle, 0, metadata.AttributeCPUModeSystem)

			m := mb.Emit()
			require.Equal(t, 1, m.ResourceMetrics().Len())
			ms := m.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()

			var foundV1 bool
			for i := 0; i < ms.Len(); i++ {
				metric := ms.At(i)
				if metric.Name() == "system.cpu.utilization" && metric.Type() == pmetric.MetricTypeSum {
					foundV1 = true
				}
			}
			assert.Equal(t, tt.expectV1, foundV1)

			for _, log := range observedLogs.All() {
				assert.Contains(t, log.Message, "same emitted name",
					"should log warning about same name collision")
			}
		})
	}
}
