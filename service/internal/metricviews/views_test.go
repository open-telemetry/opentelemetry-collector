// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

func TestDefaultViews(t *testing.T) {
	for _, tt := range []struct {
		name  string
		level configtelemetry.Level

		wantViewsCount int
	}{
		{
			name:           "None",
			level:          configtelemetry.LevelNone,
			wantViewsCount: 17,
		},
		{
			name:           "Basic",
			level:          configtelemetry.LevelBasic,
			wantViewsCount: 17,
		},
		{
			name:           "Normal",
			level:          configtelemetry.LevelNormal,
			wantViewsCount: 14,
		},
		{
			name:           "Detailed",
			level:          configtelemetry.LevelDetailed,
			wantViewsCount: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			views := DefaultViews(tt.level)
			assert.Len(t, views, tt.wantViewsCount)
		})
	}
}

func TestDefaultViewsFiltersSendFailedAttributes(t *testing.T) {
	tests := []struct {
		name                         string
		level                        configtelemetry.Level
		expectSendFailedFilteredView bool
	}{
		{
			name:                         "basic level filters send_failed attributes",
			level:                        configtelemetry.LevelBasic,
			expectSendFailedFilteredView: true,
		},
		{
			name:                         "normal level filters send_failed attributes",
			level:                        configtelemetry.LevelNormal,
			expectSendFailedFilteredView: true,
		},
		{
			name:                         "detailed level does not filter send_failed attributes",
			level:                        configtelemetry.LevelDetailed,
			expectSendFailedFilteredView: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			views := DefaultViews(tt.level)

			foundSendFailedView := false
			for _, view := range views {
				if view.Selector == nil ||
					view.Selector.InstrumentName == nil ||
					*view.Selector.InstrumentName != "otelcol_exporter_send_failed_*" {
					continue
				}
				foundSendFailedView = true
				require.NotNil(t, view.Stream, "send_failed view should have a stream")
				require.NotNil(t, view.Stream.AttributeKeys, "send_failed view should have attribute keys")
				require.Equal(t, []string{"exporter"}, view.Stream.AttributeKeys.Included,
					"send_failed view should only include 'exporter' attribute")
				break
			}

			if tt.expectSendFailedFilteredView {
				assert.True(t, foundSendFailedView,
					"Expected to find send_failed attribute filtering view at level %s", tt.level)
			} else {
				assert.False(t, foundSendFailedView,
					"Did not expect to find send_failed attribute filtering view at level %s", tt.level)
			}
		})
	}
}

func TestDefaultViews_BatchExporterMetrics(t *testing.T) {
	tests := []struct {
		name            string
		level           configtelemetry.Level
		shouldDropBytes bool
	}{
		{
			name:            "basic level drops bytes",
			level:           configtelemetry.LevelBasic,
			shouldDropBytes: true,
		},
		{
			name:            "normal level drops bytes",
			level:           configtelemetry.LevelNormal,
			shouldDropBytes: true,
		},
		{
			name:            "detailed level does not drop bytes",
			level:           configtelemetry.LevelDetailed,
			shouldDropBytes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			views := DefaultViews(tt.level)

			exporterHelperScope := "go.opentelemetry.io/collector/exporter/exporterhelper"
			bytesMetricName := "otelcol_exporter_queue_batch_send_size_bytes"

			var foundBytesDrop bool
			for _, view := range views {
				if view.Selector != nil {
					if view.Selector.MeterName != nil && *view.Selector.MeterName == exporterHelperScope {
						if view.Selector.InstrumentName != nil {
							if *view.Selector.InstrumentName == bytesMetricName {
								foundBytesDrop = true
								// Verify it's a drop view
								require.NotNil(t, view.Stream)
								require.NotNil(t, view.Stream.Aggregation)
								require.NotNil(t, view.Stream.Aggregation.Drop)
							}
						}
					}
				}
			}

			assert.Equal(t, tt.shouldDropBytes, foundBytesDrop,
				"bytes metric drop view should be %v for level %v", tt.shouldDropBytes, tt.level)
		})
	}
}
