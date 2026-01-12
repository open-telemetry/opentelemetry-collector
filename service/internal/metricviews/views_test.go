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

func TestDefaultViews_BatchExporterMetrics(t *testing.T) {
	tests := []struct {
		name             string
		level            configtelemetry.Level
		shouldDropBucket bool
		shouldDropBytes  bool
	}{
		{
			name:             "basic level drops bucket and bytes",
			level:            configtelemetry.LevelBasic,
			shouldDropBucket: true,
			shouldDropBytes:  true,
		},
		{
			name:             "normal level drops bucket and bytes",
			level:            configtelemetry.LevelNormal,
			shouldDropBucket: true,
			shouldDropBytes:  true,
		},
		{
			name:             "detailed level does not drop bucket or bytes",
			level:            configtelemetry.LevelDetailed,
			shouldDropBucket: false,
			shouldDropBytes:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			views := DefaultViews(tt.level)

			exporterHelperScope := "go.opentelemetry.io/collector/exporter/exporterhelper"
			bucketMetricName := "otelcol_exporter_queue_batch_send_size"
			bytesMetricName := "otelcol_exporter_queue_batch_send_size_bytes"

			var foundBucketDrop, foundBytesDrop bool
			for _, view := range views {
				if view.Selector != nil {
					if view.Selector.MeterName != nil && *view.Selector.MeterName == exporterHelperScope {
						if view.Selector.InstrumentName != nil {
							if *view.Selector.InstrumentName == bucketMetricName {
								foundBucketDrop = true
								// Verify it's a drop view
								require.NotNil(t, view.Stream)
								require.NotNil(t, view.Stream.Aggregation)
								require.NotNil(t, view.Stream.Aggregation.Drop)
							}
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

			assert.Equal(t, tt.shouldDropBucket, foundBucketDrop,
				"bucket metric drop view should be %v for level %v", tt.shouldDropBucket, tt.level)
			assert.Equal(t, tt.shouldDropBytes, foundBytesDrop,
				"bytes metric drop view should be %v for level %v", tt.shouldDropBytes, tt.level)
		})
	}
}
